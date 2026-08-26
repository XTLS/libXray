package share

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/xtls/xray-core/infra/conf"
)

// MarshalShareConfigJSON returns the Xray JSON subset supported by share links.
func MarshalShareConfigJSON(config *conf.Config) (json.RawMessage, error) {
	if config == nil {
		return nil, fmt.Errorf("no valid outbound found")
	}

	outbounds := make([]map[string]any, 0, len(config.OutboundConfigs))
	var firstBuildError error
	for _, outbound := range config.OutboundConfigs {
		source, err := marshalShareJSONObject(outbound)
		if err != nil {
			return nil, err
		}
		projected, supported := projectShareOutbound(source)
		if !supported {
			continue
		}
		if err := validateProjectedShareOutbound(projected); err != nil {
			if firstBuildError == nil {
				firstBuildError = err
			}
			continue
		}
		outbounds = append(outbounds, projected)
	}

	if len(outbounds) == 0 {
		if firstBuildError != nil {
			return nil, fmt.Errorf("no valid outbound found: %w", firstBuildError)
		}
		return nil, fmt.Errorf("no valid outbound found")
	}

	raw, err := json.Marshal(map[string]any{"outbounds": outbounds})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal share config: %w", err)
	}
	return raw, nil
}

func marshalShareJSONObject(value any) (map[string]any, error) {
	raw, err := json.Marshal(value)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal share config: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	var object map[string]any
	if err := decoder.Decode(&object); err != nil {
		return nil, fmt.Errorf("failed to marshal share config: %w", err)
	}
	return object, nil
}

func projectShareOutbound(source map[string]any) (map[string]any, bool) {
	protocol := strings.ToLower(shareString(source["protocol"]))
	settings, _ := shareObject(source["settings"])

	projectedSettings := map[string]any{}
	switch protocol {
	case "shadowsocks":
		copyShareFields(projectedSettings, settings, "address", "port", "method", "password")
	case "vmess":
		copyShareFields(projectedSettings, settings, "address", "port", "id", "security")
	case "vless":
		copyShareFields(projectedSettings, settings, "address", "port", "id", "flow", "encryption")
	case "socks":
		copyShareFields(projectedSettings, settings, "address", "port", "user", "pass")
	case "trojan":
		copyShareFields(projectedSettings, settings, "address", "port", "password")
	case "hysteria":
		copyShareFields(projectedSettings, settings, "version", "address", "port")
	default:
		return nil, false
	}

	projected := map[string]any{
		"protocol": protocol,
		"settings": projectedSettings,
	}
	copyShareFields(projected, source, "sendThrough", "tag")

	if stream, ok := shareObject(source["streamSettings"]); ok {
		projectedStream, supported := projectShareStream(stream)
		if !supported {
			return nil, false
		}
		if len(projectedStream) > 0 {
			projected["streamSettings"] = projectedStream
		}
	}
	return projected, true
}

func projectShareStream(source map[string]any) (map[string]any, bool) {
	networkValue := shareString(source["network"])
	if method := shareString(source["method"]); method != "" {
		networkValue = method
	}
	network, supported := canonicalShareNetwork(networkValue)
	if !supported {
		return nil, false
	}

	securityValue := shareString(source["security"])
	security := strings.ToLower(securityValue)
	if security != "" && security != "none" && security != "tls" && security != "reality" {
		return nil, false
	}

	projected := map[string]any{}
	if networkValue != "" {
		projected["network"] = network
	}
	if securityValue != "" {
		projected["security"] = security
	}

	switch network {
	case "raw":
		if settings, ok := firstShareObject(source, "rawSettings", "tcpSettings"); ok {
			if value := projectShareRAWSettings(settings); len(value) > 0 {
				projected["rawSettings"] = value
			}
		}
	case "kcp":
		// KCP share links carry only the transport discriminator.
	case "ws":
		projectShareTransportSettings(projected, source, "wsSettings", "host", "path")
	case "grpc":
		projectShareTransportSettings(projected, source, "grpcSettings", "authority", "serviceName", "multiMode")
	case "httpupgrade":
		projectShareTransportSettings(projected, source, "httpupgradeSettings", "host", "path")
	case "xhttp":
		if settings, ok := firstShareObject(source, "xhttpSettings", "splithttpSettings"); ok {
			value := map[string]any{}
			copyShareFields(value, settings, "host", "path", "mode", "extra")
			if len(value) > 0 {
				projected["xhttpSettings"] = value
			}
		}
	case "hysteria":
		projectShareTransportSettings(projected, source, "hysteriaSettings", "version", "auth")
	}

	switch security {
	case "tls":
		if settings, ok := shareObject(source["tlsSettings"]); ok {
			value := map[string]any{}
			copyShareFields(value, settings,
				"serverName", "alpn", "fingerprint", "echConfigList",
				"pinnedPeerCertSha256", "verifyPeerCertByName",
			)
			if len(value) > 0 {
				projected["tlsSettings"] = value
			}
		}
	case "reality":
		if settings, ok := shareObject(source["realitySettings"]); ok {
			value := map[string]any{}
			copyShareFields(value, settings,
				"serverName", "fingerprint", "shortId", "mldsa65Verify", "spiderX",
			)
			if password := shareString(settings["password"]); password != "" {
				value["password"] = password
			} else if publicKey := shareString(settings["publicKey"]); publicKey != "" {
				value["password"] = publicKey
			}
			if len(value) > 0 {
				projected["realitySettings"] = value
			}
		}
	}

	if finalMask, ok := shareObject(source["finalmask"]); ok {
		if value := projectShareFinalMask(finalMask); len(value) > 0 {
			projected["finalmask"] = value
		}
	}
	return projected, true
}

