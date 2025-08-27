# example

## Name

*kea* - supports Kea lease hostname lookups

## Description

This plugin will contact a Kea Control Agent to request IP addresses for a given hostname.
It supports lease and reservation lookups, for IPv4 and IPv6 DHCP servers.

### Note
Prior to Kea 3.0, the Control Agent hooks enabling reservation lookup were sold separately and not included in the open-source package. This plugin implements reservation lookup based on the documentation but *reservation support has not been tested*.

As of Kea 3.0, the Control Agent is deprecated; this plugin should still work, but it has only been tested against Kea 2.6.

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
  control_agent http://localhost:8000

  # Filter IP responses to include only ones in the specified CIDRs.
  # If unspecified, filtering will be disabled.
	networks 10.0.0.0/16 10.10.0.0/16

  # Set to "false" if you have an HTTPS proxy for your control agent
  # and you want to enforce a secure connection. "true" by default.
	insecure false
  
  # Use extract_hostname to send only the hostname of a domain name query to Kea.
  # For example, if the request will look up test.example.com, "true" here
  # would send "test" as the hostname to Kea. "false" by default.
	extract_hostname true

  # You can disable one or the other, but at least one
  # of lease and reservation lookups must be enabled.
  # Both are enabled by default.
	use_leases true
	use_reservations false

  # You can disable one or the other, but at least one of IPv4 and IPv6 support must be enabled.
  # Both are enabled by default.
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
