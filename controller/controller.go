package controller

import (
	"errors"
	"syscall"
)

var errProtectSocket = errors.New("protect Android VPN socket failed")

func protectSocket(conn syscall.RawConn, controller func(fd uintptr) bool) error {
	var protectErr error
	if err := conn.Control(func(fd uintptr) {
		if !controller(fd) {
			protectErr = errProtectSocket
		}
	}); err != nil {
		return err
	}
	return protectErr
}
