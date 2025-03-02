package xray

import (
	"os"
	"runtime"
	"runtime/debug"
	"sync/atomic"

	"github.com/xtls/libxray/nodep"
	"github.com/xtls/xray-core/common/cmdarg"
	"github.com/xtls/xray-core/core"
	_ "github.com/xtls/xray-core/main/distro/all"
)

var (
	coreServer atomic.Pointer[core.Instance]
)

func StartXray(configPath string) (*core.Instance, error) {
	// Принудительная очистка перед стартом
	runtime.GC()
	debug.FreeOSMemory()

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

// Run Xray instance.
// datDir means the dir which geosite.dat and geoip.dat are in.
// configPath means the config.json file path.
func RunXray(datDir string, configPath string) (err error) {
	// Останавливаем предыдущий инстанс если есть
	if err = StopXray(); err != nil {
		return
	}

	InitEnv(datDir)
	nodep.InitForceFree()

	server, err := StartXray(configPath)
	if err != nil {
		return
	}

	if err = server.Start(); err != nil {
		return
	}

	coreServer.Store(server)
	return nil
}

// Stop Xray instance.
func StopXray() error {
	oldServer := coreServer.Swap(nil)
	if oldServer != nil {
		err := oldServer.Close()
		if err != nil {
			return err
		}
		// Принудительная очистка после остановки
		runtime.GC()
		debug.FreeOSMemory()
	}
	return nil
}

// Xray's version
func XrayVersion() string {
	return core.Version()
}
