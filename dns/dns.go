package dns

import "net"

func ResetDns() {
	net.DefaultResolver = &net.Resolver{}
}
