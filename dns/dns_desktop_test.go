//go:build windows || (linux && !android)

package dns

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	xrayNet "github.com/xtls/xray-core/common/net"
)

func TestSetDNSInstallsAndRestoresResolver(t *testing.T) {
	interfaces, err := net.Interfaces()
	require.NoError(t, err)

	var interfaceName string
	for _, iface := range interfaces {
		if iface.Flags&net.FlagUp != 0 && iface.Flags&net.FlagLoopback == 0 {
			interfaceName = iface.Name
			break
		}
	}
	if interfaceName == "" {
		t.Skip("no active non-loopback interface")
	}

	original := net.DefaultResolver
	originalXray := xrayNet.DefaultResolver
	t.Cleanup(func() {
		resolverMu.Lock()
		defer resolverMu.Unlock()
		net.DefaultResolver = original
		xrayNet.DefaultResolver = originalXray
		previousResolver = nil
		previousXrayResolver = nil
	})

	require.NoError(t, SetDNS("8.8.8.8:53", interfaceName))
	require.NotSame(t, original, net.DefaultResolver)
	require.Same(t, net.DefaultResolver, xrayNet.DefaultResolver)
	require.True(t, net.DefaultResolver.PreferGo)

	ResetDNS()
	require.Same(t, original, net.DefaultResolver)
	require.Same(t, originalXray, xrayNet.DefaultResolver)
}

func TestValidateDNSInterface(t *testing.T) {
	tests := []struct {
		name    string
		flags   net.Flags
		wantErr bool
	}{
		{name: "active", flags: net.FlagUp},
		{name: "inactive", wantErr: true},
		{name: "loopback", flags: net.FlagUp | net.FlagLoopback, wantErr: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateDNSInterface(&net.Interface{Flags: test.flags})
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}
