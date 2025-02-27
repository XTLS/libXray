//go:build linux && !android

package dns

import (
	"context"
	"net"
	"syscall"
	"time"
)

func InitDns(dns string, deviceName string) {
	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			// address on linux is always loopback address.
			// so we use a custom dns instead.
			dialer := makeDialer(deviceName)
			return dialer.DialContext(ctx, network, dns)
		},
	}
}

func makeDialer(deviceName string) *net.Dialer {
	dialer := &net.Dialer{
		Timeout: time.Second * 16,
	}

	dialer.Control = func(network, address string, c syscall.RawConn) error {
		err := c.Control(func(fd uintptr) {
			syscall.BindToDevice(int(fd), deviceName)
		})
		return err
	}

	return dialer
}
