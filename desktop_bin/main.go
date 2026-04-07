package main

import (
	"flag"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
)

func main() {
	configPath := flag.String("configPath", "config.json", "Path of config.json")
	flag.Parse()
	err := runXray(*configPath)
	if err != nil {
		os.Exit(1)
	}
	defer stopXray()
	// Explicitly triggering GC to remove garbage from config loading.
	runtime.GC()
	debug.FreeOSMemory()

	{
		osSignals := make(chan os.Signal, 1)
		signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)
		<-osSignals
	}
}
