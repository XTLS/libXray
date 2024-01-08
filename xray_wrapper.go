// libXray is an Xray wrapper focusing on improving the experience of Xray-core mobile development.
package libXray

import (
	"github.com/xtls/libxray/nodep"
	"github.com/xtls/libxray/xray"
)

// Read geo data and cut the codes we need.
// datDir means the dir which geo dat are in.
// dstDir means the dir which new geo dat are in.
// cutCodePath means geoCutCode json file path
//
// This function is used to reduce memory when init instance.
// You can cut the country codes which rules and nameservers contain.
func CutGeoData(datDir string, dstDir string, cutCodePath string) string {
	err := xray.CutGeoData(datDir, dstDir, cutCodePath)
	return nodep.WrapError(err)
}

// Read geo data and write all codes to text file.
// datDir means the dir which geo dat are in.
// name means the geo dat file name, like "geosite", "geoip"
// geoType must be the value of geoType
func LoadGeoData(datDir string, name string, geoType string) string {
	err := xray.LoadGeoData(datDir, name, geoType)
	return nodep.WrapError(err)
}

// Ping Xray config and find the delay and country code of its outbound.
// datDir means the dir which geosite.dat and geoip.dat are in.
// configPath means the config.json file path.
// timeout means how long the http request will be cancelled if no response, in units of seconds.
// url means the website we use to test speed. "https://www.google.com" is a good choice for most cases.
// proxy means the local http/socks5 proxy, like "socks5://[::1]:1080".
func Ping(datDir string, configPath string, timeout int, url string, proxy string) string {
	return xray.Ping(datDir, configPath, timeout, url, proxy)
}

// query system stats and outbound stats.
// server means The API server address, like "127.0.0.1:8080".
// dir means the dir which result json will be wrote to.
func QueryStats(server string, dir string) string {
	err := xray.QueryStats(server, dir)
	return nodep.WrapError(err)
}

// convert text to uuid
func CustomUUID(text string) string {
	return xray.CustomUUID(text)
}

// Test Xray Config.
// datDir means the dir which geosite.dat and geoip.dat are in.
// configPath means the config.json file path.
func TestXray(datDir string, configPath string) string {
	err := xray.TestXray(datDir, configPath)
	return nodep.WrapError(err)
}

// Run Xray instance.
// datDir means the dir which geosite.dat and geoip.dat are in.
// configPath means the config.json file path.
// maxMemory means the soft memory limit of golang, see SetMemoryLimit to find more information.
func RunXray(datDir string, configPath string, maxMemory int64) string {
	err := xray.RunXray(datDir, configPath, maxMemory)
	return nodep.WrapError(err)
}

// Stop Xray instance.
func StopXray() string {
	err := xray.StopXray()
	return nodep.WrapError(err)
}

// Xray's version
func XrayVersion() string {
	return xray.XrayVersion()
}
