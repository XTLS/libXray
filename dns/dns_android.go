//go:build android

package dns

import (
	"net"
	"sync"
)

var (
	resolverMu       sync.Mutex
	previousResolver *net.Resolver
)

// SetDNS replaces Go's process-wide default resolver with an Android VPN-aware
// resolver. The caller must serialize this with the Xray lifecycle.
func SetDNS(server string, protect protectSocket) error {
	resolver, err := newResolver(server, protect)
	if err != nil {
		return err
	}

	resolverMu.Lock()
	defer resolverMu.Unlock()
	if previousResolver == nil {
		previousResolver = net.DefaultResolver
	}
	net.DefaultResolver = resolver
	return nil
}

// ResetDNS restores the resolver that was active before SetDNS.
func ResetDNS() {
	resolverMu.Lock()
	defer resolverMu.Unlock()
	if previousResolver == nil {
		return
	}
	net.DefaultResolver = previousResolver
	previousResolver = nil
}
