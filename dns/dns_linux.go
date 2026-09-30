//go:build linux && !android

package dns

import (
	"fmt"
	"net"

	"golang.org/x/sys/unix"
)

func bindDNSInterface(network string, fd uintptr, iface *net.Interface) error {
	if err := unix.BindToDevice(int(fd), iface.Name); err != nil {
		return fmt.Errorf("bind DNS %s socket to interface %q: %w", network, iface.Name, err)
	}
	return nil
}
