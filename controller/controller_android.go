//go:build android

package controller

import (
	"syscall"

	xinternet "github.com/xtls/xray-core/transport/internet"
)

// Give a callback before connection beginning. Useful for Android development.
// It depends on xray:api:beta
func RegisterDialerController(controller func(fd uintptr)) {
	xinternet.RegisterDialerController(func(network, address string, conn syscall.RawConn) error {
		return conn.Control(controller)
	})
}

// Give a callback before listener beginning. Useful for Android development.
// It depends on xray:api:beta
func RegisterListenerController(controller func(fd uintptr)) {
	xinternet.RegisterListenerController(func(network, address string, conn syscall.RawConn) error {
		return conn.Control(controller)
	})
}
