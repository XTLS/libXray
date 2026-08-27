//go:build windows || (linux && !android)

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"syscall"

	"github.com/xtls/libxray/dns"
	"github.com/xtls/libxray/xray"
)

type runOptions struct {
	dns           string
	interfaceName string
	configPath    string
}

func parseRunOptions(args []string) (runOptions, error) {
	var options runOptions
	if len(args) == 0 || args[0] != "run" {
		return options, errors.New("expected run command")
	}

	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.StringVar(&options.dns, "dns", "", "DNS server IP endpoint")
	flags.StringVar(&options.interfaceName, "interface", "", "outbound network interface")
	flags.StringVar(&options.configPath, "config", "", "Xray JSON configuration path")
	if err := flags.Parse(args[1:]); err != nil {
		return options, err
	}
	if flags.NArg() != 0 {
		return options, errors.New("unexpected positional arguments")
	}
	if options.dns == "" || options.interfaceName == "" || options.configPath == "" {
		return options, errors.New("dns, interface, and config are required")
	}
	return options, nil
}

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

func printUsage() {
	fmt.Fprintln(os.Stdout, "Usage: xray run -dns <IP:port> -interface <name> -config <xray.json>")
}

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "-h" || os.Args[1] == "--help") {
		printUsage()
		return
	}

	options, err := parseRunOptions(os.Args[1:])
	if errors.Is(err, flag.ErrHelp) {
		printUsage()
		return
	}
	if err == nil {
		err = run(options)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
