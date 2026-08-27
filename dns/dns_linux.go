//go:build linux && !android

package dns

import (
	"net"

	"golang.org/x/sys/unix"
)

func bindDNSInterface(_ string, fd uintptr, iface *net.Interface) error {
	return unix.BindToDevice(int(fd), iface.Name)
}
