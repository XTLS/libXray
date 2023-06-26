package libXray

import (
	"os"
	"runtime/debug"

	"github.com/xtls/xray-core/common/cmdarg"
	"github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
)

var (
	coreServer *core.Instance
)

func startXray(configPath string) (*core.Instance, error) {
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

func initEnv(datDir string) {
	os.Setenv("xray.location.asset", datDir)
}

func setMaxMemory(maxMemory int64) {
	os.Setenv("XRAY_MEMORY_FORCEFREE", "1")
	initForceFree(maxMemory)
}

func RunXray(datDir string, configPath string, maxMemory int64) string {
	initEnv(datDir)
	if maxMemory > 0 {
		setMaxMemory(maxMemory)
	}
	coreServer, err := startXray(configPath)
	if err != nil {
		return err.Error()
	}

	if err := coreServer.Start(); err != nil {
		return err.Error()
	}

	debug.FreeOSMemory()
	return ""
}

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

func XrayVersion() string {
	return core.Version()
}
