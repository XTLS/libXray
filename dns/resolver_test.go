package dns

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/require"
)

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
