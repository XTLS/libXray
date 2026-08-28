package dns

import (
	"context"
	"errors"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	xrayNet "github.com/xtls/xray-core/common/net"
)

func TestDefaultResolverLifecycle(t *testing.T) {
	before := net.DefaultResolver
	beforeXray := xrayNet.DefaultResolver
	original := &net.Resolver{}
	originalXray := &net.Resolver{}
	net.DefaultResolver = original
	xrayNet.DefaultResolver = originalXray
	t.Cleanup(func() {
		resolverMu.Lock()
		defer resolverMu.Unlock()
		net.DefaultResolver = before
		xrayNet.DefaultResolver = beforeXray
		previousResolver = nil
		previousXrayResolver = nil
	})

	first := &net.Resolver{PreferGo: true}
	second := &net.Resolver{StrictErrors: true}

	installDefaultResolver(first)
	require.Same(t, first, net.DefaultResolver)
	require.Same(t, first, xrayNet.DefaultResolver)

	installDefaultResolver(second)
	require.Same(t, second, net.DefaultResolver)
	require.Same(t, second, xrayNet.DefaultResolver)

	restoreDefaultResolver()
	require.Same(t, original, net.DefaultResolver)
	require.Same(t, originalXray, xrayNet.DefaultResolver)

	restoreDefaultResolver()
	require.Same(t, original, net.DefaultResolver)
	require.Same(t, originalXray, xrayNet.DefaultResolver)
}

func TestNewResolverUsesConfiguredServerAndProtectsSocket(t *testing.T) {
	server, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	defer server.Close()

	protected := false
	resolver, err := newResolver(server.LocalAddr().String(), func(uintptr) bool {
		protected = true
		return true
	})
	require.NoError(t, err)

	connection, err := resolver.Dial(
		context.Background(),
		"udp",
		"127.0.0.1:53",
	)
	require.NoError(t, err)
	defer connection.Close()

	require.True(t, protected)
	require.Equal(t, server.LocalAddr().String(), connection.RemoteAddr().String())
}

func TestNewResolverRejectsFailedProtection(t *testing.T) {
	server, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	defer server.Close()

	resolver, err := newResolver(server.LocalAddr().String(), func(uintptr) bool {
		return false
	})
	require.NoError(t, err)

	connection, err := resolver.Dial(
		context.Background(),
		"udp",
		"127.0.0.1:53",
	)
	if connection != nil {
		connection.Close()
	}
	require.ErrorIs(t, err, errProtectDNSConnection)
}

func TestNewProtectedResolverReturnsControlError(t *testing.T) {
	controlErr := errors.New("bind DNS interface")
	controlCalled := false
	resolver, err := newProtectedResolver("127.0.0.1:53", func(string, uintptr) error {
		controlCalled = true
		return controlErr
	})
	require.NoError(t, err)

	connection, err := resolver.Dial(
		context.Background(),
		"udp",
		"127.0.0.1:53",
	)
	if connection != nil {
		connection.Close()
	}

	require.True(t, controlCalled)
	require.ErrorIs(t, err, controlErr)
}

func TestNewResolverValidatesServer(t *testing.T) {
	tests := []struct {
		name   string
		server string
	}{
		{name: "missing port", server: "8.8.8.8"},
		{name: "hostname", server: "dns.example.com:53"},
		{name: "zero port", server: "8.8.8.8:0"},
		{name: "invalid port", server: "8.8.8.8:dns"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			resolver, err := newResolver(test.server, nil)
			require.Nil(t, resolver)
			require.Error(t, err)
		})
	}

	for _, server := range []string{
		"8.8.8.8:53",
		"[2001:4860:4860::8888]:53",
	} {
		t.Run(server, func(t *testing.T) {
			resolver, err := newResolver(server, nil)
			require.NotNil(t, resolver)
			require.NoError(t, err)
		})
	}
}
