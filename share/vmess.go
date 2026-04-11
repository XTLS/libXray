package share

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/xtls/xray-core/infra/conf"
)

// https://github.com/2dust/v2rayN/wiki/%E5%88%86%E4%BA%AB%E9%93%BE%E6%8E%A5%E6%A0%BC%E5%BC%8F%E8%AF%B4%E6%98%8E(ver-2)
type vmessQrCode struct {
	Ps   string      `json:"ps,omitempty"`
	Add  string      `json:"add,omitempty"`
	Port interface{} `json:"port,omitempty"`
	Id   string      `json:"id,omitempty"`
	Scy  string      `json:"scy,omitempty"`
	Net  string      `json:"net,omitempty"`
	Type string      `json:"type,omitempty"`
	Host string      `json:"host,omitempty"`
	Path string      `json:"path,omitempty"`
	Tls  string      `json:"tls,omitempty"`
	Sni  string      `json:"sni,omitempty"`
	Alpn string      `json:"alpn,omitempty"`
	Fp   string      `json:"fp,omitempty"`
}

func parseVMessQrCode(text string) (*conf.OutboundDetourConfig, error) {
	var qrcode vmessQrCode
	if err := json.Unmarshal([]byte(text), &qrcode); err != nil {
		return nil, err
	}
	return qrcode.outbound()
}

func (proxy vmessQrCode) outbound() (*conf.OutboundDetourConfig, error) {
	outbound := &conf.OutboundDetourConfig{}
	outbound.Protocol = "vmess"
	setOutboundName(outbound, proxy.Ps)

	settings := conf.VMessOutboundConfig{}
	settings.Address = parseAddress(proxy.Add)
	portStr := fmt.Sprintf("%v", proxy.Port)
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, err
	}
	settings.Port = uint16(port)
	settings.ID = proxy.Id
	settings.Security = proxy.Scy

	settingsRawMessage, err := convertJsonToRawMessage(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = &settingsRawMessage

	streamSettings, err := buildStreamFromTransportFields(transportFieldsFromVmessQR(proxy))
	if err != nil {
		return nil, err
	}
	if err := proxy.parseSecurity(streamSettings); err != nil {
		return nil, err
	}
	outbound.StreamSetting = streamSettings
	return outbound, nil
}

func (proxy vmessQrCode) parseSecurity(streamSettings *conf.StreamConfig) error {
	tlsSettings := &conf.TLSConfig{}
	tlsSettings.Fingerprint = proxy.Fp
	tlsSettings.ServerName = proxy.Sni
	if proxy.Alpn != "" {
		tlsSettings.ALPN = new(conf.StringList(strings.Split(proxy.Alpn, ",")))
	}

	if proxy.Tls == "" {
		streamSettings.Security = "none"
	} else {
		streamSettings.Security = proxy.Tls
	}

	network, err := streamSettings.Network.Build()
	if err != nil {
		return err
	}
	// some link omits too many params, here is some fixing
	if network == "websocket" && tlsSettings.ServerName == "" &&
		streamSettings.WSSettings != nil && streamSettings.WSSettings.Host != "" {
		tlsSettings.ServerName = streamSettings.WSSettings.Host
	}

	switch streamSettings.Security {
	case "tls":
		streamSettings.TLSSettings = tlsSettings
	}
	return nil
}
