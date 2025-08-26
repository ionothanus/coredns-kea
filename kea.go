package kea

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"

	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
)

var log = clog.NewWithPlugin("kea")

const KEA_LIST_LEASES_BY_HOSTNAME_TEMPLATE = `{
  "command": "lease4-get-by-hostname",
  "arguments": {
    "hostname": "%s"
  },
  "service": [
    "dhcp4",
	"dhcp6"
  ]
}`
const KEA_LIST_RESERVATIONS_BY_HOSTNAME_TEMPLATE = `{
  "command": "reservation-get-by-hostname",
  "arguments": {
    "hostname": "%s"
  },
  "service": [
    "dhcp4",
	"dhcp6"
  ]
}`

type Kea struct {
	ControlAgent    string
	Networks        []string
	ExtractHostname string
	UseLeases       string
	UseReservations string
	Insecure        string
	Next            plugin.Handler
}

func (k Kea) httpClient() *http.Client {
	if k.Insecure == "true" {
		transCfg := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		return &http.Client{Transport: transCfg}
	}
	return &http.Client{}
}

// ServeDNS implements the plugin.Handler interface. This method gets called when example is used
// in a Server.
func (k Kea) ServeDNS(ctx context.Context, w dns.ResponseWriter, r *dns.Msg) (int, error) {
	state := request.Request{W: w, Req: r}

	if state.QType() != dns.TypeA && state.QType() != dns.TypeAAAA {
		return plugin.NextOrFailure(k.Name(), k.Next, ctx, w, r)
	}

	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true
	m.RecursionAvailable = false

	nameLookup := state.QName()

	if k.ExtractHostname == "true" {
		nameLookup = strings.SplitN(nameLookup, ".", 2)[0]
	}

	ips, err := k.GetIPsForHostname(nameLookup)

	if err != nil {
		return plugin.NextOrFailure(k.Name(), k.Next, ctx, w, r)
	}

	found := false

	for _, ip := range ips {
		if ip.To4() == nil && state.QType() == dns.TypeAAAA {
			found = true
			m.Answer = append(m.Answer, &dns.AAAA{
				Hdr: dns.RR_Header{
					Name:   state.QName(),
					Rrtype: dns.TypeAAAA,
					Class:  dns.ClassINET,
					Ttl:    60,
				},
				AAAA: ip,
			})
		} else if ip.To4() != nil && state.QType() == dns.TypeA {
			found = true
			m.Answer = append(m.Answer, &dns.A{
				Hdr: dns.RR_Header{
					Name:   state.QName(),
					Rrtype: dns.TypeA,
					Class:  dns.ClassINET,
					Ttl:    60,
				},
				A: ip,
			})
		}
	}

	if !found {
		return plugin.NextOrFailure(k.Name(), k.Next, ctx, w, r)
	}
	err = w.WriteMsg(m)
	return 0, err
}

func (k Kea) Name() string { return "kea" }

