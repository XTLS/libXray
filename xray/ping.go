package xray

import (
	"fmt"

	"github.com/xtls/libxray/nodep"
)

// Ping Xray config and find the delay and country code of its outbound.
// datDir means the dir which geosite.dat and geoip.dat are in.
// configPath means the config.json file path.
// timeout means how long the http request will be cancelled if no response, in units of seconds.
// url means the website we use to test speed. "https://www.google.com" is a good choice for most cases.
// proxy means the local http/socks5 proxy, like "socks5://[::1]:1080".
func Ping(datDir string, configPath string, timeout int, url string, proxy string) string {
	InitEnv(datDir)
	server, err := StartXray(configPath)
	if err != nil {
		return fmt.Sprintf("%d:%s", nodep.PingDelayError, err)
	}

	if err := server.Start(); err != nil {
		return fmt.Sprintf("%d:%s", nodep.PingDelayError, err)
	}
	defer server.Close()

	delay, err := nodep.MeasureDelay(timeout, url, proxy)
	if err != nil {
		return fmt.Sprintf("%d:%s", delay, err)
	}

	return fmt.Sprintf("%d:", delay)
}
