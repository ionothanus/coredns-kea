package kea

import (
	"testing"

	"github.com/coredns/caddy"
)

// TestSetup tests the various things that should be parsed by setup.
// Make sure you also test for parse errors.
func TestSetup(t *testing.T) {
	tests := []struct {
		inputFileRules string
		shouldErr      bool
		//expectedInterfaces string
	}{
		{
			`kea {
				control_agent "https://kea.example.com:8000"
			}`,
			false,
		},
		{
			`kea {
				control_agent "https://kea.example.com:8000"
				insecure true
			}`,
			false,
		},
		{
			`kea {
				control_agent "https://kea.example.com:8000"
				insecure false
			}`,
			false,
		},
		{
			`kea {
				control_agent "https://kea.example.com:8000"
				use_leases false
			}`,
			false,
		},
		{
			`kea {
				control_agent "https://kea.example.com:8000"
				use_leases true
			}`,
			false,
		},
		{
			`kea {
				control_agent "https://kea.example.com:8000"
				use_reservations false
			}`,
			false,
		},
		{
			`kea {
				control_agent "https://kea.example.com:8000"
				use_reservations true
			}`,
			false,
		},
		{
			`kea {
				control_agent "https://kea.example.com:8000"
				extract_hostname true
			}`,
			false,
		},
		{
			`kea {
				control_agent "https://kea.example.com:8000"
				extract_hostname true
			}`,
			false,
		},
		{
			`kea {
				control_agent "https://kea.example.com:8000"
				use_reservations
			}`,
			true,
		},
		{
			`kea {
				control_agent "https://kea.example.com:8000"
				insecure true
				networks 10.10.22.0/24
			}`,
			false,
		},
		{
			`kea {
				control_agent "https://kea.example.com:8000"
				insecure false
				networks 10.10.22.0/24
			}`,
			false,
		},
		{
			`kea {
				control_agent "https://kea.example.com:8000"
				use_ipv4 false
			}`,
			false,
		},
		{
			`kea {
				control_agent "https://kea.example.com:8000"
				use_ipv6 false
			}`,
			false,
		},
		{
			`kea {
				control_agent "https://kea.example.com:8000"
				networks
			}`,
			true,
		},
		{
			`kea {
			}`,
			true,
		},
		{
			`kea {
				control_agent
			}`,
			true,
		},
		{
			`kea {
				control_agent "https://kea.example.com:8000"
				use_ipv4 false
				use_ipv6 false
			}`,
			true,
		},
	}

	for i, test := range tests {
		c := caddy.NewTestController("dns", test.inputFileRules)
		err := setup(c)

		if err == nil && test.shouldErr {
			t.Fatalf("Test %d expected errors, but got no error", i)
		} else if err != nil && !test.shouldErr {
			t.Fatalf("Test %d expected no errors, but got '%v'", i, err)
		}
	}
}
