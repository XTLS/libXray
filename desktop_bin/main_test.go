//go:build windows || (linux && !android)

package main

import "testing"

func TestParseRunOptions(t *testing.T) {
	options, err := parseRunOptions([]string{
		"run",
		"-dns", "8.8.8.8:53",
		"-interface", "Ethernet",
		"-config", `C:\run\xray.json`,
		"-runtime", `C:\run\runtime.json`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if options.dns != "8.8.8.8:53" || options.interfaceName != "Ethernet" || options.configPath != `C:\run\xray.json` {
		t.Fatalf("unexpected options: %#v", options)
	}
	if options.runtimePath != `C:\run\runtime.json` {
		t.Fatalf("unexpected runtime path: %q", options.runtimePath)
	}

	if _, err := parseRunOptions([]string{"run", "-config", "xray.json"}); err == nil {
		t.Fatal("missing DNS protection options were accepted")
	}
}
