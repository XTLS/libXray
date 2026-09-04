package xray

import (
	"encoding/json"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func runtimeHTTPConfig(t *testing.T) RuntimeConfig {
	t.Helper()
	config := runtimeConfig(t)
	listener, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	config.Listen, config.Token = listener.Addr().String(), strings.Repeat("a", 32)
	_ = listener.Close()
	return config
}

func requestRuntime(t *testing.T, config RuntimeConfig, method, path, token string, status int) RuntimeSnapshot {
	t.Helper()
	request, err := http.NewRequest(method, "http://"+config.Listen+path, nil)
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	data, err := io.ReadAll(response.Body)
	if err != nil || response.StatusCode != status {
		t.Fatalf("runtime HTTP status %d, want %d: %s %v", response.StatusCode, status, data, err)
	}
	if response.Header.Get("Cache-Control") != "no-store" || response.Header.Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("runtime HTTP must disable caching without enabling CORS")
	}
	if strings.Contains(string(data), config.StatePath) || strings.Contains(string(data), config.Token) {
		t.Fatal("runtime HTTP exposed host metadata")
	}
	var snapshot RuntimeSnapshot
	if status == http.StatusOK {
		if response.Header.Get("Content-Type") != "application/json" || json.Unmarshal(data, &snapshot) != nil || snapshot.Version != 1 {
			t.Fatalf("invalid runtime response: %s", data)
		}
	}
	return snapshot
}

func TestRuntimeHTTPAuthenticationCurrentSessionAndStop(t *testing.T) {
	config := runtimeHTTPConfig(t)
	runtime, up, down := runtimeFixture(t, config)
	up.Add(17)
	down.Add(23)
	if err := runtime.start(); err != nil {
		t.Fatal(err)
	}
	requestRuntime(t, config, http.MethodGet, "/runtime", "", http.StatusUnauthorized)
	requestRuntime(t, config, http.MethodGet, "/runtime", strings.Repeat("b", 32), http.StatusUnauthorized)
	current := requestRuntime(t, config, http.MethodGet, "/runtime", config.Token, http.StatusOK)
	if current.Session.Uplink != 17 || current.Session.Downlink != 23 {
		t.Fatalf("wrong current snapshot: %+v", current)
	}
	up.Add(100)
	if saved := requestRuntime(t, config, http.MethodGet, "/runtime", config.Token, http.StatusOK); saved != current || up.Value() != 117 {
		t.Fatal("HTTP must read the saved snapshot without sampling or resetting metrics")
	}
	requestRuntime(t, config, http.MethodPost, "/runtime", config.Token, http.StatusMethodNotAllowed)
	requestRuntime(t, config, http.MethodGet, "/runtime/ack", config.Token, http.StatusNotFound)
	requestRuntime(t, config, http.MethodPost, "/runtime/ack", config.Token, http.StatusNotFound)
	requestRuntime(t, config, http.MethodGet, "/control", config.Token, http.StatusNotFound)
	if err := os.Remove(config.StatePath); err != nil {
		t.Fatal(err)
	}
	requestRuntime(t, config, http.MethodGet, "/runtime", config.Token, http.StatusServiceUnavailable)
	runtime.sample()
	if err := runtime.save(); err != nil {
		t.Fatal(err)
	}
	if err := runtime.stop(); err != nil {
		t.Fatal(err)
	}
	stopped := savedRuntime(t, config.StatePath)
	if stopped.Session.Uplink != 117 || stopped.Session.EndedAtMs == 0 {
		t.Fatal("HTTP shutdown lost the final saved sample")
	}
	connection, err := net.DialTimeout("tcp4", config.Listen, time.Second)
	if err == nil {
		_ = connection.Close()
		t.Fatal("runtime HTTP listener survived stop")
	}
}

func TestRuntimeHTTPRejectsCurrentSnapshotSymlink(t *testing.T) {
	config := runtimeHTTPConfig(t)
	runtime, _, _ := runtimeFixture(t, config)
	if err := runtime.start(); err != nil {
		t.Fatal(err)
	}
	state := savedRuntime(t, config.StatePath)
	outside := filepath.Join(t.TempDir(), "runtime.json")
	if err := writeRuntimeState(outside, state); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(config.StatePath); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, config.StatePath); err != nil {
		t.Skipf("symbolic links unavailable: %v", err)
	}
	requestRuntime(t, config, http.MethodGet, "/runtime", config.Token, http.StatusServiceUnavailable)
	if savedRuntime(t, outside) != state {
		t.Fatal("HTTP changed a snapshot outside its state path")
	}
}

func TestRuntimeHTTPStartFailureClosesListener(t *testing.T) {
	config := runtimeHTTPConfig(t)
	runtime, _, _ := runtimeFixture(t, config)
	occupied, err := net.Listen("tcp4", config.Listen)
	if err != nil {
		t.Fatal(err)
	}
	defer occupied.Close()
	if err := runtime.start(); err == nil {
		t.Fatal("occupied statistics port did not reject startup")
	}
	if _, err := os.Lstat(config.StatePath); !os.IsNotExist(err) {
		t.Fatal("failed bind replaced the saved state")
	}
	_ = occupied.Close()
	runtime.config.StatePath = filepath.Join(filepath.Dir(config.StatePath), "missing", "runtime.json")
	if err := runtime.start(); err == nil {
		t.Fatal("initial save failure did not reject startup")
	}
	listener, err := net.Listen("tcp4", config.Listen)
	if err != nil {
		t.Fatalf("initial save failure leaked the HTTP listener: %v", err)
	}
	_ = listener.Close()
	runtime.config.StatePath = config.StatePath
	if err := runtime.start(); err != nil {
		t.Fatal(err)
	}
	if err := runtime.stop(); err != nil {
		t.Fatal(err)
	}
	listener, err = net.Listen("tcp4", config.Listen)
	if err != nil {
		t.Fatalf("immediate stop leaked the HTTP listener: %v", err)
	}
	_ = listener.Close()
}

func TestRuntimeHTTPConcurrentReads(t *testing.T) {
	config := runtimeHTTPConfig(t)
	runtime, up, _ := runtimeFixture(t, config)
	if err := runtime.start(); err != nil {
		t.Fatal(err)
	}
	var workers sync.WaitGroup
	workers.Go(func() {
		for range 30 {
			up.Add(1)
			runtime.sample()
			if err := runtime.save(); err != nil {
				t.Error(err)
			}
		}
	})
	workers.Go(func() {
		for range 30 {
			requestRuntime(t, config, http.MethodGet, "/runtime", config.Token, http.StatusOK)
		}
	})
	workers.Wait()
}
