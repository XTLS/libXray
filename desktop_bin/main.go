package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"runtime/debug"
	"syscall"
)

func main() {
	configPath := flag.String("configPath", "", "Path of LibXrayInvokeRequest JSON")
	flag.Parse()
	if *configPath == "" {
		flag.Usage()
		fmt.Fprintln(os.Stderr, "missing -configPath")
		os.Exit(1)
	}

	if err := runXray(*configPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer func() {
		if err := stopXray(); err != nil {
			fmt.Fprintf(os.Stderr, "stopXray failed: %v\n", err)
		}
	}()

	runtime.GC()
	debug.FreeOSMemory()

	osSignals := make(chan os.Signal, 1)
	signal.Notify(osSignals, os.Interrupt, syscall.SIGTERM)
	<-osSignals
}
