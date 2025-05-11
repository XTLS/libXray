//go:build linux

package main

import (
	"net"
	"strconv"

	"github.com/vishvananda/netlink"
)

// sudo ip route add default dev tun0 metric 20
// sudo ip -6 route add default dev tun0 metric 20
func initIpRoute(tunName string, tunPriority string) error {
	priority, err := strconv.Atoi(tunPriority)
	if err != nil {
		return err
	}
	link, err := netlink.LinkByName(tunName)
	if err != nil {
		return err
	}
	err = addRoute(link.Attrs().Index, "0.0.0.0/0", netlink.FAMILY_V4, priority)
	if err != nil {
		return err
	}
	err = addRoute(link.Attrs().Index, "::/0", netlink.FAMILY_V6, priority)
	if err != nil {
		return err
	}

	return nil
}

func addRoute(index int, cidr string, family int, priority int) error {
	_, defaultDst, err := net.ParseCIDR(cidr)
	if err != nil {
		return err
	}
	route := netlink.Route{Dst: defaultDst, LinkIndex: index, Family: family, Priority: priority}
	err = netlink.RouteAdd(&route)
	if err != nil {
		return err
	}
	return nil
}
