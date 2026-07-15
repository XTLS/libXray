//go:build android

package controller

import (
	"fmt"
	"syscall"

	corenet "github.com/xtls/xray-core/common/net"
	xinternet "github.com/xtls/xray-core/transport/internet"
)

// RegisterDialerController -> callback before connection begins.
// Depends on xray:api:beta
func RegisterDialerController(controller func(fd uintptr)) {
	xinternet.RegisterDialerController(func(network, address string, conn syscall.RawConn) error {
		return conn.Control(controller)
	})
}

// RegisterListenerController -> callback before listener begins.
// Depends on xray:api:beta
func RegisterListenerController(controller func(fd uintptr)) {
	xinternet.RegisterListenerController(func(network, address string, conn syscall.RawConn) error {
		return conn.Control(controller)
	})
}

// sdkThresholdGetConnectionOwner -> min API level for getConnectionOwnerUid().
// Below this we use /proc/net/* parsing.
const sdkThresholdGetConnectionOwner = 30

var androidSdkVersion int

// RegisterProcessFinder -> registers process finder for per-app routing.
// Pass nil to unregister. sdkVersion = Build.VERSION.SDK_INT.
// When SDK < 30, falls back to /proc/net/* parsing (pure Go).
func RegisterProcessFinder(finder func(network, srcIP string, srcPort int, destIP string, destPort int) int, sdkVersion int) {
	androidSdkVersion = sdkVersion

	if finder == nil {
		corenet.RegisterAndroidProcessFinder(nil)
		return
	}

	corenet.RegisterAndroidProcessFinder(func(network, srcIP string, srcPort uint16, destIP string, destPort uint16) (int, string, string, error) {
		if destPort == 0 || destIP == "" {
			return 0, "", "", fmt.Errorf("process finder: no destination for %s %s:%d", network, srcIP, srcPort)
		}

		if androidSdkVersion > 0 && androidSdkVersion < sdkThresholdGetConnectionOwner {
			uid, err := resolveUidFromProc(network, srcIP, int(srcPort), destIP, int(destPort))
			if err == nil {
				return uid, fmt.Sprintf("%d", uid), "", nil
			}
		}

		uid := finder(network, srcIP, int(srcPort), destIP, int(destPort))
		return uid, fmt.Sprintf("%d", uid), "", nil
	})
}
