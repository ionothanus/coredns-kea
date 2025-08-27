package kea

import (
	"github.com/coredns/caddy"
	"github.com/coredns/coredns/core/dnsserver"
	"github.com/coredns/coredns/plugin"
)

// init registers this plugin.
func init() { plugin.Register("kea", setup) }

func setup(c *caddy.Controller) error {

	controlAgent := ""
	networks := []string{}
	insecure := ""
	extractHostname := ""
	useLeases := ""
	useReservations := ""
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
				break
			case "networks":
				for c.NextArg() {
					networks = append(networks, c.Val())
				}
				if len(networks) == 0 {
					return plugin.Error("kea", c.ArgErr())
				}
				break
			case "insecure":
				if !c.NextArg() {
					return plugin.Error("kea", c.ArgErr())
				}
				insecure = c.Val()
				break
			case "extract_hostname":
				if !c.NextArg() {
					return plugin.Error("kea", c.ArgErr())
				}
				extractHostname = c.Val()
				break
			case "use_leases":
				if !c.NextArg() {
					return plugin.Error("kea", c.ArgErr())
				}
				useLeases = c.Val()
				break
			case "use_reservations":
				if !c.NextArg() {
					return plugin.Error("kea", c.ArgErr())
				}
				useReservations = c.Val()
				break
			case "use_ipv4":
				if !c.NextArg() {
					return plugin.Error("kea", c.ArgErr())
				}
				useIPv4 = c.Val()
				break
			case "use_ipv6":
				if !c.NextArg() {
					return plugin.Error("kea", c.ArgErr())
				}
				useIPv6 = c.Val()
				break
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

	if controlAgent == "" {
		return plugin.Error("kea", c.ArgErr())
	}

	if useIPv4 != "true" && useIPv6 != "true" {
		return plugin.Error("kea", c.Err("use_ipv4 and use_ipv6 cannot both be false"))
	}

	// Add the Plugin to CoreDNS, so Servers can use it in their plugin chain.
	dnsserver.GetConfig(c).AddPlugin(func(next plugin.Handler) plugin.Handler {
		return Kea{
			ControlAgent:    controlAgent,
			Networks:        networks,
			Insecure:        insecure,
			ExtractHostname: extractHostname,
			UseLeases:       useLeases,
			UseReservations: useReservations,
			Next:            next,
			UseIPv4:         useIPv4,
			UseIPv6:         useIPv6,
		}
	})

	// All OK, return a nil error.
	return nil
}
