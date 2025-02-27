//go:build windows

package dns

import (
	"context"
	"net"
	"syscall"
	"time"
)

const (
	IP_UNICAST_IF   = 31
	IPV6_UNICAST_IF = 31
)

func InitDns(_ string, deviceName string) {
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			dialer := makeDialer(deviceName)
			return dialer.DialContext(ctx, network, address)
		},
	}
}

func makeDialer(deviceName string) *net.Dialer {
	dialer := &net.Dialer{
		Timeout: time.Second * 16,
	}

	dialer.Control = func(network, address string, c syscall.RawConn) error {
		err := c.Control(func(fd uintptr) {
			iface, err := net.InterfaceByName(deviceName)
			if err != nil {
				return
			}
			switch network {
			case "tcp4", "udp4":
				syscall.SetsockoptInt(syscall.Handle(fd), syscall.IPPROTO_IP, IP_UNICAST_IF, iface.Index)
			case "tcp6", "udp6":
				syscall.SetsockoptInt(syscall.Handle(fd), syscall.IPPROTO_IPV6, IPV6_UNICAST_IF, iface.Index)
			default:
				syscall.SetsockoptInt(syscall.Handle(fd), syscall.IPPROTO_IP, IP_UNICAST_IF, iface.Index)
				syscall.SetsockoptInt(syscall.Handle(fd), syscall.IPPROTO_IPV6, IPV6_UNICAST_IF, iface.Index)
			}
		})
		return err
	}

	return dialer
}
