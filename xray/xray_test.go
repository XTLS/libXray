package xray

import (
	"errors"
	"sync"
	"testing"
)

const minimalConfig = `{
  "log": {"loglevel": "none"},
  "inbounds": [],
  "outbounds": [{"protocol": "freedom", "tag": "direct"}]
}`

func TestRunXrayFromJSONRejectsDuplicateStart(t *testing.T) {
	t.Cleanup(func() {
		if err := StopXray(); err != nil {
			t.Errorf("stop xray: %v", err)
		}
	})

	if err := StopXray(); err != nil {
		t.Fatalf("reset xray state: %v", err)
	}
	if err := RunXrayFromJSON(minimalConfig); err != nil {
		t.Fatalf("start xray: %v", err)
	}
	if !GetXrayState() {
		t.Fatal("xray should be running")
	}
	if err := RunXrayFromJSON(minimalConfig); !errors.Is(err, ErrAlreadyRunning) {
		t.Fatalf("duplicate start error = %v, want %v", err, ErrAlreadyRunning)
	}
}

func TestXrayLifecycleConcurrentStateReads(t *testing.T) {
	if err := StopXray(); err != nil {
		t.Fatalf("reset xray state: %v", err)
	}
	if err := RunXrayFromJSON(minimalConfig); err != nil {
		t.Fatalf("start xray: %v", err)
	}

	var readers sync.WaitGroup
	for range 32 {
		readers.Add(1)
		go func() {
			defer readers.Done()
			_ = GetXrayState()
		}()
	}
	readers.Wait()

	if err := StopXray(); err != nil {
		t.Fatalf("stop xray: %v", err)
	}
	if GetXrayState() {
		t.Fatal("xray should be stopped")
	}
}

func TestRunXrayFromJSONFailureDoesNotPublishInstance(t *testing.T) {
	if err := StopXray(); err != nil {
		t.Fatalf("reset xray state: %v", err)
	}
	if err := RunXrayFromJSON(`{"outbounds":[`); err == nil {
		t.Fatal("invalid config should fail")
	}
	if GetXrayState() {
		t.Fatal("failed start must not publish an instance")
	}
	if err := RunXrayFromJSON(minimalConfig); err != nil {
		t.Fatalf("start after failure: %v", err)
	}
	if err := StopXray(); err != nil {
		t.Fatalf("stop xray: %v", err)
	}
}

func TestRunXrayFromJSONSerializesConcurrentStarts(t *testing.T) {
	if err := StopXray(); err != nil {
		t.Fatalf("reset xray state: %v", err)
	}
	t.Cleanup(func() {
		if err := StopXray(); err != nil {
			t.Errorf("stop xray: %v", err)
		}
	})

	const starts = 8
	errorsByStart := make(chan error, starts)
	var starters sync.WaitGroup
	for range starts {
		starters.Add(1)
		go func() {
			defer starters.Done()
			errorsByStart <- RunXrayFromJSON(minimalConfig)
		}()
	}
	starters.Wait()
	close(errorsByStart)

	successes := 0
	duplicates := 0
	for err := range errorsByStart {
		switch {
		case err == nil:
			successes++
		case errors.Is(err, ErrAlreadyRunning):
			duplicates++
		default:
			t.Fatalf("unexpected start error: %v", err)
		}
	}
	if successes != 1 || duplicates != starts-1 {
		t.Fatalf("successes=%d duplicates=%d", successes, duplicates)
	}
}
