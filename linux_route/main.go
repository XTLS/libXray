package main

import (
	"os"
)

func main() {
	tunName := os.Args[1]
	tunPriority := os.Args[2]
	err := initIpRoute(tunName, tunPriority)
	if err != nil {
		os.Stdout.WriteString(err.Error())
		os.Exit(-1)
	}
}
