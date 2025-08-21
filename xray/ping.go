package xray

import (
	"runtime/debug"

	"github.com/xtls/libxray/memory"
	"github.com/xtls/libxray/nodep"
)

func Ping(datDir string, configPath string, timeout int, url string, proxy string) (int64, error) {
	InitEnv(datDir)
	memory.InitForceFree()
	server, err := StartXray(configPath)
	if err != nil {
		return nodep.PingDelayError, err
	}

	if err := server.Start(); err != nil {
		return nodep.PingDelayError, err
	}
	defer func() {
		server.Close()
		debug.FreeOSMemory()
	}()

	delay, err := nodep.MeasureDelay(timeout, url, proxy)
	if err != nil {
		return delay, err
	}

	return delay, nil
}

func PingTCP(datDir string, configPath string, timeout int, host string, port int, proxy string) (int64, error) {
	InitEnv(datDir)
	memory.InitForceFree()
	server, err := StartXray(configPath)
	if err != nil {
		return nodep.PingDelayError, err
	}

	if err := server.Start(); err != nil {
		return nodep.PingDelayError, err
	}
	defer func() {
		server.Close()
		debug.FreeOSMemory()
	}()

	delay, err := nodep.MeasureTCPDelay(timeout, host, port, proxy)
	if err != nil {
		return delay, err
	}

	return delay, nil
}

func Connect(datDir string, configPath string, timeout int, targetHost string, targetPort int, proxy string) (int64, error) {
	InitEnv(datDir)
	memory.InitForceFree()
	server, err := StartXray(configPath)
	if err != nil {
		return nodep.PingDelayError, err
	}

	if err := server.Start(); err != nil {
		return nodep.PingDelayError, err
	}
	defer func() {
		server.Close()
		debug.FreeOSMemory()
	}()

	delay, err := nodep.MeasureProxyConnectDelay(timeout, targetHost, targetPort, proxy)
	if err != nil {
		return delay, err
	}

	return delay, nil
}

func PingFromJSON(datDir string, configJSON string, timeout int, url string, proxy string) (int64, error) {
	InitEnv(datDir)
	memory.InitForceFree()
	server, err := StartXrayFromJSON(configJSON)
	if err != nil {
		return nodep.PingDelayError, err
	}

	if err := server.Start(); err != nil {
		return nodep.PingDelayError, err
	}
	defer func() {
		server.Close()
		debug.FreeOSMemory()
	}()

	delay, err := nodep.MeasureDelay(timeout, url, proxy)
	if err != nil {
		return delay, err
	}

	return delay, nil
}

func PingTCPFromJSON(datDir string, configJSON string, timeout int, host string, port int, proxy string) (int64, error) {
	InitEnv(datDir)
	memory.InitForceFree()
	server, err := StartXrayFromJSON(configJSON)
	if err != nil {
		return nodep.PingDelayError, err
	}

	if err := server.Start(); err != nil {
		return nodep.PingDelayError, err
	}
	defer func() {
		server.Close()
		debug.FreeOSMemory()
	}()

	delay, err := nodep.MeasureTCPDelay(timeout, host, port, proxy)
	if err != nil {
		return delay, err
	}

	return delay, nil
}

func ConnectFromJSON(datDir string, configJSON string, timeout int, targetHost string, targetPort int, proxy string) (int64, error) {
	InitEnv(datDir)
	memory.InitForceFree()
	server, err := StartXrayFromJSON(configJSON)
	if err != nil {
		return nodep.PingDelayError, err
	}

	if err := server.Start(); err != nil {
		return nodep.PingDelayError, err
	}
	defer func() {
		server.Close()
		debug.FreeOSMemory()
	}()

	delay, err := nodep.MeasureProxyConnectDelay(timeout, targetHost, targetPort, proxy)
	if err != nil {
		return delay, err
	}

	return delay, nil
}
