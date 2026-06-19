package main

import (
	"encoding/json"
	"os"

	"github.com/xtls/libxray/xray"
)

type runXrayConfig struct {
	// tun
	TunName     string `json:"tunName,omitempty"`
	TunPriority int    `json:"tunPriority,omitempty"`
	EnableIPv6  bool   `json:"enableIPv6,omitempty"`
	// xray
	DatDir     string `json:"datDir,omitempty"`
	ConfigPath string `json:"configPath,omitempty"`
	// metrics
	MetricsPort string `json:"metricsPort,omitempty"`
}

func runXray(configPath string) error {
	configBytes, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	var config runXrayConfig
	err = json.Unmarshal(configBytes, &config)
	if err != nil {
		return err
	}

	err = xray.RunXray(config.DatDir, config.ConfigPath)
	if err != nil {
		return err
	}
	err = initIpRoute(config.TunName, config.TunPriority, config.EnableIPv6)
	if err != nil {
		return err
	}
	return nil
}

func stopXray() {
	xray.StopXray()
}
