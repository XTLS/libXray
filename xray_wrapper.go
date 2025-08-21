// libXray is an Xray wrapper focusing on improving the experience of Xray-core mobile development.
package libXray

import (
	"encoding/base64"
	"encoding/json"

	"github.com/xtls/libxray/geo"
	"github.com/xtls/libxray/nodep"
	"github.com/xtls/libxray/xray"
)

type CountGeoDataRequest struct {
	DatDir  string `json:"datDir,omitempty"`
	Name    string `json:"name,omitempty"`
	GeoType string `json:"geoType,omitempty"`
}

// Read geo data and write all codes to text file.
func CountGeoData(base64Text string) string {
	var response nodep.CallResponse[string]
	req, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	var request CountGeoDataRequest
	err = json.Unmarshal(req, &request)
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	err = geo.CountGeoData(request.DatDir, request.Name, request.GeoType)
	return response.EncodeToBase64("", err)
}

type ThinGeoDataRequest struct {
	DatDir     string `json:"datDir,omitempty"`
	ConfigPath string `json:"configPath,omitempty"`
	DstDir     string `json:"dstDir,omitempty"`
}

// thin geo data
func ThinGeoData(base64Text string) string {
	var response nodep.CallResponse[string]
	req, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	var request ThinGeoDataRequest
	err = json.Unmarshal(req, &request)
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	err = geo.ThinGeoData(request.DatDir, request.ConfigPath, request.DstDir)
	return response.EncodeToBase64("", err)
}

type readGeoFilesResponse struct {
	Domain []string `json:"domain,omitempty"`
	IP     []string `json:"ip,omitempty"`
}

// thin geo data
func ReadGeoFiles(base64Text string) string {
	var response nodep.CallResponse[*readGeoFilesResponse]
	xray, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64(nil, err)
	}
	domain, ip := geo.ReadGeoFiles(xray)
	var resp readGeoFilesResponse
	resp.Domain = domain
	resp.IP = ip
	return response.EncodeToBase64(&resp, nil)
}

type pingRequest struct {
	DatDir     string `json:"datDir,omitempty"`
	ConfigPath string `json:"configPath,omitempty"`
	Timeout    int    `json:"timeout,omitempty"`
	Url        string `json:"url,omitempty"`
	Proxy      string `json:"proxy,omitempty"`
}

type pingTCPRequest struct {
	DatDir     string `json:"datDir,omitempty"`
	ConfigPath string `json:"configPath,omitempty"`
	Timeout    int    `json:"timeout,omitempty"`
	Host       string `json:"host,omitempty"`
	Port       int    `json:"port,omitempty"`
	Proxy      string `json:"proxy,omitempty"`
}

type connectRequest struct {
	DatDir     string `json:"datDir,omitempty"`
	ConfigPath string `json:"configPath,omitempty"`
	Timeout    int    `json:"timeout,omitempty"`
	TargetHost string `json:"targetHost,omitempty"`
	TargetPort int    `json:"targetPort,omitempty"`
	Proxy      string `json:"proxy,omitempty"`
}

type pingJSONRequest struct {
	DatDir     string `json:"datDir,omitempty"`
	ConfigJSON string `json:"configJSON,omitempty"`
	Timeout    int    `json:"timeout,omitempty"`
	Url        string `json:"url,omitempty"`
	Proxy      string `json:"proxy,omitempty"`
}

type pingTCPJSONRequest struct {
	DatDir     string `json:"datDir,omitempty"`
	ConfigJSON string `json:"configJSON,omitempty"`
	Timeout    int    `json:"timeout,omitempty"`
	Host       string `json:"host,omitempty"`
	Port       int    `json:"port,omitempty"`
	Proxy      string `json:"proxy,omitempty"`
}

type connectJSONRequest struct {
	DatDir     string `json:"datDir,omitempty"`
	ConfigJSON string `json:"configJSON,omitempty"`
	Timeout    int    `json:"timeout,omitempty"`
	TargetHost string `json:"targetHost,omitempty"`
	TargetPort int    `json:"targetPort,omitempty"`
	Proxy      string `json:"proxy,omitempty"`
}

// Ping Xray config and get the delay of its outbound.
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

// PingTCP Xray config and get the TCP delay of its outbound.
func PingTCP(base64Text string) string {
	var response nodep.CallResponse[int64]
	req, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64(nodep.PingDelayError, err)
	}
	var request pingTCPRequest
	err = json.Unmarshal(req, &request)
	if err != nil {
		return response.EncodeToBase64(nodep.PingDelayError, err)
	}
	delay, err := xray.PingTCP(request.DatDir, request.ConfigPath, request.Timeout, request.Host, request.Port, request.Proxy)
	return response.EncodeToBase64(delay, err)
}

