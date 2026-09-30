//go:build windows || (linux && !android)

package main

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

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

func TestCommandReportsActualStartupError(t *testing.T) {
	directory := commandTestDirectory(t)
	path := filepath.Join(directory, "xray.json.error")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"run", "-dns", "8.8.8.8:53", "-interface", "Ethernet", "-config", "xray.json", "-error-file", path}
	want := "failed to load geosite: category TEST-MISSING not found"
	var stdout, stderr bytes.Buffer
	called := false
	code := execute(args, func(runOptions) error {
		called = true
		return errors.New(want)
	}, &stdout, &stderr)
	if code != 1 || !called {
		t.Fatalf("run was not attempted: code=%d, called=%v, stderr=%q", code, called, stderr.String())
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want || stderr.String() != want+"\n" {
		t.Fatalf("diagnostics differ: file=%q, stderr=%q", data, stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("unexpected stdout: %s", &stdout)
	}
}

func TestCommandClearsStaleErrorBeforeRun(t *testing.T) {
	path := filepath.Join(commandTestDirectory(t), "xray.json.error")
	if err := os.WriteFile(path, []byte("previous failure"), 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"run", "-dns", "8.8.8.8:53", "-interface", "Ethernet", "-config", "xray.json", "-error-file", path}
	var stdout, stderr bytes.Buffer
	code := execute(args, func(runOptions) error {
		data, err := os.ReadFile(path)
		if err != nil || len(data) != 0 {
			t.Fatalf("stale error remains before run: %q, err=%v", data, err)
		}
		return nil
	}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("successful run failed: code=%d, stderr=%q", code, stderr.String())
	}
}

func TestCommandWithoutErrorFilePreservesStderr(t *testing.T) {
	args := []string{"run", "-dns", "8.8.8.8:53", "-interface", "Ethernet", "-config", "xray.json"}
	var stdout, stderr bytes.Buffer
	code := execute(args, func(runOptions) error {
		return errors.New("actual startup error")
	}, &stdout, &stderr)
	if code != 1 || stderr.String() != "actual startup error\n" {
		t.Fatalf("unexpected error: code=%d, stderr=%q", code, stderr.String())
	}
}

func TestCommandRejectsUnwritableErrorFileBeforeRun(t *testing.T) {
	path := filepath.Join(commandTestDirectory(t), "missing", "error")
	args := []string{"run", "-dns", "8.8.8.8:53", "-interface", "Ethernet", "-config", "xray.json", "-error-file", path}
	var stdout, stderr bytes.Buffer
	code := execute(args, func(runOptions) error {
		t.Fatal("run must not start when its error file cannot be prepared")
		return nil
	}, &stdout, &stderr)
	if code != 1 || !bytes.Contains(stderr.Bytes(), []byte(path)) {
		t.Fatalf("missing file error: code=%d, stderr=%q", code, stderr.String())
	}
}

func commandTestDirectory(t *testing.T) string {
	t.Helper()
	root := filepath.Join("..", "..", "references", "libxray-desktop-tests")
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	directory, err := os.MkdirTemp(root, "command-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(directory) })
	return directory
}
