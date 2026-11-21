package xray

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/xtls/libxray/nodep"
	"github.com/xtls/xray-core/infra/conf"
)

func TestReadPingOutboundsIgnoresOtherRootFields(t *testing.T) {
	xrayJSON := `{
		// These fields are deliberately invalid for the full Xray schema.
		"inbounds": "ignored",
		"routing": 42,
		"dns": false,
		"outbounds": [
			{"protocol": "freedom", "tag": "proxy"}
		]
	}`

	outbounds, err := readPingOutbounds(xrayJSON)
	if err != nil {
		t.Fatal(err)
	}
	if len(outbounds) != 1 {
		t.Fatalf("outbounds = %d, want 1", len(outbounds))
	}
	if outbounds[0].Tag != "proxy" {
		t.Fatalf("tag = %q, want proxy", outbounds[0].Tag)
	}
}

func TestPreparePingOutboundsPreservesRealityClientConfig(t *testing.T) {
	xrayJSON := `{
		"outbounds": [{
			"tag": "proxy",
			"protocol": "vless",
			"settings": {
				"address": "example.com",
				"port": 443,
				"id": "41b20a0c-b56f-4bb8-97e1-e5ceb0ab1ba4",
				"encryption": "none"
			},
			"streamSettings": {
				"network": "xhttp",
				"xhttpSettings": {
					"host": "example.com",
					"path": "/xhttp",
					"mode": "auto"
				},
				"security": "reality",
				"realitySettings": {
					"fingerprint": "chrome",
					"serverName": "example.com",
					"password": "itSUyWpOsw4n3_Rc-hK0r_ryzTBGRqHdrPJI-OFuAwM"
				}
			}
		}]
	}`

	outbounds, err := readPingOutbounds(xrayJSON)
	if err != nil {
		t.Fatal(err)
	}
	prepared, targetTag, err := preparePingOutbounds(outbounds, "proxy", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared) != 1 {
		t.Fatalf("outbounds = %d, want 1", len(prepared))
	}
	if targetTag != "__libxray_ping_0_0" {
		t.Fatalf("target tag = %q", targetTag)
	}
}

func TestPreparePingOutboundsSelectsAndRewritesDependencies(t *testing.T) {
	outbounds := []conf.OutboundDetourConfig{
		{
			Protocol: "blackhole",
			Tag:      "unused",
		},
		{
			Protocol: "not-a-protocol",
			Tag:      "unused",
		},
		{
			Protocol: "freedom",
			Tag:      "proxy",
			StreamSetting: &conf.StreamConfig{
				SocketSettings: &conf.SocketConfig{DialerProxy: "fragment"},
			},
		},
		{
			Protocol: "freedom",
			Tag:      "fragment",
		},
	}

	prepared, targetTag, err := preparePingOutbounds(outbounds, "", 7)
	if err != nil {
		t.Fatal(err)
	}
	if len(prepared) != 2 {
		t.Fatalf("prepared outbounds = %d, want 2", len(prepared))
	}
	if targetTag != "__libxray_ping_7_0" {
		t.Fatalf("target tag = %q", targetTag)
	}
	if prepared[0].Tag != targetTag {
		t.Fatalf("root tag = %q, want %q", prepared[0].Tag, targetTag)
	}
	if prepared[1].Tag != "__libxray_ping_7_1" {
		t.Fatalf("dependency tag = %q", prepared[1].Tag)
	}
	if got := prepared[0].StreamSetting.SocketSettings.DialerProxy; got != prepared[1].Tag {
		t.Fatalf("dialerProxy = %q, want %q", got, prepared[1].Tag)
	}
	for _, outbound := range prepared {
		if outbound.Tag == "unused" {
			t.Fatal("unrelated outbound was included")
		}
	}
}

func TestPreparePingOutboundsRejectsRemovedProxySettings(t *testing.T) {
	outbounds, err := readPingOutbounds(`{"outbounds":[
		{"protocol":"freedom","tag":"proxy","proxySettings":{"tag":"transport"}},
		{"protocol":"freedom","tag":"transport"}
	]}`)
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = preparePingOutbounds(outbounds, "proxy", 1)
	if err == nil || !strings.Contains(err.Error(), "proxySettings") {
		t.Fatalf("expected Xray-core to reject proxySettings, got %v", err)
	}
}

func TestPreparePingOutboundsRejectsDependencyCycle(t *testing.T) {
	outbounds := []conf.OutboundDetourConfig{
		{
			Protocol: "freedom",
			Tag:      "proxy",
			StreamSetting: &conf.StreamConfig{
				SocketSettings: &conf.SocketConfig{DialerProxy: "next"},
			},
		},
		{
			Protocol: "freedom",
			Tag:      "next",
			StreamSetting: &conf.StreamConfig{
				SocketSettings: &conf.SocketConfig{DialerProxy: "proxy"},
			},
		},
	}

	_, _, err := preparePingOutbounds(outbounds, "", 0)
	if err == nil || !strings.Contains(err.Error(), "cycle") {
		t.Fatalf("error = %v, want dependency cycle", err)
	}
}

