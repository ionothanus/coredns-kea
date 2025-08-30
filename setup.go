package kea

import (
	"encoding/json"
	"os"

	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

// init registers this plugin.
func init() { plugin.Register("kea", setup) }

func setup(c *caddy.Controller) error {

	controlAgent := ""
	dhcp4_conf := ""
	dhcp6_conf := ""
	networks := []string{}
	insecure := "false"
	extractHostname := "false"
	controlAgentLeases := ""
	controlAgentReservations := "true"
	useIPv4 := "true"
	useIPv6 := "true"

	c.Next()
	if c.NextBlock() {
		for {
			switch c.Val() {
			case "control_agent":
				if !c.NextArg() {
					return plugin.Error("kea", c.ArgErr())
				}
				controlAgent = c.Val()
				controlAgentLeases = "true"
			case "dhcp4_conf":
				if !c.NextArg() {
					return plugin.Error("kea", c.ArgErr())
				}
				dhcp4_conf = c.Val()
				useIPv4 = "true"
			case "dhcp6_conf":
				if !c.NextArg() {
					return plugin.Error("kea", c.ArgErr())
				}
				dhcp6_conf = c.Val()
				useIPv6 = "true"
			case "networks":
				for c.NextArg() {
					networks = append(networks, c.Val())
				}
				if len(networks) == 0 {
					return plugin.Error("kea", c.ArgErr())
				}
			case "insecure":
				if !c.NextArg() {
					return plugin.Error("kea", c.ArgErr())
				}
				insecure = c.Val()
			case "extract_hostname":
				if !c.NextArg() {
					return plugin.Error("kea", c.ArgErr())
				}
				extractHostname = c.Val()
			case "control_agent_leases":
				if !c.NextArg() {
					return plugin.Error("kea", c.ArgErr())
				}
				controlAgentLeases = c.Val()
			case "control_agent_reservations":
				if !c.NextArg() {
					return plugin.Error("kea", c.ArgErr())
				}
				controlAgentReservations = c.Val()
			case "use_ipv4":
				if !c.NextArg() {
					return plugin.Error("kea", c.ArgErr())
				}
				useIPv4 = c.Val()
			case "use_ipv6":
				if !c.NextArg() {
					return plugin.Error("kea", c.ArgErr())
				}
				useIPv6 = c.Val()
			default:
				if c.Val() != "}" {
					return plugin.Error("kea", c.Err("unknown property"))
				}
			}
			if !c.Next() {
				break
			}
		}
	}

	if controlAgent == "" && dhcp4_conf == "" && dhcp6_conf == "" {
		return plugin.Error("kea", c.Err("One of control_agent, dhcp4_conf or dhcp6_conf must be set"))
	}

	if dhcp4_conf != "" && useIPv4 != "true" {
		return plugin.Error("kea", c.Err("dhcp4_conf requires use_ipv4 to be true"))
	}

	if dhcp6_conf != "" && useIPv6 != "true" {
		return plugin.Error("kea", c.Err("dhcp6_conf requires use_ipv6 to be true"))
	}

	if controlAgent == "" && controlAgentLeases == "true" {
		return plugin.Error("kea", c.Err("use_leases is only valid when control_agent is set (conf files only provide reservations)"))
	}

	if controlAgentLeases != "true" && controlAgentReservations != "true" {
		return plugin.Error("kea", c.Err("control_agent_leases and control_agent_reservations cannot both be false"))
	}

	if useIPv4 != "true" && useIPv6 != "true" {
		return plugin.Error("kea", c.Err("use_ipv4 and use_ipv6 cannot both be false"))
	}

	var dhcp4Conf KeaDHCP4Conf
	var dhcp6Conf KeaDHCP6Conf
	if dhcp4_conf != "" {
		data, err := os.ReadFile(dhcp4_conf)
		if err != nil {
			return plugin.Error("kea", err)
		}
		err = json.Unmarshal(data, &dhcp4Conf)
		if err != nil {
			return plugin.Error("kea", err)
		}
	}
	if dhcp6_conf != "" {
		data, err := os.ReadFile(dhcp6_conf)
		if err != nil {
			return plugin.Error("kea", err)
		}
		err = json.Unmarshal(data, &dhcp6Conf)
		if err != nil {
			return plugin.Error("kea", err)
		}
	}

	// Add the Plugin to CoreDNS, so Servers can use it in their plugin chain.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return Kea{
			ControlAgent:             controlAgent,
			Networks:                 networks,
			Insecure:                 insecure,
			ExtractHostname:          extractHostname,
			ControlAgentLeases:       controlAgentLeases,
			ControlAgentReservations: controlAgentReservations,
			Next:                     next,
			UseIPv4:                  useIPv4,
			UseIPv6:                  useIPv6,
			DHCP4ConfPath:            dhcp4_conf,
			DHCP6ConfPath:            dhcp6_conf,
			DHCP4Conf:                dhcp4Conf,
			DHCP6Conf:                dhcp6Conf,
		}
	})

	// All OK, return a nil error.
	return nil
}
