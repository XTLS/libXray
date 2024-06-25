// libXray is an Xray wrapper focusing on improving the experience of Xray-core mobile development.
package libXray

import (
	"encoding/base64"
	"encoding/json"

	"github.com/xtls/libxray/nodep"
	"github.com/xtls/libxray/xray"
)

type loadGeoDataRequest struct {
	DatDir  string `json:"datDir,omitempty"`
	Name    string `json:"name,omitempty"`
	GeoType string `json:"geoType,omitempty"`
}

// Read geo data and write all codes to text file.
// datDir means the dir which geo dat are in.
// name means the geo dat file name, like "geosite", "geoip"
// geoType must be the value of geoType
func LoadGeoData(base64Text string) string {
	var response nodep.CallResponse[string]
	req, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	var request loadGeoDataRequest
	err = json.Unmarshal(req, &request)
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	err = xray.LoadGeoData(request.DatDir, request.Name, request.GeoType)
	return response.EncodeToBase64("", err)
}

type pingRequest struct {
	DatDir     string `json:"datDir,omitempty"`
	ConfigPath string `json:"configPath,omitempty"`
	Timeout    int    `json:"timeout,omitempty"`
	Url        string `json:"url,omitempty"`
	Proxy      string `json:"proxy,omitempty"`
}

// Ping Xray config and find the delay and country code of its outbound.
// datDir means the dir which geosite.dat and geoip.dat are in.
// configPath means the config.json file path.
// timeout means how long the http request will be cancelled if no response, in units of seconds.
// url means the website we use to test speed. "https://www.google.com" is a good choice for most cases.
// proxy means the local http/socks5 proxy, like "socks5://[::1]:1080".
func Ping(base64Text string) string {
	var response nodep.CallResponse[int64]
	req, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64(nodep.PingDelayError, err)
	}
	var request pingRequest
	err = json.Unmarshal(req, &request)
	if err != nil {
		return response.EncodeToBase64(nodep.PingDelayError, err)
	}
	delay, err := xray.Ping(request.DatDir, request.ConfigPath, request.Timeout, request.Url, request.Proxy)
	return response.EncodeToBase64(delay, err)
}

type queryStatsResponse struct {
	SysStats string `json:"sysStats,omitempty"`
	Stats    string `json:"stats,omitempty"`
}

// query system stats and outbound stats.
// server means The API server address, like "127.0.0.1:8080".
// dir means the dir which result json will be wrote to.
func QueryStats(base64Text string) string {
	var response nodep.CallResponse[*queryStatsResponse]
	server, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64(nil, err)
	}

	sysStats, stats, err := xray.QueryStats(string(server))
	if err != nil {
		return response.EncodeToBase64(nil, err)
	}
	var queryStats queryStatsResponse
	queryStats.SysStats = sysStats
	queryStats.Stats = stats
	return response.EncodeToBase64(&queryStats, nil)
}

// convert text to uuid
func CustomUUID(base64Text string) string {
	var response nodep.CallResponse[string]
	text, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	uuid := xray.CustomUUID(string(text))
	return response.EncodeToBase64(uuid, nil)
}

type testXrayRequest struct {
	DatDir     string `json:"datDir,omitempty"`
	ConfigPath string `json:"configPath,omitempty"`
}

// Test Xray Config.
// datDir means the dir which geosite.dat and geoip.dat are in.
// configPath means the config.json file path.
func TestXray(base64Text string) string {
	var response nodep.CallResponse[string]
	req, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	var request testXrayRequest
	err = json.Unmarshal(req, &request)
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	err = xray.TestXray(request.DatDir, request.ConfigPath)
	return response.EncodeToBase64("", err)
}

type runXrayRequest struct {
	DatDir     string `json:"datDir,omitempty"`
	ConfigPath string `json:"configPath,omitempty"`
	MaxMemory  int64  `json:"maxMemory,omitempty"`
}

// Run Xray instance.
// datDir means the dir which geosite.dat and geoip.dat are in.
// configPath means the config.json file path.
// maxMemory means the soft memory limit of golang, see SetMemoryLimit to find more information.
func RunXray(base64Text string) string {
	var response nodep.CallResponse[string]
	req, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	var request runXrayRequest
	err = json.Unmarshal(req, &request)
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	err = xray.RunXray(request.DatDir, request.ConfigPath, request.MaxMemory)
	return response.EncodeToBase64("", err)
}

// Stop Xray instance.
func StopXray() string {
	var response nodep.CallResponse[string]
	err := xray.StopXray()
	return response.EncodeToBase64("", err)
}

// Xray's version
func XrayVersion() string {
	var response nodep.CallResponse[string]
	return response.EncodeToBase64(xray.XrayVersion(), nil)
}
