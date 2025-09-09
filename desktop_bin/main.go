package main

import (
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"

	"github.com/xtls/libxray/xray"
)

func main() {
	configPath := os.Args[1]
	fmt.Println("configPath:", configPath)
	err := runXray(configPath)
	if err != nil {
		os.Exit(1)
	}
	defer xray.StopXray()
	// Explicitly triggering GC to remove garbage from config loading.
	runtime.GC()
	debug.FreeOSMemory()

	{
		osSignals := make(chan os.Signal, 1)
		signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)
		<-osSignals
	}
}
