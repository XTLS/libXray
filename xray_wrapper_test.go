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

func TestRunXrayWithVmess(t *testing.T) {
	// Example VMess configuration
	vmess := `eyJhZGQiOiAiMzguMTY1LjMzLjEyNiIsICJhaWQiOiAiMCIsICJob3N0IjogIiIsICJpZCI6ICJhYjliMWUwZC05YzczLTQxNzYtODE5OS00N2I0OTNhMjJlNGMiLCAibmV0IjogImtjcCIsICJwYXRoIjogIiIsICJwb3J0IjogMjYzODgsICJwcyI6ICIiLCAic2N5IjogIm5vbmUiLCAidGxzIjogIiIsICJ0eXBlIjogIm5vbmUiLCAidiI6ICIyIn0=`

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
	configPath := filepath.Join(projectRoot, "config", "xray_config_textRunXray.json")
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

func TestRunXray(t *testing.T) {
	// Example VMess configuration (same as in the previous test)
	vmess := `eyJhZGQiOiAiMzguMTY1LjMzLjEyNiIsICJhaWQiOiAiMCIsICJob3N0IjogIiIsICJpZCI6ICJhYjliMWUwZC05YzczLTQxNzYtODE5OS00N2I0OTNhMjJlNGMiLCAibmV0IjogImtjcCIsICJwYXRoIjogIiIsICJwb3J0IjogMjYzODgsICJwcyI6ICIiLCAic2N5IjogIm5vbmUiLCAidGxzIjogIiIsICJ0eXBlIjogIm5vbmUiLCAidiI6ICIyIn0=`

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

	// Generate an Xray configuration file (use the same config as before)
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

	// Prepare the paths for configuration
	projectRoot, _ := filepath.Abs(".")
	configPath := filepath.Join(projectRoot, "config", "xray_config_runXray.json")

	// Write the Xray config to a file
	err = writeConfigToFile(xrayConfig, configPath)
	if err != nil {
		t.Fatalf("Failed to write Xray config: %v", err)
	}

	// Create a request for RunXray with the paths and max memory settings
	datDir := filepath.Join(projectRoot, "dat")
	runRequest := libXray.RunXrayRequest{
		DatDir:     datDir,
		ConfigPath: configPath,
		MaxMemory:  1024, // Set max memory limit for the Xray instance
	}

	// Convert the request to base64
	requestBytes, _ := json.Marshal(runRequest)
	base64Request := base64.StdEncoding.EncodeToString(requestBytes)

	// Call RunXray
	response := libXray.RunXray(base64Request)

	// Decode the response from base64
	decodedResponse, err := base64.StdEncoding.DecodeString(response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(decodedResponse, &result); err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Check if the response indicates success
	if success, ok := result["success"].(bool); !ok || !success {
		t.Fatalf("RunXray failed: %v", response)
	}

	t.Log("RunXray test passed successfully", string(decodedResponse))
}
