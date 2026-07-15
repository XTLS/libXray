//go:build android

package libXray

import c "github.com/xtls/libxray/controller"

type DialerController interface {
	ProtectFd(int) bool
}

// ProcessFinder -> implemented by apps to resolve UID from connection details.
// See RegisterProcessFinder.
type ProcessFinder interface {
	// FindProcessByConnection -> returns UID owning the connection, or -1.
	FindProcessByConnection(network, srcIP string, srcPort int, destIP string, destPort int) int
}

func RegisterDialerController(controller DialerController) {
	c.RegisterDialerController(func(fd uintptr) {
		controller.ProtectFd(int(fd))
	})
}

func RegisterListenerController(controller DialerController) {
	c.RegisterListenerController(func(fd uintptr) {
		controller.ProtectFd(int(fd))
	})
}

// RegisterProcessFinder -> registers process finder for per-app routing.
// Pass nil to unregister.
// sdkVersion = Build.VERSION.SDK_INT.
// On API < 30, /proc/net/* parsing is used automatically; the Java callback
// is only called on API 30+.
func RegisterProcessFinder(finder ProcessFinder, sdkVersion int) {
	if finder == nil {
		c.RegisterProcessFinder(nil, 0)
		return
	}

	c.RegisterProcessFinder(func(network, srcIP string, srcPort int, destIP string, destPort int) int {
		return finder.FindProcessByConnection(network, srcIP, srcPort, destIP, destPort)
	}, sdkVersion)
}
