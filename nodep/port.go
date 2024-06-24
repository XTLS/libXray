package nodep

import (
	"net"
)

// https://github.com/phayes/freeport/blob/master/freeport.go
// GetFreePort asks the kernel for free open ports that are ready to use.
func GetFreePorts(count int) ([]int, error) {
	var ports []int
	for i := 0; i < count; i++ {
		addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
		if err != nil {
			return ports, err
		}

		l, err := net.ListenTCP("tcp", addr)
		if err != nil {
			return ports, err
		}
		defer l.Close()
		ports = append(ports, l.Addr().(*net.TCPAddr).Port)
	}
	return ports, nil
}
