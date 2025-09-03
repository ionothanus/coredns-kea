# example

## Name

*kea* - supports Kea lease hostname lookups

## Description

This plugin offers two methods to load hostname/IP data from Kea.

| Supports           | Reservations | Leases |
|--------------------|--------------|--------|
| Control agent [^1] | ⚠️ [^2]       | ✅      |
| Configuration file | ✅            | ❌      |

[^1]: The Kea Control Agent has been deprecated in Kea 3.0 in favour of directly contacting the dhcp4 and dhcp6 agents. This plugin doesn't yet support those connections.

[^2]: The Kea Control Agent supports reservation information if it is built with the host control hook. This was a paid add-on before Kea 2.7.7/Kea 3.0. 

### Authentication note

OPNsense doesn't support configuring authentication on its Kea control agent, so neither does this plugin at this time.

## Compilation

This package will always be compiled as part of CoreDNS and not in a standalone way. It will require you to use `go get` or as a dependency on [plugin.cfg](https://github.com/coredns/coredns/blob/master/plugin.cfg).

The [manual](https://coredns.io/manual/toc/#what-is-coredns) will have more information about how to configure and extend the server with external plugins.

A simple way to consume this plugin, is by adding the following on [plugin.cfg](https://github.com/coredns/coredns/blob/master/plugin.cfg), and recompile it as [detailed on coredns.io](https://coredns.io/2017/07/25/compile-time-enabling-or-disabling-plugins/#build-with-compile-time-configuration-file).

~~~
kea:github.com/ionothanus/coredns-kea
~~~

You can compile coredns by:

``` sh
go generate
go build
```

Or you can instead use make:

``` sh
make
```

## Syntax

~~~ txt
kea {
  # One of the following must be configured.
  control_agent http://localhost:8000
  dhcp4_conf /etc/kea/kea-dhcp4.conf
  dhcp6_conf /etc/kea/kea-dhcp6.conf

  # Filter IP responses to include only ones in the specified CIDRs.
  # If unspecified, filtering will be disabled.
	networks 10.0.0.0/16 10.10.0.0/16

  # Set to "false" if you have an HTTPS proxy for your control agent
  # and you want to enforce a secure connection. "false" by default.
	insecure false
  
  # Use extract_hostname to send only the hostname of a domain name query to Kea.
  # For example, if the request will look up test.example.com, "true" here
  # would send "test" as the hostname to Kea. "false" by default.
	extract_hostname true

  # You can disable one or the other, but at least one of lease and reservation lookups 
  # must be enabled when the control agent is enabled.
  # Both are enabled by default.
	control_agent_leases true
	control_agent_reservations false

  # You can disable one or the other, but at least one of IPv4 and IPv6 support must be enabled.
  # Both are enabled by default with the control agent. 
  # They are automatically enabled as appropriate when dhcp[4,6]_conf are set.
	use_ipv4 true
	use_ipv6 true
}
~~~

## Metrics

If monitoring is enabled (via the *prometheus* directive) the following metric is exported:

* `coredns_kea_request_count_total{server}` - query count to the *kea* plugin.

The `server` label indicated which server handled the request, see the *metrics* plugin for details.

## Ready

This plugin reports readiness to the ready plugin. It will be immediately ready.

## Also See

See the [manual](https://coredns.io/manual).
This plugin was derived from [coredns-proxmox](https://github.com/konairius/coredns-proxmox).
