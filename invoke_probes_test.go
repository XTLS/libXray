package libXray

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
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

func TestInvokePingZeroDelayWireField(t *testing.T) {
	raw, err := json.Marshal(PingBatchItemResponse{Success: true, Delay: 0})
	if err != nil {
		t.Fatal(err)
	}
	if string(raw) != `{"success":true,"delay":0}` {
		t.Fatalf("zero latency response = %s", raw)
	}
}

func TestInvokePingBatchUsesOnlyLatencyHeadRequests(t *testing.T) {
	var heads, unexpected atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			heads.Add(1)
		} else {
			unexpected.Add(1)
		}
	}))
	defer server.Close()
	request := PingBatchRequest{
		Configs: []PingBatchItemRequest{{XrayJson: `{"outbounds":[{"protocol":"freedom"}]}`}},
		Timeout: 1, URL: server.URL,
	}
	response := invokeForTest(t, LibXrayMethodPingBatch, request)
	if !response.Success {
		t.Fatalf("error = %s", response.Err)
	}
	result := decodeDataObject[PingBatchResponse](t, response).Results[0]
	if !result.Success || result.Error != "" {
		t.Fatalf("result = %+v", result)
	}
	if heads.Load() != 1 || unexpected.Load() != 0 {
		t.Fatalf("requests = HEAD %d, other %d", heads.Load(), unexpected.Load())
	}
}
