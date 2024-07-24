//go:build linux

package tun

import (
	"golang.zx2c4.com/wireguard/tun"
)

var (
	tunDevice *tun.Device
)

func StartTun(name string, mtu int) (int, error) {
	device, err := tun.CreateTUN(name, mtu)
	if err != nil {
		return 0, err
	}
	tunDevice = &device
	fd := device.File().Fd()
	return int(fd), nil
}

func StopTun() error {
	if tunDevice != nil {
		tun := *tunDevice
		err := tun.Close()
		if err != nil {
			return err
		}
		tunDevice = nil
	}
	return nil
}
