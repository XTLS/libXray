//go:build android

package dns

import (
	"context"
	"net"
	"syscall"
	"time"

	"github.com/xtls/libxray/controller"
)

// Give a callback when parsing server domain. Useful for Android development.
// It depends on xray:api:beta
func InitDns(controller controller.DialerController, dns string) {
	if dnsDialer != nil {
		dnsDialer = nil
	}

	dnsDialer = &net.Dialer{
		Timeout: time.Second * 16,
	}

	if controller != nil {
		dnsDialer.Control = func(network, address string, c syscall.RawConn) error {
			return c.Control(func(fd uintptr) {
				controller.ProtectFd(int(fd))
			})
		}
	}

	net.DefaultResolver = &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
			return dnsDialer.DialContext(ctx, network, dns)
		},
	}
}

func ResetDns() {
	if dnsDialer != nil {
		dnsDialer = nil
	}

	net.DefaultResolver = &net.Resolver{}
}
