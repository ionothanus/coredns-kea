package kea

import (
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

func MakeTestKea() Kea {
	return Kea{
		ControlAgent:    controlAgent,
		Insecure:        insecure,
		UseLeases:       "true",
		UseReservations: includeReservationTests,
		UseIPv4:         useIPv4,
		UseIPv6:         useIPv6,
	}
}

func TestGetIPsForLeaseHostname(t *testing.T) {
	kea := MakeTestKea()
	kea.UseLeases = "true"
	kea.UseReservations = "false"

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

	kea := MakeTestKea()
	kea.UseLeases = "false"
	kea.UseReservations = "true"
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
	kea := MakeTestKea()
	kea.Networks = includedNetworks
	kea.UseLeases = "true"
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
	kea := MakeTestKea()
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
