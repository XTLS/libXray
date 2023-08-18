package libXray

import (
	"syscall"

	xinternet "github.com/xtls/xray-core/transport/internet"
)

type DialerController interface {
	FdCallback(int) bool
}

// Give a callback before connection beginning. Useful for Android development.
// It depends on xray:api:beta
func RegisterDialerController(controller DialerController) {
	xinternet.RegisterDialerController(func(network, address string, conn syscall.RawConn) error {
		return conn.Control(func(fd uintptr) {
			controller.FdCallback(int(fd))
		})
	})
}
