package xray

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	appstats "github.com/xtls/xray-core/app/stats"
	"github.com/xtls/xray-core/features/stats"
)

func runtimeConfig(t *testing.T) RuntimeConfig {
	t.Helper()
	return RuntimeConfig{
		StatePath:  filepath.Join(t.TempDir(), "runtime.json"),
		InboundTag: "tunIn",
	}
}

func runtimeFixture(t *testing.T, config RuntimeConfig) (*managedRuntime, stats.Counter, stats.Counter) {
	t.Helper()
	runtime, err := prepareRuntime(&config)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = runtime.stop()
		_ = runtime.stateLock.Close()
	})
	manager, err := appstats.NewManager(context.Background(), &appstats.Config{})
	if err != nil {
		t.Fatal(err)
	}
	runtime.manager = manager
	up, _ := manager.RegisterCounter(runtime.counterName("uplink"))
	down, _ := manager.RegisterCounter(runtime.counterName("downlink"))
	return runtime, up, down
}

func saveRuntimeSample(t *testing.T, runtime *managedRuntime) runtimeSnapshot {
	t.Helper()
	runtime.sample()
	if err := runtime.save(); err != nil {
		t.Fatal(err)
	}
	return savedRuntime(t, runtime.config.StatePath)
}

func savedRuntime(t *testing.T, path string) runtimeSnapshot {
	t.Helper()
	snapshot, err := readRuntimeState(path)
	if err != nil || snapshot.Version != 1 {
		t.Fatalf("read snapshot: %+v %v", snapshot, err)
	}
	return snapshot
}

func TestRuntimeStoresRawCountersWithoutResetOrTotals(t *testing.T) {
	config := runtimeConfig(t)
	runtime, up, down := runtimeFixture(t, config)
	up.Add(100)
	down.Add(200)
	first := saveRuntimeSample(t, runtime)
	repeated := saveRuntimeSample(t, runtime)
	if first.Session.Uplink != 100 || first.Session.Downlink != 200 || first.Session != repeated.Session || up.Value() != 100 || down.Value() != 200 {
		t.Fatalf("sampling duplicated or reset raw counters: %+v %+v", first, repeated)
	}
	// A nonnegative rollback remains a raw value, not a synthetic accumulated delta.
	up.Set(2)
	rollback := saveRuntimeSample(t, runtime)
	up.Add(3)
	resumed := saveRuntimeSample(t, runtime)
	if rollback.Session.Uplink != 2 || resumed.Session.Uplink != 5 || !resumed.Available {
		t.Fatalf("rollback changed raw counter semantics: %+v %+v", rollback, resumed)
	}
	up.Set(-1)
	negative := saveRuntimeSample(t, runtime)
	if negative.Available || negative.Error != "counters_unavailable" || negative.Session != resumed.Session {
		t.Fatalf("negative counter corrupted the last valid sample: %+v", negative)
	}
	_ = runtime.manager.UnregisterCounter(runtime.counterName("uplink"))
	missing := saveRuntimeSample(t, runtime)
	if missing.Available || missing.Error != "counters_unavailable" || missing.Session != resumed.Session {
		t.Fatalf("missing counter became a fabricated zero: %+v", missing)
	}
	if info, err := os.Stat(config.StatePath); err != nil || info.Mode().Perm() != 0600 {
		t.Fatalf("private file mode: %v %v", info, err)
	}
	encoded, _ := json.Marshal(first)
	var fields map[string]json.RawMessage
	_ = json.Unmarshal(encoded, &fields)
	if len(fields) != 6 || fields["ledger"] != nil || fields["totalUplink"] != nil || fields["resetGeneration"] != nil || strings.Contains(string(encoded), config.StatePath) {
		t.Fatalf("unexpected runtime snapshot fields: %s", encoded)
	}
	metadata, _ := json.Marshal(config)
	var metadataFields map[string]json.RawMessage
	_ = json.Unmarshal(metadata, &metadataFields)
	if len(metadataFields) != 2 || metadataFields["planId"] != nil || metadataFields["controlAddress"] != nil || metadataFields["controlToken"] != nil {
		t.Fatalf("runtime metadata exposed a control surface: %s", metadata)
	}
}

