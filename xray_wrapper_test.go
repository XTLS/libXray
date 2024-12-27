package libXray_test

import (
	"encoding/base64"
	"encoding/json"
	"github.com/xtls/libxray"
	"os"
	"path/filepath"
	"testing"
)

// writeConfigToFile writes the configuration to a file
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

// decodeVmessConfig decodes and unmarshals the VMess configuration from a base64 string
func decodeVmessConfig(vmess string) (map[string]interface{}, error) {
	// Decode the VMess configuration from base64
	decodedVmess, err := base64.StdEncoding.DecodeString(vmess)
	if err != nil {
		return nil, err
	}

	// Unmarshal the decoded string into a map
	var vmessConfig map[string]interface{}
	err = json.Unmarshal(decodedVmess, &vmessConfig)
	if err != nil {
		return nil, err
	}

	return vmessConfig, nil
}

// createXrayConfig creates an Xray configuration map based on the VMess configuration
func createXrayConfig(vmessConfig map[string]interface{}) map[string]interface{} {
	return map[string]interface{}{
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
}

// base64EncodeRequest encodes a request struct into a base64 string
func base64EncodeRequest(request interface{}) (string, error) {
	// Marshal the request struct into JSON bytes
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	// Encode the JSON bytes to a base64 string
	return base64.StdEncoding.EncodeToString(requestBytes), nil
}

// handleTestResponse decodes and checks the response from Xray
func handleTestResponse(response string, t *testing.T) {
	// Decode the base64 response
	decoded, err := base64.StdEncoding.DecodeString(response)
	if err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Parse the decoded response into a map
	var result map[string]interface{}
	if err := json.Unmarshal(decoded, &result); err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Check if the "success" field is true
	if success, ok := result["success"].(bool); !ok || !success {
		t.Fatalf("TestXray failed: %v", response)
	}

	t.Log("TestXray passed successfully", string(decoded))
}

// TestRunXrayWithVmess tests running Xray with VMess configuration
func TestRunXrayWithVmess(t *testing.T) {
	// Example VMess configuration (base64 encoded)
	vmess := `eyJhZGQiOiAiMzguMTY1LjMzLjEyNiIsICJhaWQiOiAiMCIsICJob3N0IjogIiIsICJpZCI6ICJhYjliMWUwZC05YzczLTQxNzYtODE5OS00N2I0OTNhMjJlNGMiLCAibmV0IjogImtjcCIsICJwYXRoIjogIiIsICJwb3J0IjogMjYzODgsICJwcyI6ICIiLCAic2N5IjogIm5vbmUiLCAidGxzIjogIiIsICJ0eXBlIjogIm5vbmUiLCAidiI6ICIyIn0=`

	// Decode and parse the VMess configuration
	vmessConfig, err := decodeVmessConfig(vmess)
	if err != nil {
		t.Fatalf("Failed to decode VMess: %v", err)
	}

	// Create an Xray configuration from the VMess configuration
	xrayConfig := createXrayConfig(vmessConfig)

	// Prepare the path for the configuration file
	projectRoot, _ := filepath.Abs(".")
	configPath := filepath.Join(projectRoot, "config", "xray_config_test.json")

	// Write the Xray configuration to a file
	err = writeConfigToFile(xrayConfig, configPath)
	if err != nil {
		t.Fatalf("Failed to write Xray config: %v", err)
	}

	// Create a request for testing Xray
	datDir := filepath.Join(projectRoot, "dat")
	request := libXray.TestXrayRequest{
		DatDir:     datDir,
		ConfigPath: configPath,
	}

	// Encode the request to base64
	base64Request, err := base64EncodeRequest(request)
	if err != nil {
		t.Fatalf("Failed to encode request: %v", err)
	}

	// Call TestXray with the base64-encoded request
	response := libXray.TestXray(base64Request)

	// Handle and check the response
	handleTestResponse(response, t)
}

// TestRunXray tests running Xray with a VMess configuration for real-world usage
func TestRunXray(t *testing.T) {
	// Example VMess configuration (same as in the previous test)
	vmess := `eyJhZGQiOiAiMzguMTY1LjMzLjEyNiIsICJhaWQiOiAiMCIsICJob3N0IjogIiIsICJpZCI6ICJhYjliMWUwZC05YzczLTQxNzYtODE5OS00N2I0OTNhMjJlNGMiLCAibmV0IjogImtjcCIsICJwYXRoIjogIiIsICJwb3J0IjogMjYzODgsICJwcyI6ICIiLCAic2N5IjogIm5vbmUiLCAidGxzIjogIiIsICJ0eXBlIjogIm5vbmUiLCAydiI6ICIyIn0=`

	// Decode and parse the VMess configuration
	vmessConfig, err := decodeVmessConfig(vmess)
	if err != nil {
		t.Fatalf("Failed to decode VMess: %v", err)
	}

	// Create an Xray configuration from the VMess configuration
	xrayConfig := createXrayConfig(vmessConfig)

	// Prepare the path for the configuration file
	projectRoot, _ := filepath.Abs(".")
	configPath := filepath.Join(projectRoot, "config", "xray_config_run.json")

	// Write the Xray configuration to a file
	err = writeConfigToFile(xrayConfig, configPath)
	if err != nil {
		t.Fatalf("Failed to write Xray config: %v", err)
	}

	// Create a request for running Xray
	datDir := filepath.Join(projectRoot, "dat")
	runRequest := libXray.RunXrayRequest{
		DatDir:     datDir,
		ConfigPath: configPath,
		MaxMemory:  1024, // Set max memory limit for Xray
	}

	// Encode the request to base64
	base64Request, err := base64EncodeRequest(runRequest)
	if err != nil {
		t.Fatalf("Failed to encode request: %v", err)
	}

	// Call RunXray with the base64-encoded request
	response := libXray.RunXray(base64Request)

	// Handle and check the response
	handleTestResponse(response, t)
}
