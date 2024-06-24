package libXray

import (
	"encoding/json"

	"github.com/xtls/libxray/nodep"
)

type getFreePortsResponse struct {
	Ports []int `json:"ports,omitempty"`
}

// Wrapper of nodep.GetFreePorts
// count means how many ports you need.
// return ports divided by ":", like "1080:1081"
func GetFreePorts(count int) string {
	ports, err := nodep.GetFreePorts(count)
	if err != nil {
		return makeCallResponse("", err)
	}
	var res getFreePortsResponse
	res.Ports = ports
	b, err := json.Marshal(res)
	if err != nil {
		return makeCallResponse("", err)
	}
	return makeCallResponse(string(b), nil)
}

// Convert share text to XrayJson
// support XrayJson, v2rayN plain text, v2rayN base64 text, Clash yaml, Clash.Meta yaml
func ConvertShareLinksToXrayJson(links string) string {
	res, err := nodep.ConvertShareLinksToXrayJson(links)
	return makeCallResponse(res, err)
}

// Convert XrayJson to share links.
// VMess will generate VMessAEAD link.
func ConvertXrayJsonToShareLinks(xray string) string {
	res, err := nodep.ConvertXrayJsonToShareLinks(xray)
	return makeCallResponse(res, err)
}

func makeCallResponse(result string, err error) string {
	var response nodep.CallResponse
	response.Result = result
	if err != nil {
		response.Success = false
		response.Err = err.Error()
	} else {
		response.Success = true
	}
	b, err := json.Marshal(response)
	if err != nil {
		return ""
	}
	return string(b)
}
