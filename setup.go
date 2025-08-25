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
		}
	})

	// All OK, return a nil error.
	return nil
}
