package libXray

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestInvokeShareOutboundsResponseShape(t *testing.T) {
	const validLink = "vless://12345678-abcd-abcd-abcd-123456789abc@example.com:443?encryption=none&security=tls&sni=example.com"
	response := invokeForTest(t, LibXrayMethodConvertShareLinksToXrayJson, ConvertShareLinksToXrayJsonRequest{
		Text: validLink + "\nvless://bad@example.com:443",
	})
	if !response.Success {
		t.Fatal(response.Err)
	}
	var root map[string]json.RawMessage
	if err := json.Unmarshal(response.Data, &root); err != nil {
		t.Fatal(err)
	}
	if len(root) != 1 || root["outbounds"] == nil {
		t.Fatalf("data = %s, want only outbounds", response.Data)
	}
	if config := decodeShareConfig(t, response); len(config.OutboundConfigs) != 1 {
		t.Fatalf("outbounds = %d, want 1", len(config.OutboundConfigs))
	}
	for _, text := range []string{"vless://bad@example.com:443", `{"outbounds":[]}`, `{"outbounds":`, "-----BEGIN AGE ENCRYPTED FILE-----\ninvalid"} {
		response := invokeForTest(t, LibXrayMethodConvertShareLinksToXrayJson, ConvertShareLinksToXrayJsonRequest{Text: text})
		if response.Success || string(response.Data) != "null" || response.Err == "" {
			t.Fatalf("response = %+v", response)
		}
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
