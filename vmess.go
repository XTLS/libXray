package libxray

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

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

func parseVMessQrCode(text string) (*xrayOutbound, error) {
	qrcodeBytes := []byte(text)
	qrcode := vmessQrCode{}

	err := json.Unmarshal(qrcodeBytes, &qrcode)
	if err != nil {
		return nil, err
	}

	return qrcode.outbound()
}

func (proxy vmessQrCode) outbound() (*xrayOutbound, error) {
	var outbound xrayOutbound
	outbound.Protocol = "vmess"
	outbound.Name = proxy.Ps

	var user xrayVMessVnextUser
	user.Id = proxy.Id
	user.Security = proxy.Scy

	var vnext xrayVMessVnext
	vnext.Address = proxy.Add
	portStr := fmt.Sprintf("%v", proxy.Port)
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, err
	}
	vnext.Port = port

	vnext.Users = []xrayVMessVnextUser{user}

	var settings xrayVMess
	settings.Vnext = []xrayVMessVnext{vnext}

	setttingsBytes, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = (*json.RawMessage)(&setttingsBytes)

	outbound.StreamSettings = proxy.streamSettings()

	return &outbound, nil
}

func (proxy vmessQrCode) streamSettings() *xrayStreamSettings {
	var streamSettings xrayStreamSettings
	network := proxy.Net
	if len(network) == 0 {
		streamSettings.Network = "tcp"
	} else {
		streamSettings.Network = network
	}

	switch streamSettings.Network {
	case "tcp":
		headerType := proxy.Type
		if headerType == "http" {
			var request xrayTcpSettingsHeaderRequest
			path := proxy.Path
			if len(path) > 0 {
				request.Path = strings.Split(path, ",")
			}
			host := proxy.Host
			if len(host) > 0 {
				var headers xrayTcpSettingsHeaderRequestHeaders
				headers.Host = strings.Split(host, ",")
				request.Headers = &headers
			}
			var header xrayTcpSettingsHeader
			header.Type = headerType
			header.Request = &request

			var tcpSettings xrayTcpSettings
			tcpSettings.Header = &header

			streamSettings.TcpSettings = &tcpSettings
		}
	case "kcp":
		var kcpSettings xrayKcpSettings
		headerType := proxy.Type
		if len(headerType) > 0 {
			var header xrayFakeHeader
			header.Type = headerType
			kcpSettings.Header = &header
		}
		kcpSettings.Seed = proxy.Path

		streamSettings.KcpSettings = &kcpSettings
	case "ws":
		var wsSettings xrayWsSettings
		wsSettings.Path = proxy.Path
		host := proxy.Host
		if len(host) > 0 {
			var headers xrayWsSettingsHeaders
			headers.Host = host
			wsSettings.Headers = &headers
		}

		streamSettings.WsSettings = &wsSettings
	case "grpc":
		var grcpSettings xrayGrpcSettings
		grcpSettings.ServiceName = proxy.Path
		mode := proxy.Type
		grcpSettings.MultiMode = mode == "multi"

		streamSettings.GrpcSettings = &grcpSettings
	case "quic":
		var quicSettings xrayQuicSettings
		headerType := proxy.Type
		if len(headerType) > 0 {
			var header xrayFakeHeader
			header.Type = headerType
			quicSettings.Header = &header
		}
		quicSettings.Security = proxy.Host
		quicSettings.Key = proxy.Path

		streamSettings.QuicSettings = &quicSettings
	case "http":
		var httpSettings xrayHttpSettings
		host := proxy.Host
		httpSettings.Host = strings.Split(host, ",")
		httpSettings.Path = proxy.Path

		streamSettings.HttpSettings = &httpSettings
	}

	proxy.parseSecurity(&streamSettings)

	return &streamSettings
}

func (proxy vmessQrCode) parseSecurity(streamSettings *xrayStreamSettings) {
	var tlsSettings xrayTlsSettings

	tlsSettings.Fingerprint = proxy.Fp
	tlsSettings.ServerName = proxy.Sni

	alpn := proxy.Alpn
	if len(alpn) > 0 {
		tlsSettings.Alpn = strings.Split(alpn, ",")
	}

	security := proxy.Tls
	if len(security) == 0 {
		streamSettings.Security = "none"
	} else {
		streamSettings.Security = security
	}

	switch streamSettings.Security {
	case "tls":
		streamSettings.TlsSettings = &tlsSettings
	}
}
