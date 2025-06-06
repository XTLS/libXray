//go:build !android

package libXray

import (
	"encoding/base64"
	"encoding/json"

	"github.com/xtls/libxray/dns"
	"github.com/xtls/libxray/nodep"
)

type InitDnsRequest struct {
	Dns        string `json:"dns,omitempty"`
	DeviceName string `json:"deviceName,omitempty"`
}

// Init Dns Request
func NewInitDnsRequest(dns, deviceName string) (string, error) {
	request := InitDnsRequest{
		Dns:        dns,
		DeviceName: deviceName,
	}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return "", err
	}
	// Encode the JSON bytes to a base64 string
	return base64.StdEncoding.EncodeToString(requestBytes), nil
}

// Init Dns.
func InitDns(base64Text string) string {
	var response nodep.CallResponse[string]
	req, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	var request InitDnsRequest
	err = json.Unmarshal(req, &request)
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	dns.InitDns(request.Dns, request.DeviceName)
	return response.EncodeToBase64("", err)
}

func ResetDns() string {
	var response nodep.CallResponse[string]
	dns.ResetDns()
	return response.EncodeToBase64("", nil)
}
