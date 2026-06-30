// libXray is an Xray wrapper focusing on improving the experience of Xray-core mobile development.
package libXray

import "encoding/json"

type LibXrayMethod string

const (
	LibXrayMethodGetFreePorts                LibXrayMethod = "getFreePorts"
	LibXrayMethodConvertShareLinksToXrayJson LibXrayMethod = "convertShareLinksToXrayJson"
	LibXrayMethodConvertXrayJsonToShareLinks LibXrayMethod = "convertXrayJsonToShareLinks"
	LibXrayMethodCountGeoData                LibXrayMethod = "countGeoData"
	LibXrayMethodPing                        LibXrayMethod = "ping"
	LibXrayMethodTestXray                    LibXrayMethod = "testXray"
	LibXrayMethodRunXray                     LibXrayMethod = "runXray"
	LibXrayMethodRunXrayFromJson             LibXrayMethod = "runXrayFromJson"
	LibXrayMethodStopXray                    LibXrayMethod = "stopXray"
	LibXrayMethodXrayVersion                 LibXrayMethod = "xrayVersion"
	LibXrayMethodGetXrayState                LibXrayMethod = "getXrayState"
)

type LibXrayInvokeRequest struct {
	APIVersion int             `json:"apiVersion,omitempty"`
	Method     LibXrayMethod   `json:"method,omitempty"`
	Env        *LibXrayEnvJson `json:"env,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
}

type LibXrayEnvJson struct {
	ConfigLocation       string `json:"xray.location.config,omitempty"`
	ConfdirLocation      string `json:"xray.location.confdir,omitempty"`
	AssetLocation        string `json:"xray.location.asset,omitempty"`
	CertLocation         string `json:"xray.location.cert,omitempty"`
	UseReadV             string `json:"xray.buf.readv,omitempty"`
	UseFreedomSplice     string `json:"xray.buf.splice,omitempty"`
	UseVmessPadding      string `json:"xray.vmess.padding,omitempty"`
	UseCone              string `json:"xray.cone.disabled,omitempty"`
	UseStrictJSON        string `json:"xray.json.strict,omitempty"`
	BufferSize           string `json:"xray.ray.buffer.size,omitempty"`
	BrowserDialerAddress string `json:"xray.browser.dialer,omitempty"`
	XUDPLog              string `json:"xray.xudp.show,omitempty"`
	XUDPBaseKey          string `json:"xray.xudp.basekey,omitempty"`
	TunFd                string `json:"xray.tun.fd,omitempty"`
}

type GetFreePortsRequest struct {
	Count int `json:"count,omitempty"`
}

type GetFreePortsResponse struct {
	Ports []int `json:"ports,omitempty"`
}

type ConvertShareLinksToXrayJsonRequest struct {
	Text string `json:"text,omitempty"`
}

type ConvertXrayJsonToShareLinksRequest struct {
	XrayJson string `json:"xrayJson,omitempty"`
}

type ConvertXrayJsonToShareLinksResponse struct {
	Links string `json:"links,omitempty"`
}

type CountGeoDataRequest struct {
	Name    string `json:"name,omitempty"`
	GeoType string `json:"geoType,omitempty"`
}

type PingRequest struct {
	ConfigPath string `json:"configPath,omitempty"`
	Timeout    int    `json:"timeout,omitempty"`
	URL        string `json:"url,omitempty"`
	Proxy      string `json:"proxy,omitempty"`
}

type PingResponse struct {
	Delay int64 `json:"delay,omitempty"`
}

type RunXrayRequest struct {
	ConfigPath string `json:"configPath,omitempty"`
}

type RunXrayFromJSONRequest struct {
	ConfigJSON string `json:"configJSON,omitempty"`
}

type XrayVersionResponse struct {
	Version string `json:"version,omitempty"`
}

type GetXrayStateResponse struct {
	Running bool `json:"running"`
}
