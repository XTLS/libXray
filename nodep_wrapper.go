package libXray

import (
	"encoding/base64"

	"github.com/xtls/libxray/nodep"
)

type getFreePortsResponse struct {
	Ports []int `json:"ports,omitempty"`
}

// Wrapper of nodep.GetFreePorts
// count means how many ports you need.
func GetFreePorts(count int) string {
	var response nodep.CallResponse[*getFreePortsResponse]
	ports, err := nodep.GetFreePorts(count)
	if err != nil {
		return response.EncodeToBase64(nil, err)
	}
	var res getFreePortsResponse
	res.Ports = ports
	return response.EncodeToBase64(&res, nil)
}

// Convert share text to XrayJson
// support XrayJson, v2rayN plain text, v2rayN base64 text, Clash yaml, Clash.Meta yaml
func ConvertShareLinksToXrayJson(base64Text string) string {
	var response nodep.CallResponse[*nodep.XrayJson]
	links, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64(nil, err)
	}
	xrayJson, err := nodep.ConvertShareLinksToXrayJson(string(links))
	return response.EncodeToBase64(xrayJson, err)
}

// Convert XrayJson to share links.
// VMess will generate VMessAEAD link.
func ConvertXrayJsonToShareLinks(base64Text string) string {
	var response nodep.CallResponse[string]
	xray, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	links, err := nodep.ConvertXrayJsonToShareLinks(xray)
	return response.EncodeToBase64(links, err)
}
