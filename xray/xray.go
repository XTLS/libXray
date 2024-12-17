package xray

import (
	"os"
	"runtime/debug"
	"sync"

	"github.com/xtls/libxray/nodep"
	"github.com/xtls/xray-core/common/cmdarg"
	"github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
)

var (
	// Map to store multiple server instances
	serverInstances = make(map[string]*core.Instance)
	serverMutex    sync.RWMutex
)

func StartXray(configPath string, tag string) (*core.Instance, error) {
	file := cmdarg.Arg{configPath}
	config, err := core.LoadConfig("json", file)
	if err != nil {
		return nil, err
	}

	server, err := core.New(config)
	if err != nil {
		return nil, err
	}

	serverMutex.Lock()
	serverInstances[tag] = server
	serverMutex.Unlock()

	return server, nil
}

func InitEnv(datDir string) {
	os.Setenv("xray.location.asset", datDir)
}

func setMaxMemory(maxMemory int64) {
	nodep.InitForceFree(maxMemory, 1)
}

// Run Xray instance with a specific tag
func RunXray(datDir string, configPath string, maxMemory int64, tag string) (err error) {
	InitEnv(datDir)
	if maxMemory > 0 {
		setMaxMemory(maxMemory)
	}
	
	server, err := StartXray(configPath, tag)
	if err != nil {
		return
	}

	if err = server.Start(); err != nil {
		return
	}

	debug.FreeOSMemory()
	return nil
}

// Stop specific Xray instance
func StopXrayInstance(tag string) error {
	serverMutex.Lock()
	defer serverMutex.Unlock()
	
	if server, exists := serverInstances[tag]; exists {
		err := server.Close()
		if err != nil {
			return err
		}
		delete(serverInstances, tag)
	}
	return nil
}

// Stop all Xray instances
func StopXray() error {
	serverMutex.Lock()
	defer serverMutex.Unlock()

	for tag, server := range serverInstances {
		err := server.Close()
		if err != nil {
			return err
		}
		delete(serverInstances, tag)
	}
	return nil
}

// Get specific Xray instance
func GetXrayInstance(tag string) *core.Instance {
	serverMutex.RLock()
	defer serverMutex.RUnlock()
	
	return serverInstances[tag]
}

// Xray's version
func XrayVersion() string {
	return core.Version()
}
