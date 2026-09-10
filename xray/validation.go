package xray

import (
	"errors"
)

// TestXray constructs and closes an instance without calling Start.
// Constructors may change process state and acquire resources. Callers own any
// configuration projection; startup resources and connectivity are not tested.
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
	return server.Close()
}