func TestRuntimeReplacesPreviousSessionOnStart(t *testing.T) {
	config := runtimeConfig(t)
	runtime, up, down := runtimeFixture(t, config)
	if err := runtime.start(); err != nil {
		t.Fatal(err)
	}
	initial := savedRuntime(t, config.StatePath)
	if initial.Session.Uplink != 0 || initial.Session.Downlink != 0 || !initial.Available || initial.Session.EndedAtMs != 0 {
		t.Fatalf("new session was not saved at start: %+v", initial)
	}
	up.Add(17)
	down.Add(23)
	if err := runtime.stop(); err != nil {
		t.Fatal(err)
	}
	stopped := savedRuntime(t, config.StatePath)
	if stopped.Session.Uplink != 17 || stopped.Session.Downlink != 23 || stopped.Session.EndedAtMs == 0 {
		t.Fatalf("stop did not save final raw counters: %+v", stopped)
	}
	_ = runtime.stateLock.Close()
	// Preparation alone cannot overwrite the current snapshot.
	for range 2 {
		next, err := prepareRuntime(&config)
		if err != nil {
			t.Fatal(err)
		}
		_ = next.stateLock.Close()
		if savedRuntime(t, config.StatePath) != stopped {
			t.Fatal("preparation replaced the previous current snapshot")
		}
	}
	next, _, _ := runtimeFixture(t, config)
	if err := next.start(); err != nil {
		t.Fatal(err)
	}
	current := savedRuntime(t, config.StatePath)
	if current.Session.ID == stopped.Session.ID || current.Session.Uplink != 0 || current.Session.Downlink != 0 {
		t.Fatalf("restart reused a session or inherited totals: %+v", current)
	}
	if _, err := os.Lstat(filepath.Join(filepath.Dir(config.StatePath), "runtime-sessions")); !os.IsNotExist(err) {
		t.Fatal("restart created a runtime session archive")
	}
}

func TestRuntimeWriteFailuresPreserveSavedFile(t *testing.T) {
	config := runtimeConfig(t)
	runtime, up, _ := runtimeFixture(t, config)
	up.Add(20)
	committed := saveRuntimeSample(t, runtime)
	runtime.config.StatePath = filepath.Join(filepath.Dir(config.StatePath), "missing", "runtime.json")
	up.Add(5)
	runtime.sample()
	if err := runtime.save(); err == nil || runtime.snapshot.Error != "state_write_failed" || runtime.snapshot.SavedAtMs != committed.SavedAtMs {
		t.Fatalf("failed save incorrectly advanced the watermark: %+v %v", runtime.snapshot, err)
	}
	if savedRuntime(t, config.StatePath) != committed {
		t.Fatal("failed save changed the last saved file")
	}
	runtime.config.StatePath = config.StatePath
	up.Add(8)
	recovered := saveRuntimeSample(t, runtime)
	if recovered.Session.Uplink != 33 || recovered.Error != "" {
		t.Fatalf("retry synthesized or lost raw bytes: %+v", recovered)
	}
}

