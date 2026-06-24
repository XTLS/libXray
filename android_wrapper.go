//go:build android

package libXray

import c "github.com/xtls/libxray/controller"

type DialerController interface {
	ProtectFd(int) bool
}

// ProcessFinder is an interface for Android process finding functionality.
// Apps should implement FindProcessByConnection()
// and pass the implementation to RegisterProcessFinder() before starting the core.
type ProcessFinder interface {
	// FindProcessByConnection finds the UID of the process that owns the given connection.
	//
	// network: Protocol type: "tcp" or "udp"
	// srcIP: Source IP address
	// srcPort: Source port
	// destIP: Destination IP address
	// destPort: Destination port
	// Returns the UID of the owning process, or -1 if not found.
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

// RegisterProcessFinder registers an Android process finder with Xray-core,
// enabling per-app routing based on UID. Must be called before starting the
// core for process-based routing rules to work.
// Pass nil to unregister a previously registered finder.
func RegisterProcessFinder(finder ProcessFinder) {
	if finder == nil {
		c.RegisterProcessFinder(nil)
		return
	}

	c.RegisterProcessFinder(func(network, srcIP string, srcPort int, destIP string, destPort int) int {
		return finder.FindProcessByConnection(network, srcIP, srcPort, destIP, destPort)
	})
}
