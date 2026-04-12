package share

import (
	"encoding/json"
	"net/url"
	"strings"

	"github.com/xtls/xray-core/infra/conf"
)

// shareTransportFields is the normalized transport slice of v2rayN share links and VMess QR JSON.
type shareTransportFields struct {
	Network         string
	HeaderType      string
	Path            string
	Host            string
	Seed            string // KCP seed (URL query or VMess QR path-as-seed)
	GrpcAuthority   string
	GrpcServiceName string
	GrpcMultiMode   bool
	XHTTPMode       string
	ExtraJSON       string // SplitHTTP extra (URL only)
	FMJSON          string // serialized FinalMask (URL only)
}

func transportFieldsFromURLQuery(q url.Values) shareTransportFields {
	network := q.Get("type")
	if network == "" {
		network = "raw"
	}
	return shareTransportFields{
		Network:         network,
		HeaderType:      q.Get("headerType"),
		Path:            q.Get("path"),
		Host:            q.Get("host"),
		Seed:            q.Get("seed"),
		GrpcAuthority:   q.Get("authority"),
		GrpcServiceName: q.Get("serviceName"),
		GrpcMultiMode:   q.Get("mode") == "multi",
		XHTTPMode:       q.Get("mode"),
		ExtraJSON:       q.Get("extra"),
		FMJSON:          q.Get("fm"),
	}
}

func transportFieldsFromVmessQR(p vmessQrCode) shareTransportFields {
	network := p.Net
	if network == "" {
		network = "raw"
	}
	t := shareTransportFields{
		Network:    network,
		HeaderType: p.Type,
		Path:       p.Path,
		Host:       p.Host,
	}
	switch network {
	case "grpc", "gun":
		t.GrpcServiceName = p.Path
		t.GrpcMultiMode = p.Type == "multi"
	case "kcp", "mkcp":
		t.Seed = p.Path
	}
	if network == "xhttp" || network == "splithttp" {
		t.XHTTPMode = p.Type
	}
	return t
}

// buildStreamFromTransportFields builds StreamConfig from normalized share fields (no TLS).
func buildStreamFromTransportFields(t shareTransportFields) (*conf.StreamConfig, error) {
	streamSettings := &conf.StreamConfig{}
	network := t.Network
	if network == "" {
		network = "raw"
	}
	streamSettings.Network = new(conf.TransportProtocol(network))

	switch network {
	case "raw", "tcp":
		if t.HeaderType == "http" {
			var request XrayRawSettingsHeaderRequest
			if t.Path != "" {
				request.Path = strings.Split(t.Path, ",")
			}
			if t.Host != "" {
				request.Headers = &XrayRawSettingsHeaderRequestHeaders{Host: strings.Split(t.Host, ",")}
			}
			header := XrayRawSettingsHeader{Type: t.HeaderType, Request: &request}
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
		if t.HeaderType != "" {
			header := XrayFakeHeader{Type: t.HeaderType}
			headerRawMessage, err := convertJsonToRawMessage(header)
			if err != nil {
				return nil, err
			}
			kcpSettings.HeaderConfig = headerRawMessage
		}
		kcpSettings.Seed = new(t.Seed)
		streamSettings.KCPSettings = kcpSettings
	case "ws", "websocket":
		streamSettings.WSSettings = &conf.WebSocketConfig{Path: t.Path, Host: t.Host}
	case "grpc", "gun":
		streamSettings.GRPCSettings = &conf.GRPCConfig{
			Authority:   t.GrpcAuthority,
			ServiceName: t.GrpcServiceName,
			MultiMode:   t.GrpcMultiMode,
		}
	case "httpupgrade":
		streamSettings.HTTPUPGRADESettings = &conf.HttpUpgradeConfig{Host: t.Host, Path: t.Path}
	case "xhttp", "splithttp":
		xhttpSettings := &conf.SplitHTTPConfig{Host: t.Host, Path: t.Path, Mode: t.XHTTPMode}
		if t.ExtraJSON != "" {
			var extraConfig *conf.SplitHTTPConfig
			if err := json.Unmarshal([]byte(t.ExtraJSON), &extraConfig); err != nil {
				return nil, err
			}
			extraRawMessage, err := convertJsonToRawMessage(extraConfig)
			if err != nil {
				return nil, err
			}
			xhttpSettings.Extra = extraRawMessage
		}
		streamSettings.XHTTPSettings = xhttpSettings
	}

	if t.FMJSON != "" {
		var finalMask *conf.FinalMask
		if err := json.Unmarshal([]byte(t.FMJSON), &finalMask); err != nil {
			return nil, err
		}
		streamSettings.FinalMask = finalMask
	}

	return streamSettings, nil
}
