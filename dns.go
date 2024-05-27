package libXray

import (
	"context"
	"net"
	"syscall"
	"time"
)

var (
	dialer *net.Dialer
)

// Give a callback when parsing server domain. Useful for Android development.
// this function is under development, and there is no guarantee for its availability.
// It depends on xray:api:beta
func InitDns(controller DialerController, dns string) {
	if dialer != nil {
		dialer = nil
	}

	dialer = &net.Dialer{
		Timeout: time.Second * 16,
	}

	if controller != nil {
		dialer.Control = func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				controller.ProtectFd(int(fd))
			})
		}
	}

	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, dns)
		},
	}
}
