//go:build linux && !android

package libXray

import (
	"encoding/base64"
	"encoding/json"

	"github.com/xtls/libxray/dns"
	"github.com/xtls/libxray/nodep"
	"github.com/xtls/libxray/tun"
)

type initDnsRequest struct {
	Dns        string `json:"dns,omitempty"`
	DeviceName string `json:"deviceName,omitempty"`
}

func InitDns(base64Text string) string {
	var response nodep.CallResponse[string]
	req, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	var request initDnsRequest
	err = json.Unmarshal(req, &request)
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	dns.InitDns(request.Dns, request.DeviceName)
	return response.EncodeToBase64("", nil)
}

func ResetDns() string {
	var response nodep.CallResponse[string]
	dns.ResetDns()
	return response.EncodeToBase64("", nil)
}

type startTunRequest struct {
	Name string `json:"name,omitempty"`
	Mtu  int    `json:"mtu,omitempty"`
}

func StartTun(base64Text string) string {
	var response nodep.CallResponse[int]
	req, err := base64.StdEncoding.DecodeString(base64Text)
	if err != nil {
		return response.EncodeToBase64(0, err)
	}
	var request startTunRequest
	err = json.Unmarshal(req, &request)
	if err != nil {
		return response.EncodeToBase64(0, err)
	}
	fd, err := tun.StartTun(request.Name, request.Mtu)
	if err != nil {
		return response.EncodeToBase64(0, err)
	}
	return response.EncodeToBase64(fd, nil)
}

func StopTun() string {
	var response nodep.CallResponse[string]
	err := tun.StopTun()
	if err != nil {
		return response.EncodeToBase64("", err)
	}
	return response.EncodeToBase64("", nil)
}
