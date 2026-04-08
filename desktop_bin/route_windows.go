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

	err = addRoute("ipv4", "0.0.0.0/0", tunName, tunPriority)
	if err != nil {
		return err
	}

	err = addIPv6Address(tunName, "fc00::1/64")
	if err != nil {
		return err
	}
	err = addRoute("ipv6", "::/0", tunName, tunPriority)
	if err != nil {
		return err
	}
	return nil
}

func addIPv6Address(tunName string, address string) error {
	cmd := exec.Command("netsh", "interface", "ipv6", "add", "address",
		"interface="+tunName, "address="+address, "store=active")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netsh add ipv6 address failed: %s: %w", string(output), err)
	}
	return nil
}

func addRoute(ipVersion string, prefix string, tunName string, metric int) error {
	cmd := exec.Command("netsh", "interface", ipVersion, "add", "route",
		"prefix="+prefix, "interface="+tunName,
		"metric="+strconv.Itoa(metric), "store=active")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("netsh add route failed: %s: %w", string(output), err)
	}
	return nil
}
