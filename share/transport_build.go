package share

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/xtls/xray-core/infra/conf"
)

// shareTransportFields holds transport parameters from share-link queries.
type shareTransportFields struct {
	Network         string
	HeaderType      string
	Path            string
	Host            string
	KCPMTU          string
	KCPTTI          string
	GrpcAuthority   string
	GrpcServiceName string
	GrpcMode        string
	XHTTPMode       string
	ExtraJSON       string // SplitHTTP extra
	FMJSON          string // serialized FinalMask
}

func transportFieldsFromURLQuery(q url.Values) shareTransportFields {
	network := q.Get("type")
	if network == "" {
		network = "raw"
	}
	fields := shareTransportFields{
		Network:         network,
		Path:            q.Get("path"),
		Host:            q.Get("host"),
		KCPMTU:          q.Get("mtu"),
		KCPTTI:          q.Get("tti"),
		GrpcAuthority:   q.Get("authority"),
		GrpcServiceName: q.Get("serviceName"),
		GrpcMode:        q.Get("mode"),
		XHTTPMode:       q.Get("mode"),
		ExtraJSON:       q.Get("extra"),
		FMJSON:          q.Get("fm"),
	}
	if network == "raw" || network == "tcp" {
		fields.HeaderType = q.Get("headerType")
	}
	return fields
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
		if t.KCPMTU != "" || t.KCPTTI != "" {
			mtu, err := parseKCPParameter(t.KCPMTU)
			if err != nil {
				return nil, err
			}
			tti, err := parseKCPParameter(t.KCPTTI)
			if err != nil {
				return nil, err
			}
			streamSettings.KCPSettings = &conf.KCPConfig{Mtu: mtu, Tti: tti}
		}
	case "ws", "websocket":
		streamSettings.WSSettings = &conf.WebSocketConfig{Path: t.Path, Host: t.Host}
	case "grpc", "gun":
		if t.GrpcMode != "" && t.GrpcMode != "gun" && t.GrpcMode != "multi" {
			return nil, fmt.Errorf("unsupported gRPC share mode %q", t.GrpcMode)
		}
		streamSettings.GRPCSettings = &conf.GRPCConfig{
			Authority:   t.GrpcAuthority,
			ServiceName: t.GrpcServiceName,
			MultiMode:   t.GrpcMode == "multi",
		}
	case "httpupgrade":
		streamSettings.HTTPUPGRADESettings = &conf.HttpUpgradeConfig{Host: t.Host, Path: t.Path}
	case "xhttp", "splithttp":
		xhttpSettings := &conf.SplitHTTPConfig{Host: t.Host, Path: t.Path, Mode: t.XHTTPMode}
		if t.ExtraJSON != "" {
			if err := json.Unmarshal([]byte(t.ExtraJSON), &xhttpSettings.Extra); err != nil {
				return nil, err
			}
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

func parseKCPParameter(value string) (*uint32, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 32)
	if err != nil {
		return nil, err
	}
	return new(uint32(parsed)), nil
}
