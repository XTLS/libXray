//go:build windows || (linux && !android)

package dns

import (
	"errors"
	"net"
)

// SetDNS installs a process-wide resolver bound to interfaceName.
func SetDNS(server, interfaceName string) error {
	if interfaceName == "" {
		return errors.New("dns interface is required")
	}
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return err
	}
	if err := validateDNSInterface(iface); err != nil {
		return err
	}

	resolver, err := newProtectedResolver(server, func(network string, fd uintptr) error {
		return bindDNSInterface(network, fd, iface)
	})
	if err != nil {
		return err
	}

	installDefaultResolver(resolver)
	return nil
}

func validateDNSInterface(iface *net.Interface) error {
	if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
		return errors.New("dns interface must be an active non-loopback interface")
	}
	return nil
}

// ResetDNS restores the resolver that was active before SetDNS.
func ResetDNS() {
	restoreDefaultResolver()
}
