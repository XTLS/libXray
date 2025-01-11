//go:build android

package dns

import (
	"context"
	"net"
	"syscall"
	"time"
)

// Give a callback when parsing server domain. Useful for Android development.
func InitDns(server string, controller func(fd uintptr)) {
	if dnsDialer != nil {
		dnsDialer = nil
	}

	dnsDialer = &net.Dialer{
		Timeout: time.Second * 16,
	}

	if controller != nil {
		dnsDialer.Control = func(network, address string, c syscall.RawConn) error {
			return c.Control(controller)
		}
	}

	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return dnsDialer.DialContext(ctx, network, server)
		},
	}
}

func ResetDns() {
	if dnsDialer != nil {
		dnsDialer = nil
	}

	net.DefaultResolver = &net.Resolver{}
}
