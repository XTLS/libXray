//go:build windows || (linux && !android)

package dns

import (
	"errors"
	"net"
	"sync"
)

var (
	desktopResolverMu sync.Mutex
	previousResolver  *net.Resolver
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
	if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
		return errors.New("dns interface must be an active non-loopback interface")
	}

	resolver, err := newProtectedResolver(server, func(network string, fd uintptr) error {
		return bindDNSInterface(network, fd, iface)
	})
	if err != nil {
		return err
	}

	desktopResolverMu.Lock()
	defer desktopResolverMu.Unlock()
	if previousResolver == nil {
		previousResolver = net.DefaultResolver
	}
	net.DefaultResolver = resolver
	return nil
}

// ResetDNS restores the resolver that was active before SetDNS.
func ResetDNS() {
	desktopResolverMu.Lock()
	defer desktopResolverMu.Unlock()
	if previousResolver == nil {
		return
	}
	net.DefaultResolver = previousResolver
	previousResolver = nil
}
