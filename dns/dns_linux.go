//go:build linux && !android

package dns

import (
	"context"
	"net"
	"syscall"
	"time"
)

func InitDns(dns string, deviceName string) {
	if dnsDialer != nil {
		dnsDialer = nil
	}

	dnsDialer = &net.Dialer{
		Timeout: time.Second * 16,
	}

	dnsDialer.Control = func(network, address string, c syscall.RawConn) error {
		//copy from https://github.com/apernet/hysteria/blob/master/extras/outbounds/ob_direct_linux.go
		var errBind error
		err := c.Control(func(fd uintptr) {
			errBind = syscall.BindToDevice(int(fd), deviceName)
		})
		if err != nil {
			return err
		}
		return errBind
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
