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
	switch network {
	case "tcp4", "udp4", "ip4":
		index := int(int32(bits.ReverseBytes32(uint32(iface.Index))))
		return windows.SetsockoptInt(
			windows.Handle(fd),
			windows.IPPROTO_IP,
			ipUnicastInterface,
			index,
		)
	case "tcp6", "udp6", "ip6":
		return windows.SetsockoptInt(
			windows.Handle(fd),
			windows.IPPROTO_IPV6,
			ipv6UnicastInterface,
			iface.Index,
		)
	default:
		return fmt.Errorf("unsupported DNS network %q", network)
	}
}
