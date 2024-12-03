package libXray_test

import (
	"encoding/base64"
	libXray "github.com/xtls/libxray"
	"os"
	"path/filepath"
	"testing"
)

import (
	"encoding/json"
)

func TestRunXrayWithVmess(t *testing.T) {
	// 示例 vmess 配置
	vmess := `eyJhZGQiOiAiMzguMTY1LjMzLjEyNiIsICJhaWQiOiAiMCIsICJob3N0IjogIiIsICJpZCI6ICJhYjliMWUwZC05YzczLTQxNzYtODE5OS00N2I0OTNhMjJlNGMiLCAibmV0IjogImtjcCIsICJwYXRoIjogIiIsICJwb3J0IjogMjYzODgsICJwcyI6ICIiLCAic2N5IjogIm5vbmUiLCAidGxzIjogIiIsICJ0eXBlIjogIm5vbmUiLCAidiI6ICIyIn0=`

	// 解码 vmess 配置
	decodedVmess, err := base64.StdEncoding.DecodeString(vmess)
	if err != nil {
		t.Fatalf("Failed to decode vmess: %v", err)
	}

	var vmessConfig map[string]interface{}
	err = json.Unmarshal(decodedVmess, &vmessConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal vmess: %v", err)
	}

	// 生成一个 Xray 配置文件
	xrayConfig := map[string]interface{}{
		"log": map[string]interface{}{
			"loglevel": "debug",
		},
		"inbounds": []map[string]interface{}{
			{
				"port":     1080,
				"protocol": "socks",
				"settings": map[string]interface{}{
					"auth": "noauth",
				},
			},
		},
		"outbounds": []map[string]interface{}{
			{
				"protocol": "vmess",
				"settings": map[string]interface{}{
					"vnext": []map[string]interface{}{
						{
							"address": vmessConfig["add"],
							"port":    vmessConfig["port"],
							"users": []map[string]interface{}{
								{
									"id":       vmessConfig["id"],
									"alterId":  vmessConfig["aid"],
									"security": vmessConfig["scy"],
								},
							},
						},
					},
				},
			},
		},
	}

	projectRoot, _ := filepath.Abs(".")
	configPath := filepath.Join(projectRoot, "config", "xray_config.json")
	err = writeConfigToFile(xrayConfig, configPath)
	if err != nil {
		t.Fatalf("Failed to write Xray config: %v", err)
	}

	datDir := filepath.Join(projectRoot, "dat")
	// 调用 TestXray
	request := libXray.TestXrayRequest{
		DatDir:     datDir,
		ConfigPath: configPath,
	}
	requestBytes, _ := json.Marshal(request)
	base64Request := base64.StdEncoding.EncodeToString(requestBytes)

	response := libXray.TestXray(base64Request)
	// 解码 Base64
	decoded, err := base64.StdEncoding.DecodeString(response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(decoded, &result); err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// 检查返回内容
	if success, ok := result["success"].(bool); !ok || !success {
		t.Fatalf("TestXray failed: %v", response)
	}

	t.Log("TestXray passed successfully", string(decoded))
}

func writeConfigToFile(config map[string]interface{}, path string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(config)
}
