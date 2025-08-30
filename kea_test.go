package kea

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/joho/godotenv"
)

func TestMain(m *testing.M) {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("err loading: %v", err)
	}

	controlAgent = os.Getenv("CDNS_KEA_CONTROL_AGENT")
	insecure = os.Getenv("CDNS_KEA_INSECURE")
	leaseHostname = os.Getenv("CDNS_KEA_LEASE_HOSTNAME")
	leaseIPv4 = os.Getenv("CDNS_KEA_LEASE_IP_V4")
	reservationHostname = os.Getenv("CDNS_KEA_RESERVATION_HOSTNAME")
	reservationIPv4 = os.Getenv("CDNS_KEA_RESERVATION_IP_V4")
	awaitedAnswersIPv4 = os.Getenv("CDNS_KEA_AWAITED_ANSWERS_IP_V4")
	leaseIPv6 = os.Getenv("CDNS_KEA_LEASE_IP_V6")
	reservationIPv6 = os.Getenv("CDNS_KEA_RESERVATION_IP_V6")
	awaitedAnswersIPv6 = os.Getenv("CDNS_KEA_AWAITED_ANSWERS_IP_V6")
	includeReservationTests = os.Getenv("CDNS_KEA_INCLUDE_RESERVATION_TESTS")
	includedNetworks = strings.Split(os.Getenv("CDNS_KEA_INCLUDED_NETWORKS"), " ")
	excludedNetworks = strings.Split(os.Getenv("CDNS_KEA_EXCLUDED_NETWORKS"), " ")
	useIPv4 = os.Getenv("CDNS_KEA_USE_IPV4")
	useIPv6 = os.Getenv("CDNS_KEA_USE_IPV6")

	os.Exit(m.Run())
}

var (
	controlAgent            string
	insecure                string
	leaseHostname           string
	leaseIPv4               string
	reservationHostname     string
	reservationIPv4         string
	awaitedAnswersIPv4      string
	leaseIPv6               string
	reservationIPv6         string
	awaitedAnswersIPv6      string
	includeReservationTests string
	includedNetworks        []string
	excludedNetworks        []string
	useIPv4                 string
	useIPv6                 string
)

func MakeTestKeaControlAgent() Kea {
	return Kea{
		ControlAgent:             controlAgent,
		Insecure:                 insecure,
		ControlAgentLeases:       "true",
		ControlAgentReservations: includeReservationTests,
		UseIPv4:                  useIPv4,
		UseIPv6:                  useIPv6,
	}
}

func MakeTestKeaConfFiles() Kea {
	dhcp4ConfPath := "./resources/kea-dhcp4.conf"
	dhcp6ConfPath := "./resources/kea-dhcp6.conf"

	var dhcp4Conf KeaDHCP4Conf
	var dhcp6Conf KeaDHCP6Conf
	data, err := os.ReadFile(dhcp4ConfPath)
	if err != nil {
		log.Fatalf("err loading dhcp4 conf: %v", err)
	}
	err = json.Unmarshal(data, &dhcp4Conf)
	if err != nil {
		log.Fatalf("err parsing dhcp4 conf: %v", err)
	}
	data, err = os.ReadFile(dhcp6ConfPath)
	if err != nil {
		log.Fatalf("err loading dhcp6 conf: %v", err)
	}
	err = json.Unmarshal(data, &dhcp6Conf)
	if err != nil {
		log.Fatalf("err parsing dhcp6 conf: %v", err)
	}
	return Kea{
		DHCP4ConfPath:            dhcp4ConfPath,
		DHCP6ConfPath:            dhcp6ConfPath,
		DHCP4Conf:                dhcp4Conf,
		DHCP6Conf:                dhcp6Conf,
		ControlAgentLeases:       "false",
		ControlAgentReservations: "false",
		UseIPv4:                  useIPv4,
		UseIPv6:                  useIPv6,
	}
}

func TestGetIPsForLeaseHostname(t *testing.T) {
	kea := MakeTestKeaControlAgent()
	kea.ControlAgentLeases = "true"
	kea.ControlAgentReservations = "false"

	info, err := kea.GetIPsForHostname(leaseHostname)

	if err != nil {
		t.Error(err)
	}
	if len(info) < 1 {
		t.Log("Received no results")
		t.FailNow()
	}
	found := false
	for _, ip := range info {
		t.Log(ip)
		if ip.String() == leaseIPv4 || ip.String() == leaseIPv6 {
			found = true
		}
	}
	if !found {
		t.Log("Did not find expected IP")
		t.Fail()
	}
}

func TestGetIPsForLeaseReservation(t *testing.T) {
	if includeReservationTests != "true" {
		t.Log("Skipping reservation test")
		t.SkipNow()
	}

	kea := MakeTestKeaControlAgent()
	kea.ControlAgentLeases = "false"
	kea.ControlAgentReservations = "true"
	info, err := kea.GetIPsForHostname(reservationHostname)

	if err != nil {
		t.Log("Received no results")
		t.Error(err)
	}
	if len(info) < 1 {
		t.FailNow()
	}
	for _, ip := range info {
		t.Log(ip)
	}
	t.Fail()
}

func TestGetIPsWithIncludedNetworkFilter(t *testing.T) {
	kea := MakeTestKeaControlAgent()
	kea.Networks = includedNetworks
	kea.ControlAgentLeases = "true"
	info, err := kea.GetIPsForHostname(leaseHostname)

	if err != nil {
		t.Error(err)
	}
	if len(info) < 1 {
		t.Log("Received no results")
		t.FailNow()
	}
	for _, ip := range info {
		t.Log(ip)
	}
}

func TestGetIPsWithExcludedNetworkFilter(t *testing.T) {
	kea := MakeTestKeaControlAgent()
	kea.Networks = excludedNetworks
	info, err := kea.GetIPsForHostname(leaseHostname)

	if err != nil {
		t.Error(err)
	}
	if len(info) > 0 {
		t.Log("Received unexpected result")
		t.FailNow()
	}
}

func TestGetIPsForConfFileReservationIPv4(t *testing.T) {
	kea := MakeTestKeaConfFiles()
	testReservation := kea.DHCP4Conf.Dhcp4.Subnet4[0].Reservations[0]
	info, err := kea.GetIPsForHostname(testReservation.Hostname)

	if err != nil {
		t.Log("Received no results")
		t.Error(err)
	}
	if len(info) < 1 {
		t.FailNow()
	}
	if info[0].String() != testReservation.IpAddress {
		t.Log("Did not find expected IP")
		t.Fail()
	}
}

func TestGetIPsForConfFileReservationIPv6(t *testing.T) {
	kea := MakeTestKeaConfFiles()
	testReservation := kea.DHCP6Conf.Dhcp6.Subnet6[0].Reservations[0]
	info, err := kea.GetIPsForHostname(testReservation.Hostname)

	if err != nil {
		t.Log("Received no results")
		t.Error(err)
	}
	if len(info) < 1 {
		t.FailNow()
		for _, ip := range testReservation.IpAddresses {
			t.Log(ip)
			found := false
			for _, a := range info {
				if a.String() == ip {
					found = true
					break
				}
			}
			if !found {
				t.Log("Did not find expected IP")
				t.Log(ip)
				t.Fail()
			}
		}
	}
}
