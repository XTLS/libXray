package xray

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestPingBatchLocationUsesEachForcedOutbound(t *testing.T) {
	var heads, gets atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			heads.Add(1)
			return
		}
		gets.Add(1)
		fmt.Fprint(w, `{"ip_address":"203.0.113.9","country":"jp"}`)
	}))
	defer server.Close()
	results, err := PingBatchWithLocation([]PingBatchItem{
		{XrayJSON: `{"outbounds":[{"protocol":"freedom","tag":"proxy"}]}`},
		{XrayJSON: `{"outbounds":[{"protocol":"blackhole","tag":"proxy"}]}`},
	}, 1, server.URL, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	first, second := results[0], results[1]
	if !first.Success || first.LocationJSON == nil || *first.LocationJSON != `{"ip_address":"203.0.113.9","country":"jp"}` || first.LocationError != "" {
		t.Fatalf("first result = %+v", first)
	}
	if second.Success || second.LocationJSON != nil || second.LocationError == "" {
		t.Fatalf("blocked outbound escaped through another client: %+v", second)
	}
	if heads.Load() != 1 || gets.Load() != 1 {
		t.Fatalf("requests = HEAD %d GET %d", heads.Load(), gets.Load())
	}
}

func TestPingBatchLocationFailureDoesNotFailLatency(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusServiceUnavailable)
		}
	}))
	defer server.Close()
	items := []PingBatchItem{{XrayJSON: `{"outbounds":[{"protocol":"freedom"}]}`}}
	results, err := PingBatchWithLocation(items, 1, server.URL, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	result := results[0]
	if !result.Success || result.Error != "" || result.LocationJSON != nil || result.LocationError != "location request returned HTTP 503" {
		t.Fatalf("result = %+v", result)
	}
	results, err = PingBatch(items, 1, server.URL)
	if err != nil {
		t.Fatal(err)
	}
		if !results[0].Success || results[0].LocationJSON != nil || results[0].LocationError != "" {
			t.Fatalf("latency-only result = %+v", results[0])
	}
}

func TestPingBatchLocationCanSucceedWhenLatencyFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				conn.Close()
			}
			return
		}
		fmt.Fprint(w, `{"ip_address":"2001:db8::1","country":"US"}`)
	}))
	defer server.Close()
	results, err := PingBatchWithLocation([]PingBatchItem{{XrayJSON: `{"outbounds":[{"protocol":"freedom"}]}`}}, 1, server.URL, server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Success || results[0].LocationJSON == nil || *results[0].LocationJSON != `{"ip_address":"2001:db8::1","country":"US"}` || results[0].LocationError != "" {
		t.Fatalf("result = %+v", results[0])
	}
}

func TestProbeLocationReturnsOriginalBody(t *testing.T) {
	for _, body := range []string{"", " \n{\"provider_specific\": true}\n", strings.Repeat("x", 64*1024)} {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, body) }))
		locationJSON, err := probeLocation(server.Client(), server.URL)
		server.Close()
		if err != nil || locationJSON != body {
			t.Fatalf("location JSON length = %d, error = %v", len(locationJSON), err)
		}
	}
}

func TestProbeLocationRejectsOversizedResponseWithoutLeakingInput(t *testing.T) {
	body := "private-source" + strings.Repeat("x", 64*1024+1-len("private-source"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, body)
	}))
	defer server.Close()
	locationJSON, err := probeLocation(server.Client(), server.URL)
	if err == nil || locationJSON != "" {
		t.Fatalf("location JSON = %q, error = %v", locationJSON, err)
	}
	if strings.Contains(err.Error(), "private-source") {
		t.Fatal("response body leaked")
	}
}

func TestProbeLocationHidesURLAndCredentialsOnRequestFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { <-r.Context().Done() }))
	defer server.Close()
	client := server.Client()
	client.Timeout = 5 * time.Millisecond
	_, err := probeLocation(client, strings.Replace(server.URL, "://", "://private-user:private-password@", 1))
	if err == nil || err.Error() != "location request failed" {
		t.Fatalf("error = %v", err)
	}
}

func TestPingBatchRejectsInvalidLocationURL(t *testing.T) {
	_, err := PingBatchWithLocation([]PingBatchItem{{}}, 1, "https://example.com", "file:///private/file")
	if err == nil || !strings.Contains(err.Error(), "location URL") {
		t.Fatalf("error = %v", err)
	}
}
