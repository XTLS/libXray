package xray

import (
	"errors"
	"runtime/debug"
	"sync"

	"github.com/xtls/libxray/memory"
	"github.com/xtls/xray-core/common/cmdarg"
	"github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
)

var (
	coreServerMu sync.Mutex
	coreServer   *core.Instance
)

var ErrAlreadyRunning = errors.New("xray is already running")

func StartXray(configPath string) (*core.Instance, error) {
	file := cmdarg.Arg{configPath}
	config, err := core.LoadConfig("json", file)
	if err != nil {
		return nil, err
	}

	server, err := core.New(config)
	if err != nil {
		return nil, err
	}

	return server, nil
}

func StartXrayFromJSON(configJSON string) (*core.Instance, error) {
	// Convert JSON string to bytes
	configBytes := []byte(configJSON)

	// Use core.StartInstance which can load configuration directly from bytes
	server, err := core.StartInstance("json", configBytes)
	if err != nil {
		return nil, err
	}

	return server, nil
}

// Run Xray instance.
// configPath means the config.json file path.
func RunXray(configPath string) (err error) {
	coreServerMu.Lock()
	defer coreServerMu.Unlock()
	if coreServer != nil {
		return ErrAlreadyRunning
	}

	memory.InitForceFree()
	server, err := StartXray(configPath)
	if err != nil {
		return
	}

	if err = server.Start(); err != nil {
		_ = server.Close()
		return
	}
	coreServer = server

	debug.FreeOSMemory()
	return nil
}

// Run Xray instance with JSON configuration string.
// configJSON means the JSON configuration string.
func RunXrayFromJSON(configJSON string) (err error) {
	coreServerMu.Lock()
	defer coreServerMu.Unlock()
	if coreServer != nil {
		return ErrAlreadyRunning
	}

	memory.InitForceFree()
	server, err := StartXrayFromJSON(configJSON)
	if err != nil {
		return
	}
	coreServer = server

	debug.FreeOSMemory()
	return nil
}

// Get Xray State
func GetXrayState() bool {
	coreServerMu.Lock()
	defer coreServerMu.Unlock()
	return coreServer != nil && coreServer.IsRunning()
}

// Stop Xray instance.
func StopXray() error {
	coreServerMu.Lock()
	defer coreServerMu.Unlock()
	if coreServer != nil {
		err := coreServer.Close()
		coreServer = nil
		if err != nil {
			return err
		}
	}
	return nil
}

// Xray's version
func XrayVersion() string {
	return core.Version()
}
