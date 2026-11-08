//go:build windows || (linux && !android)

package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
)

type runOptions struct {
	setDNS        bool
	dns           string
	interfaceName string
	configPath    string
	errorFile     string
}

func parseRunOptions(args []string) (runOptions, error) {
	var options runOptions
	if len(args) == 0 || args[0] != "run" {
		return options, errors.New("expected run command")
	}

	flags := flag.NewFlagSet("run", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	flags.BoolVar(&options.setDNS, "set-dns", true, "override the process DNS resolver")
	flags.StringVar(&options.dns, "dns", "", "DNS server IP endpoint")
	flags.StringVar(&options.interfaceName, "interface", "", "outbound network interface")
	flags.StringVar(&options.configPath, "config", "", "Xray JSON configuration path")
	flags.StringVar(&options.errorFile, "error-file", "", "also write command errors to this file")
	if err := flags.Parse(args[1:]); err != nil {
		return options, err
	}
	if flags.NArg() != 0 {
		return options, errors.New("unexpected positional arguments")
	}
	if options.configPath == "" {
		return options, errors.New("config is required")
	}
	if options.setDNS && (options.dns == "" || options.interfaceName == "") {
		return options, errors.New("dns and interface are required when set-dns is enabled")
	}
	return options, nil
}

func execute(args []string, run func(runOptions) error, stdout, stderr io.Writer) int {
	usage := func() {
		fmt.Fprintln(stdout, "Usage: xray run -config <xray.json> [-set-dns=true|false] [-dns <IP:port> -interface <name>] [-error-file <path>]")
		fmt.Fprintln(stdout, "SetDNS is enabled by default; -dns and -interface are required unless -set-dns=false.")
	}
	if len(args) == 1 && (args[0] == "-h" || args[0] == "--help") {
		usage()
		return 0
	}

	options, err := parseRunOptions(args)
	if errors.Is(err, flag.ErrHelp) {
		usage()
		return 0
	}
	if err == nil && options.errorFile != "" {
		// Clear the previous failure before starting. Reuse a caller-created file
		// so an elevated process preserves the caller's read permissions.
		if err := os.WriteFile(options.errorFile, nil, 0600); err != nil {
			fmt.Fprintln(stderr, err)
			return 1
		}
	}
	if err == nil {
		err = run(options)
	}
	if err != nil {
		fmt.Fprintln(stderr, err)
		if options.errorFile != "" {
			if writeErr := os.WriteFile(options.errorFile, []byte(err.Error()), 0600); writeErr != nil {
				fmt.Fprintln(stderr, writeErr)
			}
		}
		return 1
	}
	return 0
}