func canonicalShareNetwork(network string) (string, bool) {
	switch strings.ToLower(network) {
	case "", "raw", "tcp":
		return "raw", true
	case "kcp", "mkcp":
		return "kcp", true
	case "ws", "websocket":
		return "ws", true
	case "grpc":
		return "grpc", true
	case "httpupgrade":
		return "httpupgrade", true
	case "xhttp", "splithttp":
		return "xhttp", true
	case "hysteria":
		return "hysteria", true
	default:
		return "", false
	}
}

func projectShareRAWSettings(source map[string]any) map[string]any {
	header, ok := shareObject(source["header"])
	if !ok || strings.ToLower(shareString(header["type"])) != "http" {
		return nil
	}

	projectedHeader := map[string]any{"type": "http"}
	if request, ok := shareObject(header["request"]); ok {
		projectedRequest := map[string]any{}
		copyShareFields(projectedRequest, request, "path")
		if headers, ok := shareObject(request["headers"]); ok {
			projectedHeaders := map[string]any{}
			copyShareFields(projectedHeaders, headers, "Host")
			if len(projectedHeaders) > 0 {
				projectedRequest["headers"] = projectedHeaders
			}
		}
		if len(projectedRequest) > 0 {
			projectedHeader["request"] = projectedRequest
		}
	}
	return map[string]any{"header": projectedHeader}
}

func projectShareTransportSettings(projected, source map[string]any, field string, keys ...string) {
	settings, ok := shareObject(source[field])
	if !ok {
		return
	}
	value := map[string]any{}
	copyShareFields(value, settings, keys...)
	if len(value) > 0 {
		projected[field] = value
	}
}

func projectShareFinalMask(source map[string]any) map[string]any {
	projected := map[string]any{}
	for _, field := range []string{"tcp", "udp"} {
		masks, ok := source[field].([]any)
		if !ok {
			continue
		}
		projectedMasks := make([]map[string]any, 0, len(masks))
		for _, item := range masks {
			mask, ok := shareObject(item)
			if !ok {
				continue
			}
			maskType := shareString(mask["type"])
			if maskType == "" {
				continue
			}
			projectedMask := map[string]any{"type": maskType}
			if settings, exists := mask["settings"]; exists && !emptyShareValue(settings) {
				projectedMask["settings"] = settings
			}
			projectedMasks = append(projectedMasks, projectedMask)
		}
		if len(projectedMasks) > 0 {
			projected[field] = projectedMasks
		}
	}

	if params, ok := shareObject(source["quicParams"]); ok {
		projectedParams := map[string]any{}
		copyShareFields(projectedParams, params, "congestion", "brutalUp", "brutalDown")
		if udpHop, ok := shareObject(params["udpHop"]); ok {
			projectedUDPHop := map[string]any{}
			copyShareFields(projectedUDPHop, udpHop, "ports")
			if interval, exists := udpHop["interval"]; exists && !emptyShareRange(interval) {
				projectedUDPHop["interval"] = interval
			}
			if len(projectedUDPHop) > 0 {
				projectedParams["udpHop"] = projectedUDPHop
			}
		}
		if len(projectedParams) > 0 {
			projected["quicParams"] = projectedParams
		}
	}
	return projected
}

func validateProjectedShareOutbound(projected map[string]any) error {
	raw, err := json.Marshal(projected)
	if err != nil {
		return err
	}
	var outbound conf.OutboundDetourConfig
	if err := json.Unmarshal(raw, &outbound); err != nil {
		return err
	}
	// sendThrough stores the node display name during share conversion.
	outbound.SendThrough = nil
	_, err = outbound.Build()
	return err
}

func copyShareFields(target, source map[string]any, fields ...string) {
	for _, field := range fields {
		if value, ok := source[field]; ok && !emptyShareValue(value) {
			target[field] = value
		}
	}
}

func firstShareObject(source map[string]any, fields ...string) (map[string]any, bool) {
	for _, field := range fields {
		if object, ok := shareObject(source[field]); ok {
			return object, true
		}
	}
	return nil, false
}

func shareObject(value any) (map[string]any, bool) {
	object, ok := value.(map[string]any)
	return object, ok
}

func shareString(value any) string {
	text, _ := value.(string)
	return text
}

func emptyShareValue(value any) bool {
	switch value := value.(type) {
	case nil:
		return true
	case string:
		return value == ""
	case bool:
		return !value
	case json.Number:
		number, err := strconv.ParseFloat(value.String(), 64)
		return err == nil && number == 0
	case []any:
		return len(value) == 0
	case map[string]any:
		return len(value) == 0
	default:
		return false
	}
}

func emptyShareRange(value any) bool {
	if text, ok := value.(string); ok {
		number, err := strconv.ParseInt(text, 10, 32)
		return err == nil && number == 0
	}
	return emptyShareValue(value)
}
