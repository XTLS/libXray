package nodep

import (
	"fmt"
	"net"
)

// Port allocation approach: https://github.com/phayes/freeport/blob/master/freeport.go
// GetFreePorts returns distinct free TCP ports, excluding caller-reserved ports.
// Listeners are released before returning, so the ports are not reserved for use.
func GetFreePorts(count int, excludePorts []int) ([]int, error) {
	return getFreePorts(count, excludePorts, func() (net.Listener, error) {
		return net.Listen("tcp", "localhost:0")
	})
}

func getFreePorts(count int, excludePorts []int, listen func() (net.Listener, error)) ([]int, error) {
	excluded := make(map[int]struct{}, len(excludePorts))
	for _, port := range excludePorts {
		if port < 1 || port > 65535 {
			return nil, fmt.Errorf("excluded port must be between 1 and 65535: %d", port)
		}
		excluded[port] = struct{}{}
	}
	if count < 0 || count > 65535-len(excluded) {
		return nil, fmt.Errorf("requested port count exceeds the available range: %d", count)
	}

	var ports []int
	var listeners []net.Listener
	defer func() {
		for _, listener := range listeners {
			listener.Close()
		}
	}()
	for len(ports) < count {
		listener, err := listen()
		if err != nil {
			return ports, err
		}
		// Hold rejected ports too: closing them here could let the kernel pick
		// the same excluded port indefinitely. All listeners close on return.
		listeners = append(listeners, listener)
		port := listener.Addr().(*net.TCPAddr).Port
		if _, skip := excluded[port]; !skip {
			excluded[port] = struct{}{}
			ports = append(ports, port)
		}
	}
	return ports, nil
}
