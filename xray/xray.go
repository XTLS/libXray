package xray

import (
	"errors"
	"runtime/debug"
	"strings"
	"sync"

	"github.com/xtls/libxray/memory"
	"github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
)

var (
	coreServerMu sync.Mutex
	coreServer   *core.Instance
	coreRuntime  *managedRuntime
)

var ErrAlreadyRunning = errors.New("xray is already running")

func newXrayInstance(xrayJSON string) (*core.Instance, error) {
	config, err := core.LoadConfig("json", strings.NewReader(xrayJSON))
	if err != nil {
		return nil, err
	}

	server, err := core.New(config)
	if err != nil {
		return nil, err
	}

	return server, nil
}

// Run Xray instance.
// xrayJSON is the serialized Xray JSON configuration.
func RunXray(xrayJSON string) (err error) {
	return RunXrayWithRuntime(xrayJSON, nil)
}

// RunXrayWithRuntime optionally saves this session's raw inbound counters.
func RunXrayWithRuntime(xrayJSON string, config *RuntimeConfig) (err error) {
	coreServerMu.Lock()
	defer coreServerMu.Unlock()
	if coreServer != nil {
		return ErrAlreadyRunning
	}
	runtime, err := prepareRuntime(config)
	if err != nil {
		return err
	}
	if runtime != nil {
		defer func() {
			if err != nil {
				_ = runtime.stateLock.Close()
			}
		}()
	}

	memory.InitForceFree()
	server, err := newXrayInstance(xrayJSON)
	if err != nil {
		return
	}
	if runtime != nil {
		if err = runtime.attach(server); err != nil {
			_ = server.Close()
			return err
		}
	}

	if err = server.Start(); err != nil {
		_ = server.Close()
		return
	}
	if runtime != nil {
		if err = runtime.start(); err != nil {
			_ = server.Close()
			return err
		}
	}
	coreServer = server
	coreRuntime = runtime

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
		var runtimeErr error
		if coreRuntime != nil {
			defer coreRuntime.stateLock.Close()
			runtimeErr = coreRuntime.stop()
			coreRuntime = nil
		}
		err := errors.Join(runtimeErr, coreServer.Close())
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
