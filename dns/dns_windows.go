//go:build windows

package dns

import (
	"fmt"
	"math/bits"
	"net"

	"golang.org/x/sys/windows"
)

const (
	ipUnicastInterface   = 31
	ipv6UnicastInterface = 31
)

func bindDNSInterface(network string, fd uintptr, iface *net.Interface) error {
	var err error
	switch network {
	case "tcp4", "udp4", "ip4":
		index := int(int32(bits.ReverseBytes32(uint32(iface.Index))))
		err = windows.SetsockoptInt(
			windows.Handle(fd),
			windows.IPPROTO_IP,
			ipUnicastInterface,
			index,
		)
	case "tcp6", "udp6", "ip6":
		err = windows.SetsockoptInt(
			windows.Handle(fd),
			windows.IPPROTO_IPV6,
			ipv6UnicastInterface,
			iface.Index,
		)
	default:
		return fmt.Errorf("unsupported DNS network %q for interface %q", network, iface.Name)
	}
	if err != nil {
		return fmt.Errorf("bind DNS %s socket to interface %q: %w", network, iface.Name, err)
	}
	return nil
}
