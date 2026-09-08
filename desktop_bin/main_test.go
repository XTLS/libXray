//go:build windows || (linux && !android)

package main

import "testing"

func TestParseRunOptions(t *testing.T) {
	options, err := parseRunOptions([]string{
		"run",
		"-dns", "8.8.8.8:53",
		"-interface", "Ethernet",
		"-config", `C:\run\xray.json`,
	})
	if err != nil {
		t.Fatal(err)
	}
	if options.dns != "8.8.8.8:53" || options.interfaceName != "Ethernet" || options.configPath != `C:\run\xray.json` {
		t.Fatalf("unexpected options: %#v", options)
	}

	if _, err := parseRunOptions([]string{"run", "-config", "xray.json"}); err == nil {
		t.Fatal("missing DNS protection options were accepted")
	}
}
