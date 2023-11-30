package xray

import (
	"os"
	"runtime/debug"

	"github.com/xtls/libxray/nodep"
	"github.com/xtls/xray-core/common/cmdarg"
	"github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
)

var (
	coreServer *core.Instance
)

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

func InitEnv(datDir string) {
	os.Setenv("xray.location.asset", datDir)
}

func setMaxMemory(maxMemory int64) {
	nodep.InitForceFree(maxMemory, 1)
}

// Run Xray instance.
// datDir means the dir which geosite.dat and geoip.dat are in.
// configPath means the config.json file path.
// maxMemory means the soft memory limit of golang, see SetMemoryLimit to find more information.
func RunXray(datDir string, configPath string, maxMemory int64) string {
	InitEnv(datDir)
	if maxMemory > 0 {
		setMaxMemory(maxMemory)
	}
	coreServer, err := StartXray(configPath)
	if err != nil {
		return err.Error()
	}

	if err := coreServer.Start(); err != nil {
		return err.Error()
	}

	debug.FreeOSMemory()
	return ""
}

// Stop Xray instance.
func StopXray() string {
	if coreServer != nil {
		err := coreServer.Close()
		coreServer = nil
		if err != nil {
			return err.Error()
		}
	}
	return ""
}

// Xray's version
func XrayVersion() string {
	return core.Version()
}
