package xray

import (
	"os"
	"runtime/debug"

	"github.com/xtls/libxray/memory"
	"github.com/xtls/xray-core/common/cmdarg"
	"github.com/xtls/xray-core/common/platform"
	"github.com/xtls/xray-core/core"
	"github.com/xtls/xray-core/infra/conf/serial"
	"github.com/xtls/xray-core/main/commands/base"
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

func InitEnv(datDir string, mphCachePath string) {
	os.Setenv(platform.AssetLocation, datDir)
	os.Setenv(platform.CertLocation, datDir)

	if mphCachePath != "" {
		os.Setenv(platform.MphCachePath, mphCachePath)
	}
}

// Run Xray instance.
// datDir means the dir which geosite.dat and geoip.dat are in.
// mphCachePath means the path of mph cache file. leave it empty if you don't use mph cache.
// configPath means the config.json file path.
func RunXray(datDir string, mphCachePath string, configPath string) (err error) {
	InitEnv(datDir, mphCachePath)
	memory.InitForceFree()
	coreServer, err = StartXray(configPath)
	if err != nil {
		return
	}

	if err = coreServer.Start(); err != nil {
		return
	}

	debug.FreeOSMemory()
	return nil
}

// Run Xray instance with JSON configuration string.
// datDir means the dir which geosite.dat and geoip.dat are in.
// mphCachePath means the path of mph cache file. leave it empty if you don't use mph cache.
// configJSON means the JSON configuration string.
func RunXrayFromJSON(datDir string, mphCachePath string, configJSON string) (err error) {
	InitEnv(datDir, mphCachePath)
	memory.InitForceFree()
	coreServer, err = StartXrayFromJSON(configJSON)
	if err != nil {
		return
	}

	debug.FreeOSMemory()
	return nil
}

// Get Xray State
func GetXrayState() bool {
	return coreServer != nil && coreServer.IsRunning()
}

// Stop Xray instance.
func StopXray() error {
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

// https://github.com/XTLS/Xray-core/blob/main/main/commands/all/buildmphcache.go
func BuildMphCache(datDir string, mphCachePath string, configPath string) error {
	InitEnv(datDir, "")
	cf, err := os.Open(configPath)
	if err != nil {
		base.Fatalf("failed to open config file: %v", err)
	}
	defer cf.Close()

	config, err := serial.DecodeJSONConfig(cf)
	if err != nil {
		base.Fatalf("failed to decode config file: %v", err)
		return err
	}

	if err := config.BuildMPHCache(&mphCachePath); err != nil {
		base.Fatalf("failed to build MPH cache: %v", err)
		return err
	}
	return nil
}
