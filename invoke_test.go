package libXray

import (
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/xtls/xray-core/common/platform"
)

type testResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Err     string          `json:"error,omitempty"`
}

func invokeForTest(t *testing.T, method LibXrayMethod, env *LibXrayEnvJson, payload any) testResponse {
	t.Helper()
	rawPayload, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	rawRequest, err := json.Marshal(&LibXrayInvokeRequest{
		APIVersion: 1,
		Method:     method,
		Env:        env,
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

func writeConfigToFile(t *testing.T, config any, path string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := json.NewEncoder(file).Encode(config); err != nil {
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
	projectRoot, _ := filepath.Abs(".")
	configPath := filepath.Join(projectRoot, "config", "xray_config_test.json")
	writeConfigToFile(t, testXrayConfig(t), configPath)

	response := invokeForTest(
		t,
		LibXrayMethodTestXray,
		&LibXrayEnvJson{AssetLocation: filepath.Join(projectRoot, "dat"), CertLocation: filepath.Join(projectRoot, "dat")},
		RunXrayRequest{ConfigPath: configPath},
	)
	if !response.Success {
		t.Fatalf("TestXray failed: %s", response.Err)
	}
	requireNoDataObject(t, response)
}

func TestInvokeRunXray(t *testing.T) {
	projectRoot, _ := filepath.Abs(".")
	configPath := filepath.Join(projectRoot, "config", "xray_config_run.json")
	writeConfigToFile(t, testXrayConfig(t), configPath)

	response := invokeForTest(
		t,
		LibXrayMethodRunXray,
		&LibXrayEnvJson{AssetLocation: filepath.Join(projectRoot, "dat"), CertLocation: filepath.Join(projectRoot, "dat")},
		RunXrayRequest{ConfigPath: configPath},
	)
	defer xrayStopForTest(t)
	if !response.Success {
		t.Fatalf("RunXray failed: %s", response.Err)
	}
	requireNoDataObject(t, response)
}

func TestInvokeXrayVersion(t *testing.T) {
	response := invokeForTest(t, LibXrayMethodXrayVersion, nil, nil)
	if !response.Success {
		t.Fatalf("XrayVersion failed: %s", response.Err)
	}
	version := decodeDataObject[XrayVersionResponse](t, response)
	if version.Version == "" {
		t.Fatal("Xray version should not be empty")
	}
}

func TestInvokeMapResponseShape(t *testing.T) {
	response := invokeForTest(t, LibXrayMethodGetFreePorts, nil, GetFreePortsRequest{Count: 1})
	if !response.Success {
		t.Fatalf("GetFreePorts failed: %s", response.Err)
	}
	ports := decodeDataObject[GetFreePortsResponse](t, response)
	if len(ports.Ports) != 1 {
		t.Fatalf("ports = %v", ports.Ports)
	}

	response = invokeForTest(t, LibXrayMethodGetXrayState, nil, nil)
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
		nil,
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

func TestInvokeEnvOnlySetsProvidedFields(t *testing.T) {
	t.Setenv(platform.AssetLocation, "initial-asset")
	t.Setenv(platform.CertLocation, "initial-cert")
	response := invokeForTest(
		t,
		LibXrayMethodXrayVersion,
		&LibXrayEnvJson{AssetLocation: "updated-asset"},
		nil,
	)
	if !response.Success {
		t.Fatalf("XrayVersion failed: %s", response.Err)
	}
	if got := os.Getenv(platform.AssetLocation); got != "updated-asset" {
		t.Fatalf("asset env = %q", got)
	}
	if got := os.Getenv(platform.CertLocation); got != "initial-cert" {
		t.Fatalf("cert env = %q", got)
	}
}

func TestInvokeSetsAllSupportedEnvFields(t *testing.T) {
	tests := []struct {
		name  string
		key   string
		value string
	}{
		{"config", platform.ConfigLocation, "config-value"},
		{"confdir", platform.ConfdirLocation, "confdir-value"},
		{"asset", platform.AssetLocation, "asset-value"},
		{"cert", platform.CertLocation, "cert-value"},
		{"readv", platform.UseReadV, "readv-value"},
		{"splice", platform.UseFreedomSplice, "splice-value"},
		{"vmess padding", platform.UseVmessPadding, "vmess-padding-value"},
		{"cone", platform.UseCone, "cone-value"},
		{"strict json", platform.UseStrictJSON, "strict-json-value"},
		{"buffer size", platform.BufferSize, "buffer-size-value"},
		{"browser dialer", platform.BrowserDialerAddress, "browser-dialer-value"},
		{"xudp log", platform.XUDPLog, "xudp-log-value"},
		{"xudp base key", platform.XUDPBaseKey, "xudp-base-key-value"},
		{"tun fd", platform.TunFdKey, "123"},
	}
	for _, tt := range tests {
		t.Setenv(tt.key, "")
		t.Run(tt.name, func(t *testing.T) {
			requestJSON := `{"apiVersion":1,"method":"xrayVersion","env":{"` + tt.key + `":"` + tt.value + `"}}`
			var response testResponse
			if err := json.Unmarshal([]byte(Invoke(requestJSON)), &response); err != nil {
				t.Fatal(err)
			}
			if !response.Success {
				t.Fatalf("XrayVersion failed: %s", response.Err)
			}
			if got := os.Getenv(tt.key); got != tt.value {
				t.Fatalf("%s = %q", tt.key, got)
			}
		})
	}
}

func TestInvokeUnknownMethod(t *testing.T) {
	response := invokeForTest(t, LibXrayMethod("unknown"), nil, nil)
	if response.Success {
		t.Fatal("unknown method should fail")
	}
}

func TestInvokeAPIVersion(t *testing.T) {
	response := invokeRawForTest(t, `{"method":"xrayVersion"}`)
	if !response.Success {
		t.Fatalf("omitted apiVersion should default to v1: %s", response.Err)
	}

	t.Setenv(platform.AssetLocation, "initial-asset")
	response = invokeRawForTest(t, `{"apiVersion":2,"method":"xrayVersion","env":{"xray.location.asset":"updated-asset"}}`)
	if response.Success {
		t.Fatal("unsupported apiVersion should fail")
	}
	if got := string(response.Data); got != "null" {
		t.Fatalf("data = %s, want null", got)
	}
	if got := os.Getenv(platform.AssetLocation); got != "initial-asset" {
		t.Fatalf("asset env = %q", got)
	}
}

func TestInvokeNoDataResponseShape(t *testing.T) {
	response := invokeForTest(t, LibXrayMethodStopXray, nil, nil)
	if !response.Success {
		t.Fatalf("StopXray failed: %s", response.Err)
	}
	requireNoDataObject(t, response)

	response = invokeRawForTest(t, `{"apiVersion":1,"method":"runXray","payload":"invalid"}`)
	if response.Success {
		t.Fatal("invalid runXray payload should fail")
	}
	if got := string(response.Data); got != "null" {
		t.Fatalf("data = %s, want null", got)
	}
}

func TestInvokeIgnoresUnknownEnvField(t *testing.T) {
	const key = "XRAY_LIBXRAY_UNKNOWN_ENV_TEST"
	_ = os.Unsetenv(key)
	t.Cleanup(func() { _ = os.Unsetenv(key) })
	requestJSON := `{"apiVersion":1,"method":"xrayVersion","env":{"` + key + `":"/tmp"}}`
	var response testResponse
	if err := json.Unmarshal([]byte(Invoke(requestJSON)), &response); err != nil {
		t.Fatal(err)
	}
	if !response.Success {
		t.Fatalf("unknown env field should be ignored: %s", response.Err)
	}
	if _, found := os.LookupEnv(key); found {
		t.Fatal("unknown env field should not be set")
	}
}

func xrayStopForTest(t *testing.T) {
	t.Helper()
	response := invokeForTest(t, LibXrayMethodStopXray, nil, nil)
	if !response.Success {
		t.Fatalf("StopXray failed: %s", response.Err)
	}
}
