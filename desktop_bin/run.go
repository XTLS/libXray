package main

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/xtls/libxray/xray"
)

type runXrayConfig struct {
	DatDir     string `json:"datDir,omitempty"`
	ConfigPath string `json:"configPath,omitempty"`
}

func runXray(configPath string) error {
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	var config runXrayConfig
	if err := json.Unmarshal(configBytes, &config); err != nil {
		return err
	}
	if config.DatDir == "" {
		return fmt.Errorf("datDir is required")
	}
	if config.ConfigPath == "" {
		return fmt.Errorf("configPath is required")
	}

	return xray.RunXray(config.DatDir, config.ConfigPath)
}

func stopXray() {
	_ = xray.StopXray()
}
