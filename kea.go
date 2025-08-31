package kea

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"slices"
	"strings"

	"github.com/coredns/coredns/plugin"
	clog "github.com/coredns/coredns/plugin/pkg/log"
	"github.com/coredns/coredns/request"

	"github.com/miekg/dns"
)

var log = clog.NewWithPlugin("kea")

const KEA_IPV4_SERVICE_NAME = "dhcp4"
const KEA_IPV6_SERVICE_NAME = "dhcp6"

const KEA_LIST_LEASES_BY_HOSTNAME_TEMPLATE = `{
  "command": "lease4-get-by-hostname",
  "arguments": {
    "hostname": "%s"
  },
  "service": [
    %s
  ]
}`
const KEA_LIST_RESERVATIONS_BY_HOSTNAME_TEMPLATE = `{
  "command": "reservation-get-by-hostname",
  "arguments": {
    "hostname": "%s"
  },
  "service": [
    %s
  ]
}`

type Kea struct {
	ControlAgent             string
	Networks                 []string
	ExtractHostname          string
	ControlAgentLeases       string
	ControlAgentReservations string
	Insecure                 string
	Next                     plugin.Handler
	UseIPv4                  string
	UseIPv6                  string
	DHCP4ConfPath            string
	DHCP6ConfPath            string
	DHCP4Conf                KeaDHCP4Conf
	DHCP6Conf                KeaDHCP6Conf
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

func (k Kea) serviceNames() string {
	services := []string{}
	if k.UseIPv4 == "true" {
		services = append(services, fmt.Sprintf(`"%s"`, KEA_IPV4_SERVICE_NAME))
	}
	if k.UseIPv6 == "true" {
		services = append(services, fmt.Sprintf(`"%s"`, KEA_IPV6_SERVICE_NAME))
	}
	return strings.Join(services, ", ")
}

func (k Kea) MakeControlAgentRequest(requestBody string) (responseBody []byte, err error) {
	requestBodyBytes := bytes.NewBufferString(requestBody)

	resp, err := k.httpClient().Post(k.ControlAgent, "application/json", requestBodyBytes)

	if err != nil {
		return
	}
	if resp.StatusCode != 200 {
		err = errors.New(resp.Status)
		return
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return
	}

	return body, nil
}

func (k Kea) ControlAgentGetIPsForLease(deviceName string) (ips []net.IP, err error) {
	responseBody, err := k.MakeControlAgentRequest(
		fmt.Sprintf(KEA_LIST_LEASES_BY_HOSTNAME_TEMPLATE,
			deviceName,
			k.serviceNames()))
	if err != nil {
		return
	}

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
		case Error:
			log.Warning("Kea error: " + leaseRecord.Text)
		}
	}

	if len(k.Networks) > 0 {
		ips, err = FilterIPsInCIDRs(*ipResult, k.Networks)
		if err != nil {
			return nil, err
		}
	} else {
		ips = *ipResult
	}

	return
}

func (k Kea) ControlAgentGetIPsForReservation(deviceName string) (ips []net.IP, err error) {
	responseBody, err := k.MakeControlAgentRequest(
		fmt.Sprintf(
			KEA_LIST_RESERVATIONS_BY_HOSTNAME_TEMPLATE,
			deviceName,
			k.serviceNames()))

	if err != nil {
		return
	}

	var reservationRecords KeaReservationRecords
	err = json.Unmarshal(responseBody, &reservationRecords)
	if err != nil {
		return
	}

	ipResult := new([]net.IP)

	for _, reservationRecord := range reservationRecords {
		switch reservationRecord.Result {
		case Success:
			for _, lease := range reservationRecord.Arguments.Leases {
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
		case Error:
			log.Warning("Kea reservation error: " + reservationRecord.Text)
		}
	}

	ips, err = FilterIPsInCIDRs(ips, k.Networks)
	if err != nil {
		return nil, err
	}

	return
}

func CompareCIDRs(subnet1 string, subnet2 string) bool {
	_, net1, err := net.ParseCIDR(subnet1)
	if err != nil {
		return false
	}
	_, net2, err := net.ParseCIDR(subnet2)
	if err != nil {
		return false
	}
	return net1.String() == net2.String()
}

func (k Kea) GetIPsForHostname(deviceName string) (ips []net.IP, err error) {
	if k.ControlAgentLeases == "true" {
		leases, err := k.ControlAgentGetIPsForLease(deviceName)
		if err != nil {
			return nil, err
		}
		if len(leases) > 0 {
			ips = append(ips, leases...)
		}
	}

	if k.ControlAgentReservations == "true" {
		reservations, err := k.ControlAgentGetIPsForReservation(deviceName)
		if err != nil {
			return nil, err
		}
		if len(reservations) > 0 {
			ips = append(ips, reservations...)
		}
	}

	if k.DHCP4ConfPath != "" {
		for _, subnet := range k.DHCP4Conf.Dhcp4.Subnet4 {
			if slices.IndexFunc(k.Networks, func(n string) bool {
				return CompareCIDRs(n, subnet.Subnet)
			}) != -1 || len(k.Networks) == 0 {
				for _, reservation := range subnet.Reservations {
					if reservation.Hostname == deviceName {
						ip := net.ParseIP(reservation.IpAddress)
						if !ip.IsLoopback() {
							ips = append(ips, ip)
						}
					}
				}
			}
		}
	}

	if k.DHCP6ConfPath != "" {
		for _, subnet := range k.DHCP6Conf.Dhcp6.Subnet6 {
			if slices.IndexFunc(k.Networks, func(n string) bool {
				return CompareCIDRs(n, subnet.Subnet)
			}) != -1 || len(k.Networks) == 0 {
				for _, reservation := range subnet.Reservations {
					if reservation.Hostname == deviceName {
						for _, ipString := range reservation.IpAddresses {
							ip := net.ParseIP(ipString)
							if !ip.IsLoopback() {
								ips = append(ips, ip)
							}
						}
					}
				}
			}
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
		if ipInAnyNetwork(ip, networks) {
			filteredIps = append(filteredIps, ip)
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

type KeaDHCP4Conf struct {
	Dhcp4 struct {
		Subnet4 []struct {
			Subnet       string `json:"subnet"`
			Reservations []struct {
				IpAddress string `json:"ip-address"`
				HwAddress string `json:"hw-address"`
				Hostname  string `json:"hostname,omitempty"`
			} `json:"reservations,omitempty"`
		} `json:"subnet4"`
	} `json:"Dhcp4"`
}

type KeaDHCP6Conf struct {
	Dhcp6 struct {
		Subnet6 []struct {
			Subnet       string `json:"subnet"`
			Reservations []struct {
				IpAddresses []string `json:"ip-addresses"`
				HwAddress   string   `json:"hw-address"`
				Hostname    string   `json:"hostname,omitempty"`
			} `json:"reservations,omitempty"`
		} `json:"subnet6"`
	} `json:"Dhcp6"`
}
