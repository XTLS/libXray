// libXray is an Xray wrapper focusing on improving the experience of Xray-core mobile development.
package libXray

import (
	"encoding/json"
	"strconv"

	"github.com/xtls/libxray/xray"
)

// Read geo data and write all codes to text file.
// datDir means the dir which geo dat are in.
// name means the geo dat file name, like "geosite", "geoip"
// geoType must be the value of geoType
func LoadGeoData(datDir string, name string, geoType string) string {
	err := xray.LoadGeoData(datDir, name, geoType)
	return makeCallResponse("", err)
}

// Ping Xray config and find the delay and country code of its outbound.
// datDir means the dir which geosite.dat and geoip.dat are in.
// configPath means the config.json file path.
// timeout means how long the http request will be cancelled if no response, in units of seconds.
// url means the website we use to test speed. "https://www.google.com" is a good choice for most cases.
// proxy means the local http/socks5 proxy, like "socks5://[::1]:1080".
func Ping(datDir string, configPath string, timeout int, url string, proxy string) string {
	delay, err := xray.Ping(datDir, configPath, timeout, url, proxy)
	return makeCallResponse(strconv.FormatInt(delay, 10), err)
}

type queryStatsResponse struct {
	SysStats string `json:"sysStats,omitempty"`
	Stats    string `json:"stats,omitempty"`
}

// query system stats and outbound stats.
// server means The API server address, like "127.0.0.1:8080".
// dir means the dir which result json will be wrote to.
func QueryStats(server string) string {
	sysStats, stats, err := xray.QueryStats(server)
	if err != nil {
		return makeCallResponse("", err)
	}
	var res queryStatsResponse
	res.SysStats = sysStats
	res.Stats = stats
	b, err := json.Marshal(res)
	if err != nil {
		return makeCallResponse("", err)
	}
	return makeCallResponse(string(b), nil)
}

// convert text to uuid
func CustomUUID(text string) string {
	return makeCallResponse(xray.CustomUUID(text), nil)
}

// Test Xray Config.
// datDir means the dir which geosite.dat and geoip.dat are in.
// configPath means the config.json file path.
func TestXray(datDir string, configPath string) string {
	err := xray.TestXray(datDir, configPath)
	return makeCallResponse("", err)
}

// Run Xray instance.
// datDir means the dir which geosite.dat and geoip.dat are in.
// configPath means the config.json file path.
// maxMemory means the soft memory limit of golang, see SetMemoryLimit to find more information.
func RunXray(datDir string, configPath string, maxMemory int64) string {
	err := xray.RunXray(datDir, configPath, maxMemory)
	return makeCallResponse("", err)
}

// Stop Xray instance.
func StopXray() string {
	err := xray.StopXray()
	return makeCallResponse("", err)
}

// Xray's version
func XrayVersion() string {
	return makeCallResponse(xray.XrayVersion(), nil)
}
