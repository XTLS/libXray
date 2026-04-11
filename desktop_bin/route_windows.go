//go:build windows

package main

import (
	"fmt"
	"net"
	"os/exec"
	"strconv"
)

func initIpRoute(tunName string, tunPriority int) error {
	err := retryRouteInitStep("find tun device "+tunName, func() error {
		var err error
		_, err = net.InterfaceByName(tunName)
		return err
	})
	if err != nil {
		return err
	}

	err = addAddress(tunName, "ipv4", defaultTunIPv4Address)
	if err != nil {
		return err
	}

	err = addAddress(tunName, "ipv6", defaultTunIPv6Address)
	if err != nil {
		return err
	}

	err = addRoute(tunName, "ipv4", defaultIPv4Route, defaultTunIPv4Gateway, tunPriority)
	if err != nil {
		return err
	}

	err = addRoute(tunName, "ipv6", defaultIPv6Route, defaultTunIPv6Gateway, tunPriority)
	if err != nil {
		return err
	}
	return nil
}

func addAddress(tunName string, ipVersion string, address string) error {
	ip, ipNet, err := net.ParseCIDR(address)
	if err != nil {
		return fmt.Errorf("invalid %s address %q: %w", ipVersion, address, err)
	}

	switch ipVersion {
	case "ipv4":
		if ip.To4() == nil {
			return fmt.Errorf("ipv4 address must be an IPv4 CIDR: %q", address)
		}
		mask := net.IP(ipNet.Mask).String()
		args := []string{"interface", "ipv4", "add", "address",
			"name=" + tunName, "address=" + ip.String(), "mask=" + mask}
		args = append(args, "store=active")
		cmd := exec.Command("netsh", args...)
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("netsh add ipv4 address failed: %s: %w", string(output), err)
		}
		return nil
	case "ipv6":
		if ip.To4() != nil {
			return fmt.Errorf("ipv6 address must be an IPv6 CIDR: %q", address)
		}
		cmd := exec.Command("netsh", "interface", "ipv6", "add", "address",
			"interface="+tunName, "address="+address, "store=active")
		output, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("netsh add ipv6 address failed: %s: %w", string(output), err)
		}
		return nil
	default:
		return fmt.Errorf("unsupported ip version %q", ipVersion)
	}
}

func addRoute(tunName string, ipVersion string, prefix string, gateway string, metric int) error {
	routeIP, _, err := net.ParseCIDR(prefix)
	if err != nil {
		return fmt.Errorf("invalid %s route prefix %q: %w", ipVersion, prefix, err)
	}

	args := []string{"interface", ipVersion, "add", "route",
		"prefix=" + prefix, "interface=" + tunName}

	switch ipVersion {
	case "ipv4":
		if routeIP.To4() == nil {
			return fmt.Errorf("ipv4 route prefix must be an IPv4 CIDR: %q", prefix)
		}
		if gateway != "" {
			gatewayIP := net.ParseIP(gateway)
			if gatewayIP == nil || gatewayIP.To4() == nil {
				return fmt.Errorf("ipv4 route gateway must be an IPv4 address: %q", gateway)
			}
			args = append(args, "nexthop="+gateway)
		}
	case "ipv6":
		if routeIP.To4() != nil {
			return fmt.Errorf("ipv6 route prefix must be an IPv6 CIDR: %q", prefix)
		}
		if gateway != "" {
			gatewayIP := net.ParseIP(gateway)
			if gatewayIP == nil || gatewayIP.To4() != nil {
				return fmt.Errorf("ipv6 route gateway must be an IPv6 address: %q", gateway)
			}
			args = append(args, "nexthop="+gateway)
		}
	default:
		return fmt.Errorf("unsupported ip version %q", ipVersion)
	}

	args = append(args, "metric="+strconv.Itoa(metric), "store=active")
	cmd := exec.Command("netsh", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netsh add %s route failed: %s: %w", ipVersion, string(output), err)
	}
	return nil
}
