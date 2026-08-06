package libXray

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/metacubex/age"
	"github.com/metacubex/age/armor"
	"github.com/xtls/libxray/nodep"
	"github.com/xtls/xray-core/common/geodata"
	"github.com/xtls/xray-core/infra/conf"
	"google.golang.org/protobuf/proto"
)

type testResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Err     string          `json:"error,omitempty"`
}

func invokeForTest(t *testing.T, method LibXrayMethod, payload any) testResponse {
	t.Helper()
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	rawRequest, err := json.Marshal(&LibXrayInvokeRequest{
		APIVersion: LibXrayAPIVersion,
		Method:     method,
		Payload:    rawPayload,
	})
	if err != nil {
		t.Fatal(err)
	}
	var response testResponse
	if err := json.Unmarshal([]byte(Invoke(string(rawRequest))), &response); err != nil {
		t.Fatal(err)
	}
	return response
}

func invokeRawForTest(t *testing.T, requestJSON string) testResponse {
	t.Helper()
	var response testResponse
	if err := json.Unmarshal([]byte(Invoke(requestJSON)), &response); err != nil {
		t.Fatal(err)
	}
	return response
}

func requireNoDataObject(t *testing.T, response testResponse) {
	t.Helper()
	if got := string(response.Data); got != "{}" {
		t.Fatalf("data = %s, want {}", got)
	}
}