func TestRuntimeConfigAndStateBoundary(t *testing.T) {
	config := runtimeConfig(t)
	for _, mutate := range []func(*RuntimeConfig){
		func(c *RuntimeConfig) { c.StatePath = "relative.json" },
		func(c *RuntimeConfig) { c.InboundTag = "" },
		func(c *RuntimeConfig) { c.Listen = "127.0.0.1:12345" },
		func(c *RuntimeConfig) { c.Token = strings.Repeat("a", 32) },
		func(c *RuntimeConfig) { c.Listen, c.Token = "0.0.0.0:12345", strings.Repeat("a", 32) },
		func(c *RuntimeConfig) { c.Listen, c.Token = "localhost:12345", strings.Repeat("a", 32) },
		func(c *RuntimeConfig) { c.Listen, c.Token = "[::1]:12345", strings.Repeat("a", 32) },
		func(c *RuntimeConfig) { c.Listen, c.Token = "127.0.0.1:0", strings.Repeat("a", 32) },
		func(c *RuntimeConfig) { c.Listen, c.Token = "127.0.0.1:65536", strings.Repeat("a", 32) },
		func(c *RuntimeConfig) { c.Listen, c.Token = "127.0.0.1:12345", strings.Repeat("A", 32) },
		func(c *RuntimeConfig) { c.Listen, c.Token = "127.0.0.1:12345", "secret" },
	} {
		invalid := config
		mutate(&invalid)
		if r, err := prepareRuntime(&invalid); err == nil {
			_ = r.stateLock.Close()
			t.Fatal("invalid runtime metadata accepted")
		}
	}
	for _, text := range []string{
		`{`, `{"version":9}`,
		`{"version":1,"session":{"id":"../../outside","startedAtMs":1},"sampledAtMs":1,"savedAtMs":1}`,
		`{"version":1,"session":{"id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","startedAtMs":1,"uplink":-1},"sampledAtMs":1,"savedAtMs":1}`,
		`{"version":1,"session":{"id":"aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","planId":"old","startedAtMs":1},"sampledAtMs":1,"savedAtMs":1}`,
	} {
		if err := os.WriteFile(config.StatePath, []byte(text), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := readRuntimeState(config.StatePath); err == nil {
			t.Fatal("invalid saved session was accepted")
		}
		if saved, err := os.ReadFile(config.StatePath); err != nil || string(saved) != text {
			t.Fatal("invalid saved session was overwritten")
		}
	}
	runtime, err := prepareRuntime(&config)
	if err != nil {
		t.Fatalf("saved state unnecessarily blocked preparation: %v", err)
	}
	_ = runtime.stateLock.Close()
}

func TestManagedRuntimeStartFailureAndStop(t *testing.T) {
	t.Cleanup(func() { _ = StopXray() })
	config := runtimeHTTPConfig(t)
	if err := RunXrayWithRuntime(minimalConfig, &config); err == nil || GetXrayState() {
		t.Fatal("missing statistics were accepted")
	}
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	_, port, _ := net.SplitHostPort(listener.Addr().String())
	xrayJSON := fmt.Sprintf(`{"log":{"loglevel":"none"},"stats":{},"policy":{"system":{"statsInboundUplink":true,"statsInboundDownlink":true}},"inbounds":[{"tag":"tunIn","listen":"127.0.0.1","port":%s,"protocol":"socks","settings":{"udp":false}}],"outbounds":[{"protocol":"freedom","tag":"direct"}]}`, port)
	if err := RunXrayWithRuntime(xrayJSON, &config); err == nil || GetXrayState() {
		t.Fatal("occupied inbound port did not fail startup")
	}
	_ = listener.Close()
	statisticsListener, err := net.Listen("tcp4", config.Listen)
	if err != nil {
		t.Fatal(err)
	}
	if err := RunXrayWithRuntime(xrayJSON, &config); err == nil || GetXrayState() {
		t.Fatal("occupied statistics port did not close the constructed core")
	}
	_ = statisticsListener.Close()
	validPath := config.StatePath
	config.StatePath = filepath.Join(filepath.Dir(validPath), "missing", "runtime.json")
	if err := RunXrayWithRuntime(xrayJSON, &config); err == nil || GetXrayState() {
		t.Fatal("unwritable runtime directory did not fail startup")
	}
	config.StatePath = validPath
	if err := RunXrayWithRuntime(xrayJSON, &config); err != nil {
		t.Fatal(err)
	}
	snapshot := savedRuntime(t, config.StatePath)
	if !snapshot.Available || snapshot.Session.Uplink != 0 || snapshot.Session.EndedAtMs != 0 {
		t.Fatalf("idle statistics should be available zero: %+v", snapshot)
	}
	// Even a final persistence error must close the core and release the owner lock.
	coreRuntime.config.StatePath = filepath.Join(filepath.Dir(validPath), "missing", "runtime.json")
	if err := StopXray(); err == nil || GetXrayState() {
		t.Fatalf("failed final save did not close core: %v", err)
	}
	statisticsListener, err = net.Listen("tcp4", config.Listen)
	if err != nil {
		t.Fatalf("failed final save leaked the statistics listener: %v", err)
	}
	_ = statisticsListener.Close()
	next, err := prepareRuntime(&config)
	if err != nil {
		t.Fatalf("stop did not release ownership: %v", err)
	}
	_ = next.stateLock.Close()
}

func TestRuntimePeriodicSaveAndSingleOwner(t *testing.T) {
	config := runtimeConfig(t)
	runtime, up, down := runtimeFixture(t, config)
	if err := runtime.start(); err != nil {
		t.Fatal(err)
	}
	initial := savedRuntime(t, config.StatePath)
	if other, err := prepareRuntime(&config); err == nil || err.Error() != "runtime state is in use" {
		if other != nil {
			_ = other.stateLock.Close()
		}
		t.Fatalf("two writers acquired the same path: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRuntimeChild$")
	command.Env = append(os.Environ(), "LIBXRAY_TEST_RUNTIME_PATH="+config.StatePath, "LIBXRAY_TEST_RUNTIME_ACTION=lock")
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("cross-process owner lock failed: %v: %s", err, output)
	}
	up.Add(31)
	down.Add(47)
	// Exercise the real 30s timer without adding a production interval option.
	deadline := time.Now().Add(35 * time.Second)
	for time.Now().Before(deadline) {
		snapshot := savedRuntime(t, config.StatePath)
		if snapshot.Session.Uplink == 31 && snapshot.Session.Downlink == 47 && snapshot.SavedAtMs > initial.SavedAtMs {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatal("host timer did not save counters without any UI/control request")
}

func TestRuntimeKilledOwnerStateIsReplacedOnRestart(t *testing.T) {
	config := runtimeConfig(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestRuntimeChild$")
	command.Env = append(os.Environ(), "LIBXRAY_TEST_RUNTIME_PATH="+config.StatePath, "LIBXRAY_TEST_RUNTIME_ACTION=kill")
	pipe, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	scanner := bufio.NewScanner(pipe)
	ready := false
	for scanner.Scan() {
		if scanner.Text() == "READY" {
			ready = true
			break
		}
	}
	if !ready {
		_ = command.Process.Kill()
		_ = command.Wait()
		t.Fatalf("child did not reach unsaved tail: %v", scanner.Err())
	}
	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	if err := command.Wait(); err == nil {
		t.Fatal("child was not forcibly terminated")
	}
	saved := savedRuntime(t, config.StatePath)
	if saved.Session.Uplink != 17 || saved.Session.Downlink != 23 || saved.Session.EndedAtMs != 0 {
		t.Fatalf("kill invented a final sample or ending: %+v", saved)
	}
	runtime, _, _ := runtimeFixture(t, config)
	if err := runtime.start(); err != nil {
		t.Fatal(err)
	}
	if current := savedRuntime(t, config.StatePath); current.Session.ID == saved.Session.ID || current.Session.Uplink != 0 || current.Session.Downlink != 0 {
		t.Fatalf("restart reused killed owner's counters: %+v", current)
	}
	if _, err := os.Lstat(filepath.Join(filepath.Dir(config.StatePath), "runtime-sessions")); !os.IsNotExist(err) {
		t.Fatal("restart archived the killed owner's saved state")
	}
}

func TestRuntimeChild(t *testing.T) {
	path := os.Getenv("LIBXRAY_TEST_RUNTIME_PATH")
	if path == "" {
		return
	}
	config := RuntimeConfig{StatePath: path, InboundTag: "tunIn"}
	if os.Getenv("LIBXRAY_TEST_RUNTIME_ACTION") == "lock" {
		if other, err := prepareRuntime(&config); err == nil || err.Error() != "runtime state is in use" {
			if other != nil {
				_ = other.stateLock.Close()
			}
			t.Fatalf("another process acquired active session ownership: %v", err)
		}
		return
	}
	runtime, up, down := runtimeFixture(t, config)
	if err := runtime.start(); err != nil {
		t.Fatal(err)
	}
	up.Add(17)
	down.Add(23)
	saveRuntimeSample(t, runtime)
	up.Add(101)
	down.Add(103)
	fmt.Fprintln(os.Stdout, "READY")
	select {}
}
