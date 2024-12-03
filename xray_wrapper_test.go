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
	// Example VMess configuration
	vmess := `xxxx`

	// Decode VMess configuration
	decodedVmess, err := base64.StdEncoding.DecodeString(vmess)
	if err != nil {
		t.Fatalf("Failed to decode VMess: %v", err)
	}

	var vmessConfig map[string]interface{}
	err = json.Unmarshal(decodedVmess, &vmessConfig)
	if err != nil {
		t.Fatalf("Failed to unmarshal VMess: %v", err)
	}

	// Generate an Xray configuration file
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
	// Call TestXray
	request := libXray.TestXrayRequest{
		DatDir:     datDir,
		ConfigPath: configPath,
	}
	requestBytes, _ := json.Marshal(request)
	base64Request := base64.StdEncoding.EncodeToString(requestBytes)

	response := libXray.TestXray(base64Request)
	// Decode Base64 response
	decoded, err := base64.StdEncoding.DecodeString(response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(decoded, &result); err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Check the response content
	if success, ok := result["success"].(bool); !ok || !success {
		t.Fatalf("TestXray failed: %v", response)
	}

	t.Log("TestXray passed successfully", string(decoded))
}

func writeConfigToFile(config map[string]interface{}, path string) error {
	// Ensure the directory exists
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	// Create the configuration file
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()

	// Write the configuration to the file
	encoder := json.NewEncoder(file)
	return encoder.Encode(config)
}