func (k Kea) MakeControlAgentRequest(requestBody string) (responseBody []byte, err error) {
	requestBodyBytes := bytes.NewBufferString(requestBody)

	req, err := http.NewRequest(http.MethodPost, k.ControlAgent, requestBodyBytes)
	resp, err := k.httpClient().Do(req)
	if err != nil {
		return
	}
	if resp.StatusCode != 200 {
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	return body, nil
}

func (k Kea) GetIPsForLease(deviceName string) (ips []net.IP, err error) {
	responseBody, err := k.MakeControlAgentRequest(fmt.Sprintf(KEA_LIST_LEASES_BY_HOSTNAME_TEMPLATE, deviceName))
	var leaseRecords KeaLeaseRecords
	err = json.Unmarshal(responseBody, &leaseRecords)
	if err != nil {
		return
	}

	ipResult := new([]net.IP)

	for _, leaseRecord := range leaseRecords {
		switch leaseRecord.Result {
		case Success:
			for _, lease := range leaseRecord.Arguments.Leases {
				ip := net.ParseIP(lease.IPAddress)
				if !ip.IsLoopback() {
					*ipResult = append(*ipResult, ip)
				}
			}
		}
	}

	ips, err = FilterIPsInCIDRs(ips, k.Networks)
	if err != nil {
		return nil, err
	}

	return
}

func (k Kea) GetIPsForReservation(deviceName string) (ips []net.IP, err error) {
	responseBody, err := k.MakeControlAgentRequest(fmt.Sprintf(KEA_LIST_RESERVATIONS_BY_HOSTNAME_TEMPLATE, deviceName))
	var leaseRecords KeaReservationRecords
	err = json.Unmarshal(responseBody, &leaseRecords)
	if err != nil {
		return
	}

	ipResult := new([]net.IP)

	for _, leaseRecord := range leaseRecords {
		switch leaseRecord.Result {
		case Success:
			for _, lease := range leaseRecord.Arguments.Leases {
				ip := net.ParseIP(lease.IPAddress)
				if !ip.IsLoopback() {
					*ipResult = append(*ipResult, ip)
				}

				for _, innerIPString := range lease.IPAddresses {
					innerIp := net.ParseIP(innerIPString)
					if !innerIp.IsLoopback() {
						*ipResult = append(*ipResult, ip)
					}
				}

			}
		}
	}

	ips, err = FilterIPsInCIDRs(ips, k.Networks)
	if err != nil {
		return nil, err
	}

	return
}

func (k Kea) GetIPsForHostname(deviceName string) (ips []net.IP, err error) {
	if k.UseLeases == "true" {
		leases, err := k.GetIPsForLease(deviceName)
		if err != nil {
			return nil, err
		}
		if len(ips) > 0 {
			ips = append(ips, leases...)
		}
	}

	if k.UseReservations == "true" {
		reservations, err := k.GetIPsForReservation(deviceName)
		if err != nil {
			return nil, err
		}
		if len(ips) > 0 {
			ips = append(ips, reservations...)
		}
	}

	return ips, nil
}

func FilterIPsInCIDRs(ips []net.IP, cidrs []string) (filteredIps []net.IP, err error) {
	var networks []*net.IPNet
	for _, cidr := range cidrs {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, err
		}
		networks = append(networks, network)
	}

	for _, ip := range ips {
		var filtered []net.IP

		if ipInAnyNetwork(ip, networks) {
			filtered = append(filtered, ip)
		}
	}

	return
}

func ipInAnyNetwork(ip net.IP, networks []*net.IPNet) bool {
	for _, network := range networks {
		if network.Contains(ip) {
			return true
		}
	}
	return false
}

// ResponsePrinter wrap a dns.ResponseWriter and will write example to standard output when WriteMsg is called.
type ResponsePrinter struct {
	dns.ResponseWriter
}

// NewResponsePrinter returns ResponseWriter.
func NewResponsePrinter(w dns.ResponseWriter) *ResponsePrinter {
	return &ResponsePrinter{ResponseWriter: w}
}

// WriteMsg calls the underlying ResponseWriter's WriteMsg method and prints "example" to standard output.
func (r *ResponsePrinter) WriteMsg(res *dns.Msg) error {
	log.Info("example")
	return r.ResponseWriter.WriteMsg(res)
}

type KeaResultCode int

const (
	Success KeaResultCode = iota
	Error
	Unsupported
	NoContent
)

type KeaLeaseRecords []struct {
	Arguments struct {
		Leases []struct {
			Cltt      int    `json:"cltt"`
			FqdnFwd   bool   `json:"fqdn-fwd"`
			FqdnRev   bool   `json:"fqdn-rev"`
			Hostname  string `json:"hostname"`
			HwAddress string `json:"hw-address"`
			IPAddress string `json:"ip-address"`
			State     int    `json:"state"`
			SubnetID  int    `json:"subnet-id"`
			ValidLft  int    `json:"valid-lft"`
		} `json:"leases"`
	} `json:"arguments,omitempty"`
	Result KeaResultCode `json:"result"`
	Text   string        `json:"text"`
}

// TODO: this is a guess from the docs. I don't yet have a Kea instance
// running which supports reservation records in the control agent interface.
// (It was a paid feature until 2.7.7/3.0.)
type KeaReservationRecords []struct {
	Arguments struct {
		Leases []struct {
			Hostname    string   `json:"hostname"`
			HwAddress   string   `json:"hw-address"`
			IPAddress   string   `json:"ip-address,omitempty"`
			IPAddresses []string `json:"ip-addresses,omitempty"`
			SubnetID    int      `json:"subnet-id"`
		} `json:"leases"`
	} `json:"arguments,omitempty"`
	Result KeaResultCode `json:"result"`
	Text   string        `json:"text"`
}
