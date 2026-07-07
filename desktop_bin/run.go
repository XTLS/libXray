package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	libXray "github.com/xtls/libxray"
)

type invokeResponse struct {
	Success bool   `json:"success"`
	Err     string `json:"error"`
}

func runXray(configPath string) error {
	requestBytes, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var request libXray.LibXrayInvokeRequest
	if err := json.Unmarshal(requestBytes, &request); err != nil {
		return err
	}
	if request.Method != libXray.LibXrayMethodRunXray {
		return fmt.Errorf("unsupported method %q: desktop_bin only supports runXray", request.Method)
	}

	responseText := libXray.Invoke(string(requestBytes))
	var response invokeResponse
	if err := json.Unmarshal([]byte(responseText), &response); err != nil {
		return err
	}
	if !response.Success {
		if response.Err == "" {
			response.Err = "runXray failed"
		}
		return errors.New(response.Err)
	}
	return nil
}

func stopXray() {
	requestBytes, err := json.Marshal(&libXray.LibXrayInvokeRequest{
		APIVersion: 1,
		Method:     libXray.LibXrayMethodStopXray,
	})
	if err != nil {
		return
	}
	_ = libXray.Invoke(string(requestBytes))
}
