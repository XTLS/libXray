package libxray

import (
	xinternet "github.com/xtls/xray-core/transport/internet"
)

type DialerController interface {
	FdCallback(int) bool
}

func RegisterDialerController(controller DialerController) {
	xinternet.RegisterDialerController(func(network, address string, fd uintptr) error {
		controller.FdCallback(int(fd))
		return nil
	})
}