func TestPingBatchRunsRequestsConcurrently(t *testing.T) {
	requestStarted := make(chan struct{}, 2)
	releaseRequests := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(
		response http.ResponseWriter,
		request *http.Request,
	) {
		requestStarted <- struct{}{}
		<-releaseRequests
		response.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	config := `{"outbounds":[{"protocol":"freedom","tag":"proxy"}]}`
	type batchResult struct {
		results []PingBatchResult
		err     error
	}
	done := make(chan batchResult, 1)
	go func() {
		results, err := PingBatch(
			[]PingBatchItem{
				{XrayJSON: config},
				{XrayJSON: config},
			},
			2,
			server.URL,
		)
		done <- batchResult{results: results, err: err}
	}()

	for range 2 {
		select {
		case <-requestStarted:
		case <-time.After(2 * time.Second):
			t.Fatal("requests did not run concurrently")
		}
	}
	close(releaseRequests)

	result := <-done
	if result.err != nil {
		t.Fatal(result.err)
	}
	if len(result.results) != 2 {
		t.Fatalf("results = %d, want 2", len(result.results))
	}
	for index := range result.results {
		if !result.results[index].Success {
			t.Fatalf(
				"result %d failed: %s",
				index,
				result.results[index].Error,
			)
		}
	}
}

func connectionServer(
	handler http.HandlerFunc,
) (*httptest.Server, *atomic.Int32) {
	var connections atomic.Int32
	server := httptest.NewUnstartedServer(handler)
	server.Config.ConnState = func(_ net.Conn, state http.ConnState) {
		if state == http.StateNew {
			connections.Add(1)
		}
	}
	server.Start()
	return server, &connections
}

// delayServer answers the nth request after delays[n], or at once past the end.
func delayServer(
	delays ...time.Duration,
) (server *httptest.Server, requests, connections *atomic.Int32) {
	requests = new(atomic.Int32)
	server, connections = connectionServer(func(
		response http.ResponseWriter,
		request *http.Request,
	) {
		if index := int(requests.Add(1)) - 1; index < len(delays) {
			select {
			case <-time.After(delays[index]):
			case <-request.Context().Done():
			}
		}
		response.WriteHeader(http.StatusNoContent)
	})
	return server, requests, connections
}

func TestPingBatchWarmTimesSecondRequestOnOneConnection(t *testing.T) {
	const firstDelay = 300 * time.Millisecond
	server, _, connections := delayServer(firstDelay)
	defer server.Close()

	results, err := PingBatchWarm(
		[]PingBatchItem{{XrayJSON: minimalConfig}},
		2,
		server.URL,
	)
	if err != nil {
		t.Fatal(err)
	}
	if !results[0].Success {
		t.Fatalf("warm probe failed: %s", results[0].Error)
	}
	if results[0].Delay >= firstDelay.Milliseconds() {
		t.Fatalf("delay = %d, want the second request only", results[0].Delay)
	}
	if got := connections.Load(); got != 1 {
		t.Fatalf("connections = %d, want 1", got)
	}
}

func TestPingBatchWarmFailsWithoutKeepAlive(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(
		response http.ResponseWriter,
		request *http.Request,
	) {
		if requests.Add(1) > 1 {
			<-request.Context().Done()
		}
		response.Header().Set("Connection", "close")
		response.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	results, err := PingBatchWarm(
		[]PingBatchItem{{XrayJSON: minimalConfig}},
		2,
		server.URL,
	)
	if err != nil {
		t.Fatal(err)
	}
	if results[0].Success || results[0].Delay != nodep.PingDelayError {
		t.Fatalf("result = %+v, want an error", results[0])
	}
	if got := requests.Load(); got != 1 {
		t.Fatalf("requests = %d, want 1", got)
	}
}

func TestPingBatchRedirects(t *testing.T) {
	for _, test := range []struct {
		name        string
		ping        func([]PingBatchItem, int, string) ([]PingBatchResult, error)
		location    string
		requests    int32
		connections int32
	}{
		{"default follows on a new connection", PingBatch, "/next", 2, 2},
		{"warm answers with the redirect", PingBatchWarm, "/next", 2, 1},
		{"warm ignores a malformed location", PingBatchWarm, "/%", 2, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			var requests atomic.Int32
			server, connections := connectionServer(func(
				response http.ResponseWriter,
				request *http.Request,
			) {
				requests.Add(1)
				if request.URL.Path == "/" {
					response.Header().Set("Location", test.location)
					response.WriteHeader(http.StatusFound)
					return
				}
				response.WriteHeader(http.StatusNoContent)
			})
			defer server.Close()

			results, err := test.ping(
				[]PingBatchItem{{XrayJSON: minimalConfig}},
				2,
				server.URL+"/",
			)
			if err != nil {
				t.Fatal(err)
			}
			if !results[0].Success {
				t.Fatalf("probe failed: %s", results[0].Error)
			}
			if got := requests.Load(); got != test.requests {
				t.Fatalf("requests = %d, want %d", got, test.requests)
			}
			if got := connections.Load(); got != test.connections {
				t.Fatalf("connections = %d, want %d", got, test.connections)
			}
		})
	}
}

func TestPingBatchSendsURLCredentials(t *testing.T) {
	for _, test := range []struct {
		name string
		ping func([]PingBatchItem, int, string) ([]PingBatchResult, error)
	}{
		{"default", PingBatch},
		{"warm", PingBatchWarm},
	} {
		t.Run(test.name, func(t *testing.T) {
			var anonymous atomic.Int32
			server := httptest.NewServer(http.HandlerFunc(func(
				response http.ResponseWriter,
				request *http.Request,
			) {
				if user, password, _ := request.BasicAuth(); user != "user" || password != "pass" {
					anonymous.Add(1)
				}
				response.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			results, err := test.ping(
				[]PingBatchItem{{XrayJSON: minimalConfig}},
				2,
				strings.Replace(server.URL, "http://", "http://user:pass@", 1),
			)
			if err != nil {
				t.Fatal(err)
			}
			if !results[0].Success {
				t.Fatalf("probe failed: %s", results[0].Error)
			}
			if got := anonymous.Load(); got != 0 {
				t.Fatalf("requests without credentials = %d, want 0", got)
			}
		})
	}
}

func TestPingBatchTimesOutWithinOneTimeout(t *testing.T) {
	for _, test := range []struct {
		name     string
		ping     func([]PingBatchItem, int, string) ([]PingBatchResult, error)
		delays   []time.Duration
		requests int32
	}{
		{"default", PingBatch, []time.Duration{5 * time.Second}, 1},
		// A fresh deadline would let the second request answer in time.
		{
			"warm",
			PingBatchWarm,
			[]time.Duration{1200 * time.Millisecond, 1200 * time.Millisecond},
			2,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			server, requests, connections := delayServer(test.delays...)
			defer server.Close()

			start := time.Now()
			results, err := test.ping(
				[]PingBatchItem{{XrayJSON: minimalConfig}},
				2,
				server.URL,
			)
			if err != nil {
				t.Fatal(err)
			}
			if results[0].Success || results[0].Delay != nodep.PingDelayTimeout {
				t.Fatalf("result = %+v, want a timeout", results[0])
			}
			if elapsed := time.Since(start); elapsed > 2800*time.Millisecond {
				t.Fatalf("batch took %s, want one timeout", elapsed)
			}
			if got := requests.Load(); got != test.requests {
				t.Fatalf("requests = %d, want %d", got, test.requests)
			}
			if got := connections.Load(); got != 1 {
				t.Fatalf("connections = %d, want 1", got)
			}
		})
	}
}

func TestPingBatchKeepsPerItemConfigErrorsInInputOrder(t *testing.T) {
	results, err := PingBatch(
		[]PingBatchItem{
			{XrayJSON: "not JSON"},
			{XrayJSON: `{"outbounds":[]}`},
		},
		1,
		"https://example.com",
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %d, want 2", len(results))
	}
	for index, result := range results {
		if result.Success {
			t.Fatalf("result %d unexpectedly succeeded", index)
		}
		if result.Delay != 10000 {
			t.Fatalf("result %d delay = %d, want 10000", index, result.Delay)
		}
		if result.Error == "" {
			t.Fatalf("result %d has no error", index)
		}
	}
	if results[0].Error == "" {
		t.Fatal("first result should contain a JSON parsing error")
	}
	if results[1].Error != "ping config has no outbounds" {
		t.Fatalf("second result error = %q, want empty outbounds error", results[1].Error)
	}
}

func TestValidatePingBatchRequest(t *testing.T) {
	tests := []struct {
		name      string
		items     []PingBatchItem
		timeout   int
		targetURL string
		errorText string
	}{
		{
			name:      "empty configs",
			timeout:   1,
			targetURL: "https://example.com",
			errorText: "configs are empty",
		},
		{
			name:      "invalid URL",
			items:     []PingBatchItem{{}},
			timeout:   1,
			targetURL: "example.com",
			errorText: "absolute HTTP",
		},
		{
			name: "too many configs",
			items: []PingBatchItem{
				{},
				{},
				{},
				{},
				{},
				{},
			},
			timeout:   1,
			targetURL: "https://example.com",
			errorText: "more than 5 configs",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validatePingBatchRequest(
				test.items,
				test.timeout,
				test.targetURL,
			)
			if err == nil || !strings.Contains(err.Error(), test.errorText) {
				t.Fatalf("error = %v, want %q", err, test.errorText)
			}
		})
	}
}

func TestValidatePingBatchRequestAcceptsFiveConfigs(t *testing.T) {
	err := validatePingBatchRequest(
		[]PingBatchItem{
			{},
			{},
			{},
			{},
			{},
		},
		1,
		"https://example.com",
	)
	if err != nil {
		t.Fatal(err)
	}
}

func ExamplePingBatch() {
	results, _ := PingBatch(
		[]PingBatchItem{{XrayJSON: `{"outbounds":[{"protocol":"freedom"}]}`}},
		5,
		"https://cp.cloudflare.com/",
	)
	fmt.Println(len(results))
}
