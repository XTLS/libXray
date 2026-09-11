//go:build windows || (linux && !android)

package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/xtls/libxray/dns"
	"github.com/xtls/libxray/xray"
)

func run(options runOptions) error {
	config, err := os.ReadFile(options.configPath)
	if err != nil {
		return err
	}
	if err := dns.SetDNS(options.dns, options.interfaceName); err != nil {
		return err
	}
	defer dns.ResetDNS()

	if err := xray.RunXray(string(config)); err != nil {
		return err
	}

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, os.Interrupt, syscall.SIGTERM)
	defer signal.Stop(signals)
	<-signals
	return xray.StopXray()
}

func main() {
	os.Exit(execute(os.Args[1:], run, os.Stdout, os.Stderr))
}
