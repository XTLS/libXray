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
	qrcodeBytes := []byte(text)
	qrcode := vmessQrCode{}

	err := json.Unmarshal(qrcodeBytes, &qrcode)
	if err != nil {
		return nil, err
	}

	return qrcode.outbound()
}

func (proxy vmessQrCode) outbound() (*conf.OutboundDetourConfig, error) {
	outbound := &conf.OutboundDetourConfig{}
	outbound.Protocol = "vmess"
	setOutboundName(outbound, proxy.Ps)

	user := &conf.VMessAccount{}
	user.ID = proxy.Id
	user.Security = proxy.Scy

	vnext := &conf.VMessOutboundTarget{}
	vnext.Address = parseAddress(proxy.Add)

	portStr := fmt.Sprintf("%v", proxy.Port)
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, err
	}
	vnext.Port = uint16(port)

	userRawMessage, err := convertJsonToRawMessage(user)
	if err != nil {
		return nil, err
	}
	vnext.Users = []json.RawMessage{userRawMessage}

	settings := conf.VMessOutboundConfig{}
	settings.Receivers = []*conf.VMessOutboundTarget{vnext}

	settingsRawMessage, err := convertJsonToRawMessage(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = &settingsRawMessage

	streamSettings, err := proxy.streamSettings()
	if err != nil {
		return nil, err
	}
	outbound.StreamSetting = streamSettings

	return outbound, nil
}

func (proxy vmessQrCode) streamSettings() (*conf.StreamConfig, error) {
	streamSettings := &conf.StreamConfig{}
	network := proxy.Net
	if len(network) == 0 {
		network = "raw"
	}
	transportProtocol := conf.TransportProtocol(network)
	streamSettings.Network = &transportProtocol

	switch network {
	case "raw", "tcp":
		headerType := proxy.Type
		if headerType == "http" {
			var request XrayRawSettingsHeaderRequest
			path := proxy.Path
			if len(path) > 0 {
				request.Path = strings.Split(path, ",")
			}
			host := proxy.Host
			if len(host) > 0 {
				var headers XrayRawSettingsHeaderRequestHeaders
				headers.Host = strings.Split(host, ",")
				request.Headers = &headers
			}
			var header XrayRawSettingsHeader
			header.Type = headerType
			header.Request = &request

			rawSettings := &conf.TCPConfig{}

			headerRawMessage, err := convertJsonToRawMessage(header)
			if err != nil {
				return nil, err
			}
			rawSettings.HeaderConfig = headerRawMessage

			streamSettings.RAWSettings = rawSettings
		}
	case "kcp", "mkcp":
		kcpSettings := &conf.KCPConfig{}
		headerType := proxy.Type
		if len(headerType) > 0 {
			var header XrayFakeHeader
			header.Type = headerType

			headerRawMessage, err := convertJsonToRawMessage(header)
			if err != nil {
				return nil, err
			}
			kcpSettings.HeaderConfig = headerRawMessage
		}
		kcpSettings.Seed = &proxy.Path

		streamSettings.KCPSettings = kcpSettings
	case "ws", "websocket":
		wsSettings := &conf.WebSocketConfig{}
		wsSettings.Path = proxy.Path
		wsSettings.Host = proxy.Host

		streamSettings.WSSettings = wsSettings
	case "grpc", "gun":
		grcpSettings := &conf.GRPCConfig{}
		grcpSettings.ServiceName = proxy.Path
		mode := proxy.Type
		grcpSettings.MultiMode = mode == "multi"
		streamSettings.GRPCSettings = grcpSettings
	}

	err := proxy.parseSecurity(streamSettings)
	if err != nil {
		return nil, err
	}

	return streamSettings, nil
}

func (proxy vmessQrCode) parseSecurity(streamSettings *conf.StreamConfig) error {
	tlsSettings := &conf.TLSConfig{}

	tlsSettings.Fingerprint = proxy.Fp
	tlsSettings.ServerName = proxy.Sni

	alpn := proxy.Alpn
	if len(alpn) > 0 {
		alpn := conf.StringList(strings.Split(alpn, ","))
		tlsSettings.ALPN = &alpn
	}

	security := proxy.Tls
	if len(security) == 0 {
		streamSettings.Security = "none"
	} else {
		streamSettings.Security = security
	}

	network, err := streamSettings.Network.Build()
	if err != nil {
		return err
	}
	// some link omits too many params, here is some fixing
	if network == "websocket" && len(tlsSettings.ServerName) == 0 {
		if streamSettings.WSSettings != nil && len(streamSettings.WSSettings.Host) > 0 {
			tlsSettings.ServerName = streamSettings.WSSettings.Host
		}
	}

	switch streamSettings.Security {
	case "tls":
		streamSettings.TLSSettings = tlsSettings
	}
	return nil
}
