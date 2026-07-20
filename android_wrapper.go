//go:build android

package libXray

import (
	"errors"

	c "github.com/xtls/libxray/controller"
	"github.com/xtls/libxray/dns"
)

type DialerController interface {
	ProtectFd(int) bool
}

// SetDNS installs an Android VPN-aware process resolver. server must be an IP
// endpoint such as 8.8.8.8:53 or [2001:4860:4860::8888]:53.
func SetDNS(controller DialerController, server string) error {
	if controller == nil {
		return errors.New("dns dialer controller is nil")
	}
	return dns.SetDNS(server, func(fd uintptr) bool {
		return controller.ProtectFd(int(fd))
	})
}

// ResetDNS restores the resolver that was active before SetDNS.
func ResetDNS() {
	dns.ResetDNS()
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
