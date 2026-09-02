package xray

import (
	"strings"

	"github.com/xtls/xray-core/core"
)

// ValidateXray only builds the configuration; it does not instantiate handlers.
// The core builder can read local assets/certificates and apply root env values.
func ValidateXray(xrayJSON string) error {
	_, err := core.LoadConfig("json", strings.NewReader(xrayJSON))
	return err
}

// Test Xray Config.
// xrayJSON is the serialized Xray JSON configuration.
func TestXray(xrayJSON string) error {
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
