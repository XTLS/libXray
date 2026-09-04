package libXray

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeShareStatsResponseShape(t *testing.T) {
	const validLink = "vless://12345678-abcd-abcd-abcd-123456789abc@example.com:443?encryption=none&security=tls&sni=example.com"
	for _, test := range []struct {
		text           string
		usable, failed int
		success        bool
	}{
		{validLink + "\nvless://bad@example.com:443", 1, 1, true},
		{"vless://bad@example.com:443", 0, 1, false},
	} {
		response := invokeForTest(t, LibXrayMethodConvertShareLinksToXrayJson, ConvertShareLinksToXrayJsonRequest{Text: test.text, IncludeStats: true})
		if response.Success != test.success {
			t.Fatalf("success = %v, error = %s", response.Success, response.Err)
		}
		var result ConvertShareLinksToXrayJsonResponse
		if err := json.Unmarshal(response.Data, &result); err != nil {
			t.Fatal(err)
		}
		if result.UsableCount != test.usable || result.FailedCount != test.failed || len(result.Config) == 0 {
			t.Fatalf("result = %+v", result)
		}
		var root map[string]json.RawMessage
		if err := json.Unmarshal(response.Data, &root); err != nil {
			t.Fatal(err)
		}
		if len(root) != 3 || root["config"] == nil || root["usableCount"] == nil || root["failedCount"] == nil {
			t.Fatalf("data = %s", response.Data)
		}
	}
	for _, text := range []string{`{"outbounds":`, "-----BEGIN AGE ENCRYPTED FILE-----\ninvalid"} {
		response := invokeForTest(t, LibXrayMethodConvertShareLinksToXrayJson, ConvertShareLinksToXrayJsonRequest{Text: text, IncludeStats: true})
		if response.Success || string(response.Data) != "null" {
			t.Fatalf("response = %+v", response)
		}
	}
}

func TestInvokeConfigurationURLProbe(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	request := TestXrayRequest{XrayJson: `{"outbounds":[{"protocol":"freedom"}]}`, URL: server.URL, Timeout: 2}
	response := invokeForTest(t, LibXrayMethodTestXray, request)
	if !response.Success {
		t.Fatal(response.Err)
	}
	result := decodeDataObject[TestXrayResponse](t, response)
	if result.Delay < 0 || !strings.Contains(string(response.Data), `"delay"`) {
		t.Fatalf("bad result: %s", response.Data)
	}
	request.BuildOnly = true
	response = invokeForTest(t, LibXrayMethodTestXray, request)
	if response.Success || string(response.Data) != "null" {
		t.Fatalf("ambiguous request accepted: %+v", response)
	}
}

func TestInvokePingLocationAndZeroDelayWireFields(t *testing.T) {
	raw, err := json.Marshal(PingBatchItemResponse{Success: true, Delay: 0})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"success":true,"delay":0}` {
		t.Fatalf("zero latency response = %s", raw)
	}
	empty := ""
	raw, err = json.Marshal(PingBatchItemResponse{Success: true, Delay: 0, LocationJSON: &empty})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"success":true,"delay":0,"locationJson":""}` {
		t.Fatalf("empty location response = %s", raw)
	}
	locationBody := `{"ip_address":"203.0.113.1","country":"SG"}`
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, locationBody)
	}))
	defer server.Close()
	request := PingBatchRequest{
		Configs: []PingBatchItemRequest{{XrayJson: `{"outbounds":[{"protocol":"freedom"}]}`}},
		Timeout: 1, URL: server.URL, LocationURL: server.URL,
	}
	response := invokeForTest(t, LibXrayMethodPingBatch, request)
	if !response.Success {
		t.Fatalf("error = %s", response.Err)
	}
	result := decodeDataObject[PingBatchResponse](t, response).Results[0]
	if !result.Success || result.LocationJSON == nil || *result.LocationJSON != `{"ip_address":"203.0.113.1","country":"SG"}` {
		t.Fatalf("result = %+v", result)
	}
	locationBody = ""
	response = invokeForTest(t, LibXrayMethodPingBatch, request)
	result = decodeDataObject[PingBatchResponse](t, response).Results[0]
	if !response.Success || result.LocationJSON == nil || *result.LocationJSON != "" {
		t.Fatalf("empty location response = %+v", response)
	}
	request.LocationURL = ""
	response = invokeForTest(t, LibXrayMethodPingBatch, request)
	if !response.Success || strings.Contains(string(response.Data), "location") {
		t.Fatalf("latency-only response = %+v", response)
	}
}
