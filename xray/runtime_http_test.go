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

func requestRuntime(t *testing.T, config RuntimeConfig, method, path, token, body string, status int) runtimeFiles {
	t.Helper()
	request, err := http.NewRequest(method, "http://"+config.Listen+path, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if token != "" {
		request.Header.Set("Authorization", "Bearer "+token)
	}
	request.Header.Set("Content-Type", "application/json")
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
	var files runtimeFiles
	if status == http.StatusOK {
		if response.Header.Get("Content-Type") != "application/json" || json.Unmarshal(data, &files) != nil || files.Archived == nil {
			t.Fatalf("invalid runtime response: %s", data)
		}
	}
	return files
}

func TestRuntimeHTTPAuthenticationArchivesAcknowledgmentAndStop(t *testing.T) {
	config := runtimeHTTPConfig(t)
	previous, up, down := runtimeFixture(t, config)
	up.Add(5)
	down.Add(7)
	archived := saveRuntimeSample(t, previous)
	_ = previous.stateLock.Close()
	runtime, up, down := runtimeFixture(t, config)
	up.Add(17)
	down.Add(23)
	if err := runtime.start(); err != nil {
		t.Fatal(err)
	}
	requestRuntime(t, config, http.MethodGet, "/runtime", "", "", http.StatusUnauthorized)
	requestRuntime(t, config, http.MethodGet, "/runtime", strings.Repeat("b", 32), "", http.StatusUnauthorized)
	requestRuntime(t, config, http.MethodPost, "/runtime/ack", "", `{"removeSessionIds":["`+archived.Session.ID+`"]}`, http.StatusUnauthorized)
	files := requestRuntime(t, config, http.MethodGet, "/runtime", config.Token, "", http.StatusOK)
	if files.Current == nil || files.Current.Session.Uplink != 17 || files.Current.Session.Downlink != 23 || len(files.Archived) != 1 || files.Archived[0] != archived {
		t.Fatalf("wrong current/archived snapshots: %+v", files)
	}
	current := *files.Current
	up.Add(100)
	files = requestRuntime(t, config, http.MethodGet, "/runtime", config.Token, "", http.StatusOK)
	if *files.Current != current || up.Value() != 117 {
		t.Fatal("HTTP must read saved snapshots without sampling or resetting metrics")
	}
	// A duplicate current archive is never acknowledged while it remains current.
	if err := archiveRuntimeState(config.StatePath, current); err != nil {
		t.Fatal(err)
	}
	body := `{"removeSessionIds":["` + archived.Session.ID + `","` + current.Session.ID + `","` + strings.Repeat("c", 32) + `"]}`
	for range 2 {
		files = requestRuntime(t, config, http.MethodPost, "/runtime/ack", config.Token, body, http.StatusOK)
		if *files.Current != current || len(files.Archived) != 1 || files.Archived[0] != current {
			t.Fatalf("ack was not idempotent or removed current: %+v", files)
		}
	}
	archivePath := filepath.Join(filepath.Dir(config.StatePath), "runtime-sessions", archived.Session.ID+".json")
	if _, err := os.Lstat(archivePath); !os.IsNotExist(err) {
		t.Fatal("acknowledged archive still exists")
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

func TestRuntimeHTTPRejectsInvalidAcknowledgments(t *testing.T) {
	config := runtimeHTTPConfig(t)
	runtime, _, _ := runtimeFixture(t, config)
	if err := runtime.start(); err != nil {
		t.Fatal(err)
	}
	archive := savedRuntime(t, config.StatePath)
	archive.Session.ID = strings.Repeat("b", 32)
	if err := archiveRuntimeState(config.StatePath, archive); err != nil {
		t.Fatal(err)
	}
	for _, body := range []string{
		`{`, `{}`, `{"removeSessionIds":null}`,
		`{"removeSessionIds":["` + archive.Session.ID + `","../../outside"]}`,
		`{"removeSessionIds":["` + strings.Repeat("A", 32) + `"]}`,
		`{"removeSessionIds":[],"path":"outside"}`,
		`{"removeSessionIds":[]} {}`,
		`{"removeSessionIds":[]}` + strings.Repeat(" ", 64*1024),
	} {
		requestRuntime(t, config, http.MethodPost, "/runtime/ack", config.Token, body, http.StatusBadRequest)
	}
	files := requestRuntime(t, config, http.MethodGet, "/runtime", config.Token, "", http.StatusOK)
	if len(files.Archived) != 1 || files.Archived[0] != archive {
		t.Fatal("invalid request partially acknowledged an archive")
	}
	requestRuntime(t, config, http.MethodPost, "/runtime", config.Token, "", http.StatusMethodNotAllowed)
	requestRuntime(t, config, http.MethodGet, "/runtime/ack", config.Token, "", http.StatusMethodNotAllowed)
	requestRuntime(t, config, http.MethodGet, "/control", config.Token, "", http.StatusNotFound)
	if err := os.Remove(config.StatePath); err != nil {
		t.Fatal(err)
	}
	files = requestRuntime(t, config, http.MethodGet, "/runtime", config.Token, "", http.StatusOK)
	if files.Current != nil || len(files.Archived) != 1 {
		t.Fatal("missing current snapshot was not reported as null")
	}
	requestRuntime(t, config, http.MethodPost, "/runtime/ack", config.Token, `{"removeSessionIds":["`+archive.Session.ID+`"]}`, http.StatusServiceUnavailable)
}

func TestRuntimeHTTPRejectsSnapshotSymlinksAndInvalidArchives(t *testing.T) {
	for _, target := range []string{"current", "archive-directory", "archive-file", "archive-id"} {
		t.Run(target, func(t *testing.T) {
			config := runtimeHTTPConfig(t)
			runtime, _, _ := runtimeFixture(t, config)
			if err := runtime.start(); err != nil {
				t.Fatal(err)
			}
			state := savedRuntime(t, config.StatePath)
			state.Session.ID = strings.Repeat("b", 32)
			outside := filepath.Join(t.TempDir(), state.Session.ID+".json")
			if err := writeRuntimeState(outside, state); err != nil {
				t.Fatal(err)
			}
			archive := filepath.Join(filepath.Dir(config.StatePath), "runtime-sessions")
			link, destination := config.StatePath, outside
			switch target {
			case "current":
				if err := os.Remove(config.StatePath); err != nil {
					t.Fatal(err)
				}
			case "archive-directory":
				link, destination = archive, filepath.Dir(outside)
			case "archive-file", "archive-id":
				if err := os.Mkdir(archive, 0700); err != nil {
					t.Fatal(err)
				}
				link = filepath.Join(archive, state.Session.ID+".json")
			}
			if target == "archive-id" {
				invalid := state
				invalid.Session.ID = strings.Repeat("c", 32)
				if err := writeRuntimeState(link, invalid); err != nil {
					t.Fatal(err)
				}
			} else if err := os.Symlink(destination, link); err != nil {
				t.Skipf("symbolic links unavailable: %v", err)
			}
			requestRuntime(t, config, http.MethodGet, "/runtime", config.Token, "", http.StatusServiceUnavailable)
			requestRuntime(t, config, http.MethodPost, "/runtime/ack", config.Token, `{"removeSessionIds":["`+state.Session.ID+`"]}`, http.StatusServiceUnavailable)
			if savedRuntime(t, outside) != state {
				t.Fatal("HTTP changed a snapshot outside its archive")
			}
		})
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

func TestRuntimeHTTPConcurrentReadsAndAcknowledgments(t *testing.T) {
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
			requestRuntime(t, config, http.MethodGet, "/runtime", config.Token, "", http.StatusOK)
		}
	})
	workers.Go(func() {
		for range 30 {
			requestRuntime(t, config, http.MethodPost, "/runtime/ack", config.Token, `{"removeSessionIds":[]}`, http.StatusOK)
		}
	})
	workers.Wait()
}
