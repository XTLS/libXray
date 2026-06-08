//go:build android

package controller

import (
	"fmt"
	"syscall"

	corenet "github.com/xtls/xray-core/common/net"
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

// RegisterProcessFinder registers an Android process finder with Xray-core,
// enabling per-app routing based on UID. Must be called before starting the
// core for process-based routing rules to work.
// Pass nil to unregister a previously registered finder.
func RegisterProcessFinder(finder func(network, srcIP string, srcPort int, destIP string, destPort int) int) {
	if finder == nil {
		corenet.RegisterAndroidProcessFinder(nil)
		return
	}

	corenet.RegisterAndroidProcessFinder(func(network, srcIP string, srcPort uint16, destIP string, destPort uint16) (int, string, string, error) {
		if destPort == 0 || destIP == "" {
			return 0, "", "", fmt.Errorf("process finder: no destination for %s %s:%d", network, srcIP, srcPort)
		}

		uid := finder(network, srcIP, int(srcPort), destIP, int(destPort))
		return uid, fmt.Sprintf("%d", uid), "", nil
	})
}
