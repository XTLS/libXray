//go:build android

package dns

// SetDNS replaces Go's process-wide default resolver with an Android VPN-aware
// resolver. The caller must serialize this with the Xray lifecycle.
func SetDNS(server string, protect protectSocket) error {
	resolver, err := newResolver(server, protect)
	if err != nil {
		return err
	}

	installDefaultResolver(resolver)
	return nil
}

// ResetDNS restores the resolver that was active before SetDNS.
func ResetDNS() {
	restoreDefaultResolver()
}
