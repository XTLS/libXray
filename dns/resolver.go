package dns

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	xrayNet "github.com/xtls/xray-core/common/net"
)

const resolverTimeout = 16 * time.Second

var errProtectDNSConnection = errors.New("protect DNS connection failed")

var (
	resolverMu           sync.Mutex
	previousResolver     *net.Resolver
	previousXrayResolver *net.Resolver
)

type protectSocket func(fd uintptr) bool
type controlSocket func(network string, fd uintptr) error

func installDefaultResolver(resolver *net.Resolver) {
	resolverMu.Lock()
	defer resolverMu.Unlock()
	if previousResolver == nil {
		previousResolver = net.DefaultResolver
		previousXrayResolver = xrayNet.DefaultResolver
	}
	net.DefaultResolver = resolver
	// Xray-core caches the standard resolver during package initialization.
	xrayNet.DefaultResolver = resolver
}

func restoreDefaultResolver() {
	resolverMu.Lock()
	defer resolverMu.Unlock()
	if previousResolver == nil {
		return
	}
	net.DefaultResolver = previousResolver
	xrayNet.DefaultResolver = previousXrayResolver
	previousResolver = nil
	previousXrayResolver = nil
}

func newResolver(server string, protect protectSocket) (*net.Resolver, error) {
	var control controlSocket
	if protect != nil {
		control = func(_ string, fd uintptr) error {
			if !protect(fd) {
				return errProtectDNSConnection
			}
			return nil
		}
	}
	return newProtectedResolver(server, control)
}

func newProtectedResolver(server string, control controlSocket) (*net.Resolver, error) {
	if err := validateServer(server); err != nil {
		return nil, err
	}

	dialer := &net.Dialer{Timeout: resolverTimeout}
	if control != nil {
		dialer.Control = func(network, _ string, connection syscall.RawConn) error {
			var controlErr error
			if err := connection.Control(func(fd uintptr) {
				controlErr = control(network, fd)
			}); err != nil {
				return err
			}
			return controlErr
		}
	}

	return &net.Resolver{
		PreferGo: true,
		Dial: func(ctx context.Context, network, _ string) (net.Conn, error) {
			// Always use the DNS endpoint selected by the VPN configuration.
			return dialer.DialContext(ctx, network, server)
		},
	}, nil
}

func preflightResolver(resolver *net.Resolver, server string) error {
	connection, err := resolver.Dial(context.Background(), "udp", server)
	if err != nil {
		return err
	}
	return connection.Close()
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
