//go:build linux

package main

import (
	"fmt"
	"net"

	"github.com/vishvananda/netlink"
)

// sudo ip addr add 198.18.0.1/15 dev tun0
// sudo ip -6 addr add fc00::1/64 dev tun0
// sudo ip route add default via 198.18.0.2 dev tun0 metric 20
// sudo ip -6 route add default via fc00::2 dev tun0 metric 20
func initIpRoute(tunName string, tunPriority int) error {
	var link netlink.Link
	err := retryRouteInitStep("find tun device "+tunName, func() error {
		var err error
		link, err = netlink.LinkByName(tunName)
		return err
	})
	if err != nil {
		return err
	}

	err = netlink.LinkSetUp(link)
	if err != nil {
		return fmt.Errorf("set %s up failed: %w", tunName, err)
	}

	err = addAddress(link, defaultTunIPv4Address)
	if err != nil {
		return err
	}

	err = addAddress(link, defaultTunIPv6Address)
	if err != nil {
		return err
	}

	err = addRoute(link.Attrs().Index, defaultIPv4Route, defaultTunIPv4Gateway, netlink.FAMILY_V4, tunPriority)
	if err != nil {
		return err
	}

	err = addRoute(link.Attrs().Index, defaultIPv6Route, defaultTunIPv6Gateway, netlink.FAMILY_V6, tunPriority)
	if err != nil {
		return err
	}

	return nil
}

func addAddress(link netlink.Link, address string) error {
	addr, err := netlink.ParseAddr(address)
	if err != nil {
		return fmt.Errorf("invalid address %q: %w", address, err)
	}
	err = netlink.AddrAdd(link, addr)
	if err != nil {
		return fmt.Errorf("add address %q failed: %w", address, err)
	}
	return nil
}

func addRoute(index int, cidr string, gateway string, family int, priority int) error {
	_, defaultDst, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	gw := net.ParseIP(gateway)
	if gw == nil {
		return fmt.Errorf("invalid gateway %q", gateway)
	}
	route := netlink.Route{Dst: defaultDst, Gw: gw, LinkIndex: index, Family: family, Priority: priority}
	err = netlink.RouteAdd(&route)
	if err != nil {
		return err
	}
	return nil
}
