package xray

import (
	"errors"
	"strings"

	"github.com/xtls/xray-core/core"
)

// ValidateXray only builds the configuration; it does not instantiate handlers.
// The core builder can read local assets/certificates and apply root env values.
func ValidateXray(xrayJSON string) error {
	coreServerMu.Lock()
	defer coreServerMu.Unlock()
	if coreServer != nil {
		return errors.New("validateXray requires an isolated process without a managed Xray instance")
	}
	_, err := core.LoadConfig("json", strings.NewReader(xrayJSON))
	return err
}

// Test Xray Config.
// xrayJSON is the serialized Xray JSON configuration.
func TestXray(xrayJSON string) error {
	coreServerMu.Lock()
	defer coreServerMu.Unlock()
	if coreServer != nil {
		return errors.New("testXray requires an isolated process without a managed Xray instance")
	}
	server, err := newXrayInstance(xrayJSON)
	if err != nil {
		return err
	}
	err = server.Close()
	if err != nil {
		return err
	}
	return nil
}
