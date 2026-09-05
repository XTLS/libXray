package xray

import (
	"errors"
	"strings"

	"github.com/xtls/xray-core/core"
)

// TestXray only builds the configuration; it does not instantiate handlers.
// The core builder can read local assets/certificates and apply root env values.
// Success does not guarantee that the configuration can start.
func TestXray(xrayJSON string) error {
	coreServerMu.Lock()
	defer coreServerMu.Unlock()
	if coreServer != nil {
		return errors.New("testXray requires an isolated process without a managed Xray instance")
	}
	_, err := core.LoadConfig("json", strings.NewReader(xrayJSON))
	return err
}
