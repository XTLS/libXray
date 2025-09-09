package main

import (
	"encoding/json"
	"os"

	"github.com/xtls/libxray/dns"
	"github.com/xtls/libxray/xray"
)

type runXrayConfig struct {
	// tun
	TunName     string `json:"tunName,omitempty"`
	TunPriority int    `json:"tunPriority,omitempty"`
	// dns
	Dns           string `json:"dns,omitempty"`
	BindInterface string `json:"bindInterface,omitempty"`
	// xray
	DatDir     string `json:"datDir,omitempty"`
	ConfigPath string `json:"configPath,omitempty"`
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
	err = initIpRoute(config.TunName, config.TunPriority)
	if err != nil {
		return err
	}
	dns.InitDns(config.Dns, config.BindInterface)
	return xray.RunXray(config.DatDir, config.ConfigPath)
}
