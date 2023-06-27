package libXray

import (
	"fmt"

	"github.com/xtls/libxray/nodep"
)

// Ping Xray config and find the delay and country code of its outbound.
// datDir means the dir which geosite.dat and geoip.dat are in.
// configPath means the config.json file path.
// timeout means how long the http request will be cancelled if no response, in units of seconds.
// url means the website we use to test speed. "https://www.google.com/gen_204" is a good choice for most cases.
// times means how many times we should test the url.
// proxy means the local http/socks5 proxy, like "http://127.0.0.1:1080".
//
// note: you must use http protocol as inbound.
func Ping(datDir string, configPath string, timeout int, url string, times int, proxy string) string {
	initEnv(datDir)
	server, err := startXray(configPath)
	if err != nil {
		return fmt.Sprintf("%d::%s", nodep.PingDelayError, err)
	}

	if err := server.Start(); err != nil {
		return fmt.Sprintf("%d::%s", nodep.PingDelayError, err)
	}
	defer server.Close()

	return nodep.MeasureDelay(timeout, url, times, proxy)
}
