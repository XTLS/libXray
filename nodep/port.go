package nodep

import (
	"fmt"
	"net"
	"strings"
)

// https://github.com/phayes/freeport/blob/master/freeport.go
// GetFreePort asks the kernel for free open ports that are ready to use.
func GetFreePorts(count int) string {
	var ports []int
	for i := 0; i < count; i++ {
		addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
		if err != nil {
			return ""
		}

		l, err := net.ListenTCP("tcp", addr)
		if err != nil {
			return ""
		}
		defer l.Close()
		ports = append(ports, l.Addr().(*net.TCPAddr).Port)
	}
	return strings.Trim(strings.Join(strings.Fields(fmt.Sprint(ports)), ":"), "[]")
}
