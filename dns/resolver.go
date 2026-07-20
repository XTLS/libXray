package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const resolverTimeout = 16 * time.Second

var errProtectDNSConnection = errors.New("protect DNS connection failed")

type protectSocket func(fd uintptr) bool

func newResolver(server string, protect protectSocket) (*net.Resolver, error) {
	if err := validateServer(server); err != nil {
		return nil, err
	}

	dialer := &net.Dialer{Timeout: resolverTimeout}
	if protect != nil {
		dialer.Control = func(_, _ string, connection syscall.RawConn) error {
			var protectErr error
			if err := connection.Control(func(fd uintptr) {
				if !protect(fd) {
					protectErr = errProtectDNSConnection
				}
			}); err != nil {
				return err
			}
			return protectErr
		}
	}

	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			// Android may report a loopback resolver to Go. Always use the DNS
			// endpoint selected by the VPN configuration instead.
			return dialer.DialContext(ctx, network, server)
		},
	}, nil
}

func validateServer(server string) error {
	host, portText, err := net.SplitHostPort(server)
	if err != nil {
		return fmt.Errorf("invalid DNS server %q: %w", server, err)
	}
	if zoneIndex := strings.LastIndexByte(host, '%'); zoneIndex >= 0 {
		host = host[:zoneIndex]
	}
	if net.ParseIP(host) == nil {
		return fmt.Errorf("invalid DNS server IP %q", host)
	}
	port, err := strconv.Atoi(portText)
	if err != nil || port < 1 || port > 65535 {
		return fmt.Errorf("invalid DNS server port %q", portText)
	}
	return nil
}