func decodeDataObject[T any](t *testing.T, response testResponse) T {
	t.Helper()
	if !json.Valid(response.Data) {
		t.Fatalf("data is not valid JSON: %s", response.Data)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(response.Data, &object); err != nil {
		t.Fatalf("data is not an object: %s", response.Data)
	}
	var value T
	if err := json.Unmarshal(response.Data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func writeGeoSiteDatForTest(t *testing.T, path string) {
	t.Helper()
	data, err := proto.Marshal(&geodata.GeoSiteList{
		Entry: []*geodata.GeoSite{
			{
				Code: "TEST",
				Domain: []*geodata.Domain{
					{
						Type:  geodata.Domain_Domain,
						Value: "example.com",
						Attribute: []*geodata.Domain_Attribute{
							{
								Key: "ads",
								TypedValue: &geodata.Domain_Attribute_BoolValue{
									BoolValue: true,
								},
							},
						},
					},
				},
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func testVmessPayloadBase64() string {
	const vmessJSON = `{"add":"127.0.0.1","port":1080,"id":"00000000-0000-0000-0000-000000000000","aid":0,"scy":"auto"}`
	return base64.StdEncoding.EncodeToString([]byte(vmessJSON))
}

func testXrayConfig(t *testing.T) any {
	t.Helper()
	decoded, err := base64.StdEncoding.DecodeString(testVmessPayloadBase64())
	if err != nil {
		t.Fatal(err)
	}
	var vmessConfig struct {
		Add  string `json:"add"`
		Port int    `json:"port"`
		ID   string `json:"id"`
		AID  int    `json:"aid"`
		Scy  string `json:"scy"`
	}
	if err := json.Unmarshal(decoded, &vmessConfig); err != nil {
		t.Fatal(err)
	}
	return struct {
		Log       any   `json:"log"`
		Inbounds  []any `json:"inbounds"`
		Outbounds []any `json:"outbounds"`
	}{
		Log: struct {
			Loglevel string `json:"loglevel"`
		}{Loglevel: "debug"},
		Inbounds: []any{
			struct {
				Port     int    `json:"port"`
				Protocol string `json:"protocol"`
				Settings any    `json:"settings"`
			}{
				Port:     1080,
				Protocol: "socks",
				Settings: struct {
					Auth string `json:"auth"`
				}{Auth: "noauth"},
			},
		},
		Outbounds: []any{
			struct {
				Protocol string `json:"protocol"`
				Settings any    `json:"settings"`
			}{
				Protocol: "vmess",
				Settings: struct {
					Vnext []any `json:"vnext"`
				}{
					Vnext: []any{
						struct {
							Address string `json:"address"`
							Port    int    `json:"port"`
							Users   []any  `json:"users"`
						}{
							Address: vmessConfig.Add,
							Port:    vmessConfig.Port,
							Users: []any{
								struct {
									ID       string `json:"id"`
									AlterID  int    `json:"alterId"`
									Security string `json:"security"`
								}{
									ID:       vmessConfig.ID,
									AlterID:  vmessConfig.AID,
									Security: vmessConfig.Scy,
								},
							},
						},
					},
				},
			},
		},
	}
}

func TestInvokeTestXray(t *testing.T) {
	xrayJSON, err := json.Marshal(testXrayConfig(t))
	if err != nil {
		t.Fatal(err)
	}

	response := invokeForTest(
		t,
		LibXrayMethodTestXray,
		TestXrayRequest{XrayJson: string(xrayJSON)},
	)
	if !response.Success {
		t.Fatalf("TestXray failed: %s", response.Err)
	}
	requireNoDataObject(t, response)
}

func TestInvokeTestXrayDoesNotReadConfigPath(t *testing.T) {
	configPath := filepath.Join(t.TempDir(), "xray.json")
	configJSON, err := json.Marshal(testXrayConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, configJSON, 0o600); err != nil {
		t.Fatal(err)
	}

	response := invokeForTest(
		t,
		LibXrayMethodTestXray,
		TestXrayRequest{XrayJson: configPath},
	)
	if response.Success {
		t.Fatal("testXray should parse xrayJson instead of reading a path")
	}
}

func TestInvokeRunXray(t *testing.T) {
	xrayJSON, err := json.Marshal(testXrayConfig(t))
	if err != nil {
		t.Fatal(err)
	}

	response := invokeForTest(
		t,
		LibXrayMethodRunXray,
		RunXrayRequest{XrayJson: string(xrayJSON)},
	)
	defer xrayStopForTest(t)
	if !response.Success {
		t.Fatalf("RunXray failed: %s", response.Err)
	}
	requireNoDataObject(t, response)
}

func TestInvokeRunXrayAppliesConfigEnv(t *testing.T) {
	const key = "XRAY_LIBXRAY_CONFIG_ENV_TEST"
	t.Setenv(key, "")

	rawConfig, err := json.Marshal(testXrayConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	var config map[string]any
	if err := json.Unmarshal(rawConfig, &config); err != nil {
		t.Fatal(err)
	}
	config["env"] = map[string]string{key: "configured"}

	xrayJSON, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	response := invokeForTest(
		t,
		LibXrayMethodRunXray,
		RunXrayRequest{XrayJson: string(xrayJSON)},
	)
	defer xrayStopForTest(t)
	if !response.Success {
		t.Fatalf("RunXray failed: %s", response.Err)
	}
	if got := os.Getenv(key); got != "configured" {
		t.Fatalf("config env = %q, want configured", got)
	}
}

func TestInvokeXrayVersion(t *testing.T) {
	response := invokeForTest(t, LibXrayMethodXrayVersion, nil)
	if !response.Success {
		t.Fatalf("XrayVersion failed: %s", response.Err)
	}
	version := decodeDataObject[XrayVersionResponse](t, response)
	if version.Version == "" {
		t.Fatal("Xray version should not be empty")
	}
}

func TestInvokeMapResponseShape(t *testing.T) {
	response := invokeForTest(t, LibXrayMethodGetFreePorts, GetFreePortsRequest{Count: 1})
	if !response.Success {
		t.Fatalf("GetFreePorts failed: %s", response.Err)
	}
	ports := decodeDataObject[GetFreePortsResponse](t, response)
	if len(ports.Ports) != 1 {
		t.Fatalf("ports = %v", ports.Ports)
	}

	response = invokeForTest(t, LibXrayMethodGetXrayState, nil)
	if !response.Success {
		t.Fatalf("GetXrayState failed: %s", response.Err)
	}
	_ = decodeDataObject[GetXrayStateResponse](t, response)

	rawConfig, err := json.Marshal(testXrayConfig(t))
	if err != nil {
		t.Fatal(err)
	}
	response = invokeForTest(
		t,
		LibXrayMethodConvertXrayJsonToShareLinks,
		ConvertXrayJsonToShareLinksRequest{XrayJson: string(rawConfig)},
	)
	if !response.Success {
		t.Fatalf("ConvertXrayJsonToShareLinks failed: %s", response.Err)
	}
	links := decodeDataObject[ConvertXrayJsonToShareLinksResponse](t, response)
	if links.Links == "" {
		t.Fatal("links should not be empty")
	}
}

func TestInvokeConvertShareLinksFiltersBuildInvalidOutbounds(t *testing.T) {
	const validName = "Valid"
	links := "vless://2418d087-648k-4990-86e8-19dca1d006d3@invalid.example:443?encryption=none&security=tls&sni=invalid.example&fp=chrome\n" +
		"vless://12345678-abcd-abcd-abcd-123456789abc@valid.example:443?encryption=none&security=tls&sni=valid.example&fp=chrome#" + validName

	response := invokeForTest(
		t,
		LibXrayMethodConvertShareLinksToXrayJson,
		ConvertShareLinksToXrayJsonRequest{Text: links},
	)
	if !response.Success {
		t.Fatalf("ConvertShareLinksToXrayJson failed: %s", response.Err)
	}
	config := decodeDataObject[conf.Config](t, response)
	if len(config.OutboundConfigs) != 1 {
		t.Fatalf("outbounds = %d, want 1", len(config.OutboundConfigs))
	}
	if config.OutboundConfigs[0].SendThrough == nil || *config.OutboundConfigs[0].SendThrough != validName {
		t.Fatalf("sendThrough = %v, want %q", config.OutboundConfigs[0].SendThrough, validName)
	}
}

func TestInvokeAgeKeyGenerationAndConversion(t *testing.T) {
	generated := invokeForTest(
		t,
		LibXrayMethodGenerateAgeKeyPair,
		GenerateAgeKeyPairRequest{KeyType: AgeKeyTypeX25519},
	)
	if !generated.Success {
		t.Fatalf("GenerateAgeKeyPair failed: %s", generated.Err)
	}
	pair := decodeDataObject[GenerateAgeKeyPairResponse](t, generated)
	if pair.SecretKey == "" || pair.PublicKey == "" {
		t.Fatalf("generated pair is incomplete: %+v", pair)
	}

	recipient, err := age.ParseX25519Recipient(pair.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	var encrypted bytes.Buffer
	armored := armor.NewWriter(&encrypted)
	writer, err := age.Encrypt(armored, recipient)
	if err != nil {
		t.Fatal(err)
	}
	const link = "vless://12345678-abcd-abcd-abcd-123456789abc@example.com:443?encryption=none&security=tls&sni=example.com#AgeInvoke"
	if _, err := io.WriteString(writer, link); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := armored.Close(); err != nil {
		t.Fatal(err)
	}

	converted := invokeForTest(
		t,
		LibXrayMethodConvertShareLinksToXrayJson,
		ConvertShareLinksToXrayJsonRequest{
			Text: encrypted.String(),
			Age:  &AgeDecryptConfig{SecretKey: pair.SecretKey},
		},
	)
	if !converted.Success {
		t.Fatalf("ConvertShareLinksToXrayJson failed: %s", converted.Err)
	}
	config := decodeDataObject[conf.Config](t, converted)
	if len(config.OutboundConfigs) != 1 {
		t.Fatalf("outbounds = %d, want 1", len(config.OutboundConfigs))
	}
}

func TestInvokeAgeFailuresHaveNullData(t *testing.T) {
	for _, test := range []struct {
		method  LibXrayMethod
		payload any
	}{
		{
			method:  LibXrayMethodGenerateAgeKeyPair,
			payload: GenerateAgeKeyPairRequest{KeyType: AgeKeyType("invalid")},
		},
		{
			method: LibXrayMethodConvertShareLinksToXrayJson,
			payload: ConvertShareLinksToXrayJsonRequest{
				Text: "-----BEGIN AGE ENCRYPTED FILE-----\ninvalid",
			},
		},
	} {
		response := invokeForTest(t, test.method, test.payload)
		if response.Success {
			t.Fatalf("method %q unexpectedly succeeded", test.method)
		}
		if got := string(response.Data); got != "null" {
			t.Fatalf("method %q data = %s, want null", test.method, got)
		}
		if response.Err == "" {
			t.Fatalf("method %q returned no error", test.method)
		}
	}
}

func TestInvokeConvertShareLinksFailsWhenAllOutboundsAreBuildInvalid(t *testing.T) {
	response := invokeForTest(
		t,
		LibXrayMethodConvertShareLinksToXrayJson,
		ConvertShareLinksToXrayJsonRequest{
			Text: "vless://2418d087-648k-4990-86e8-19dca1d006d3@invalid.example:443?encryption=none&security=tls&sni=invalid.example&fp=chrome",
		},
	)
	if response.Success {
		t.Fatal("ConvertShareLinksToXrayJson should fail when all outbounds are invalid")
	}
	if !strings.Contains(response.Err, "no valid outbound found") {
		t.Fatalf("error = %q", response.Err)
	}
	if got := string(response.Data); got != "null" {
		t.Fatalf("data = %s, want null", got)
	}
}

func TestInvokePingBatchReturnsPerItemFailures(t *testing.T) {
	response := invokeForTest(
		t,
		LibXrayMethodPingBatch,
		PingBatchRequest{
			Configs: []PingBatchItemRequest{
				{
					XrayJson: "not JSON",
				},
			},
			Timeout: 1,
			URL:     "https://example.com",
		},
	)
	if !response.Success {
		t.Fatalf("PingBatch failed: %s", response.Err)
	}
	batch := decodeDataObject[PingBatchResponse](t, response)
	if len(batch.Results) != 1 {
		t.Fatalf("results = %d, want 1", len(batch.Results))
	}
	result := batch.Results[0]
	if result.Success {
		t.Fatal("missing config unexpectedly succeeded")
	}
	if result.Delay != nodep.PingDelayError {
		t.Fatalf("delay = %d, want %d", result.Delay, nodep.PingDelayError)
	}
	if result.Error == "" {
		t.Fatal("missing config has no error")
	}
}

func TestInvokePingBatchRejectsInvalidRequest(t *testing.T) {
	response := invokeForTest(
		t,
		LibXrayMethodPingBatch,
		PingBatchRequest{
			Timeout: 1,
			URL:     "https://example.com",
		},
	)
	if response.Success {
		t.Fatal("PingBatch should reject an empty config list")
	}
	if !strings.Contains(response.Err, "configs are empty") {
		t.Fatalf("error = %q", response.Err)
	}
	if got := string(response.Data); got != "null" {
		t.Fatalf("data = %s, want null", got)
	}
}

func TestInvokePingBatchRejectsMoreThanFiveConfigs(t *testing.T) {
	configs := make([]PingBatchItemRequest, 6)
	for i := range configs {
		configs[i] = PingBatchItemRequest{
			XrayJson: `{"outbounds":[{"protocol":"freedom"}]}`,
		}
	}
	response := invokeForTest(
		t,
		LibXrayMethodPingBatch,
		PingBatchRequest{
			Configs: configs,
			Timeout: 1,
			URL:     "https://example.com",
		},
	)
	if response.Success {
		t.Fatal("PingBatch should reject more than five configs")
	}
	if !strings.Contains(response.Err, "more than 5 configs") {
		t.Fatalf("error = %q", response.Err)
	}
	if got := string(response.Data); got != "null" {
		t.Fatalf("data = %s, want null", got)
	}
}

func TestInvokeCountGeoDataUsesPayloadDatDir(t *testing.T) {
	datDir := t.TempDir()
	writeGeoSiteDatForTest(t, filepath.Join(datDir, "geosite.dat"))

	response := invokeForTest(
		t,
		LibXrayMethodCountGeoData,
		CountGeoDataRequest{
			Name:    "geosite",
			GeoType: "domain",
			DatDir:  datDir,
		},
	)
	if !response.Success {
		t.Fatalf("CountGeoData failed: %s", response.Err)
	}
	requireNoDataObject(t, response)
	output, err := os.ReadFile(filepath.Join(datDir, "geosite.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !json.Valid(output) {
		t.Fatalf("geosite.json is not valid JSON: %s", output)
	}
	var list struct {
		CategoryCount int `json:"categoryCount,omitempty"`
		RuleCount     int `json:"ruleCount,omitempty"`
	}
	if err := json.Unmarshal(output, &list); err != nil {
		t.Fatal(err)
	}
	if list.CategoryCount != 1 || list.RuleCount != 1 {
		t.Fatalf("unexpected geosite counts: %+v", list)
	}
}

func TestInvokeUnknownMethod(t *testing.T) {
	response := invokeForTest(t, LibXrayMethod("unknown"), nil)
	if response.Success {
		t.Fatal("unknown method should fail")
	}
}

func TestInvokeRemovedMethods(t *testing.T) {
	for _, method := range []string{"ping", "runXrayFromJson", "deriveAgePublicKey"} {
		response := invokeRawForTest(
			t,
			`{"apiVersion":2,"method":"`+method+`","payload":{}}`,
		)
		if response.Success {
			t.Fatalf("removed method %q should fail", method)
		}
		if response.Err != "unknown method" {
			t.Fatalf("method %q error = %q, want unknown method", method, response.Err)
		}
	}
}

func TestInvokeRejectsOversizedRequest(t *testing.T) {
	response := invokeRawForTest(t, strings.Repeat(" ", maxInvokeJSONBytes+1))
	if response.Success {
		t.Fatal("oversized request should fail")
	}
	if response.Err != "invoke request exceeds the 16 MiB size limit" {
		t.Fatalf("error = %q", response.Err)
	}
	if got := string(response.Data); got != "null" {
		t.Fatalf("data = %s, want null", got)
	}
}

func TestInvokeRejectsOversizedResponse(t *testing.T) {
	responseText := encodeInvokeResponse(
		struct {
			Text string `json:"text"`
		}{Text: strings.Repeat("x", maxInvokeJSONBytes)},
		nil,
	)
	var response testResponse
	if err := json.Unmarshal([]byte(responseText), &response); err != nil {
		t.Fatal(err)
	}
	if response.Success {
		t.Fatal("oversized response should fail")
	}
	if response.Err != "invoke response exceeds the 16 MiB size limit" {
		t.Fatalf("error = %q", response.Err)
	}
	if got := string(response.Data); got != "null" {
		t.Fatalf("data = %s, want null", got)
	}
}

func TestInvokeAPIVersion(t *testing.T) {
	response := invokeRawForTest(t, `{"method":"xrayVersion"}`)
	if response.Success {
		t.Fatal("omitted apiVersion should fail")
	}

	response = invokeRawForTest(t, `{"apiVersion":1,"method":"xrayVersion"}`)
	if response.Success {
		t.Fatal("v1 apiVersion should fail")
	}
	if got := string(response.Data); got != "null" {
		t.Fatalf("data = %s, want null", got)
	}

	response = invokeRawForTest(t, `{"apiVersion":2,"method":"xrayVersion"}`)
	if !response.Success {
		t.Fatalf("v2 apiVersion should succeed: %s", response.Err)
	}
}

func TestInvokeNoDataResponseShape(t *testing.T) {
	response := invokeForTest(t, LibXrayMethodStopXray, nil)
	if !response.Success {
		t.Fatalf("StopXray failed: %s", response.Err)
	}
	requireNoDataObject(t, response)

	response = invokeRawForTest(t, `{"apiVersion":2,"method":"runXray","payload":"invalid"}`)
	if response.Success {
		t.Fatal("invalid runXray payload should fail")
	}
	if got := string(response.Data); got != "null" {
		t.Fatalf("data = %s, want null", got)
	}
}

func TestInvokeIgnoresTopLevelEnv(t *testing.T) {
	const key = "XRAY_LIBXRAY_UNKNOWN_ENV_TEST"
	_ = os.Unsetenv(key)
	t.Cleanup(func() { _ = os.Unsetenv(key) })
	requestJSON := `{"apiVersion":2,"method":"xrayVersion","env":{"` + key + `":"/tmp"}}`
	var response testResponse
	if err := json.Unmarshal([]byte(Invoke(requestJSON)), &response); err != nil {
		t.Fatal(err)
	}
	if !response.Success {
		t.Fatalf("top-level env should be ignored: %s", response.Err)
	}
	if _, found := os.LookupEnv(key); found {
		t.Fatal("top-level env should not be set")
	}
}

func xrayStopForTest(t *testing.T) {
	t.Helper()
	response := invokeForTest(t, LibXrayMethodStopXray, nil)
	if !response.Success {
		t.Fatalf("StopXray failed: %s", response.Err)
	}
}