// Connect Xray proxy server and get the connection delay.
func Connect(base64Text string) string {
	var response nodep.CallResponse[int64]
	req, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64(nodep.PingDelayError, err)
	}
	var request connectRequest
	err = json.Unmarshal(req, &request)
	if err != nil {
		return response.EncodeToBase64(nodep.PingDelayError, err)
	}
	delay, err := xray.Connect(request.DatDir, request.ConfigPath, request.Timeout, request.TargetHost, request.TargetPort, request.Proxy)
	return response.EncodeToBase64(delay, err)
}

func PingFromJSON(base64Text string) string {
	var response nodep.CallResponse[int64]
	req, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64(nodep.PingDelayError, err)
	}
	var request pingJSONRequest
	err = json.Unmarshal(req, &request)
	if err != nil {
		return response.EncodeToBase64(nodep.PingDelayError, err)
	}
	delay, err := xray.PingFromJSON(request.DatDir, request.ConfigJSON, request.Timeout, request.Url, request.Proxy)
	return response.EncodeToBase64(delay, err)
}

func PingTCPFromJSON(base64Text string) string {
	var response nodep.CallResponse[int64]
	req, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64(nodep.PingDelayError, err)
	}
	var request pingTCPJSONRequest
	err = json.Unmarshal(req, &request)
	if err != nil {
		return response.EncodeToBase64(nodep.PingDelayError, err)
	}
	delay, err := xray.PingTCPFromJSON(request.DatDir, request.ConfigJSON, request.Timeout, request.Host, request.Port, request.Proxy)
	return response.EncodeToBase64(delay, err)
}

func ConnectFromJSON(base64Text string) string {
	var response nodep.CallResponse[int64]
	req, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64(nodep.PingDelayError, err)
	}
	var request connectJSONRequest
	err = json.Unmarshal(req, &request)
	if err != nil {
		return response.EncodeToBase64(nodep.PingDelayError, err)
	}
	delay, err := xray.ConnectFromJSON(request.DatDir, request.ConfigJSON, request.Timeout, request.TargetHost, request.TargetPort, request.Proxy)
	return response.EncodeToBase64(delay, err)
}

// query inbound and outbound stats.
func QueryStats(base64Text string) string {
	var response nodep.CallResponse[string]
	server, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64("", err)
	}

	stats, err := xray.QueryStats(string(server))
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	return response.EncodeToBase64(stats, nil)
}

type TestXrayRequest struct {
	DatDir     string `json:"datDir,omitempty"`
	ConfigPath string `json:"configPath,omitempty"`
}

// Test Xray Config.
func TestXray(base64Text string) string {
	var response nodep.CallResponse[string]
	req, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	var request TestXrayRequest
	err = json.Unmarshal(req, &request)
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	err = xray.TestXray(request.DatDir, request.ConfigPath)
	return response.EncodeToBase64("", err)
}

type RunXrayRequest struct {
	DatDir     string `json:"datDir,omitempty"`
	ConfigPath string `json:"configPath,omitempty"`
}

type RunXrayFromJSONRequest struct {
	DatDir     string `json:"datDir,omitempty"`
	ConfigJSON string `json:"configJSON,omitempty"`
}

// Create Xray Run Request
func NewXrayRunRequest(datDir, configPath string) (string, error) {
	request := RunXrayRequest{
		DatDir:     datDir,
		ConfigPath: configPath,
	}
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	// Encode the JSON bytes to a base64 string
	return base64.StdEncoding.EncodeToString(requestBytes), nil
}

// Create Xray Run From JSON Request
func NewXrayRunFromJSONRequest(datDir, configJSON string) (string, error) {
	request := RunXrayFromJSONRequest{
		DatDir:     datDir,
		ConfigJSON: configJSON,
	}
	requestBytes, err := json.Marshal(request)
	if err != nil {
		return "", err
	}

	// Encode the JSON bytes to a base64 string
	return base64.StdEncoding.EncodeToString(requestBytes), nil
}

// Run Xray instance.
func RunXray(base64Text string) string {
	var response nodep.CallResponse[string]
	req, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	var request RunXrayRequest
	err = json.Unmarshal(req, &request)
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	err = xray.RunXray(request.DatDir, request.ConfigPath)
	return response.EncodeToBase64("", err)
}

// Run Xray instance with JSON configuration.
func RunXrayFromJSON(base64Text string) string {
	var response nodep.CallResponse[string]
	req, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	var request RunXrayFromJSONRequest
	err = json.Unmarshal(req, &request)
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	err = xray.RunXrayFromJSON(request.DatDir, request.ConfigJSON)
	return response.EncodeToBase64("", err)
}

// Get Xray State
func GetXrayState() bool {
	return xray.GetXrayState()
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
