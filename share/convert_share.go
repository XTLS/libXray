package share

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"github.com/xtls/xray-core/infra/conf"
)

// ConvertShareLinksToXrayJson parses share links or an Age-encrypted subscription.
// The returned JSON contains only projected, buildable outbounds.
func ConvertShareLinksToXrayJson(links, secretKey string) (json.RawMessage, error) {
	text, encrypted, err := decryptShareText(links, secretKey)
	if err != nil {
		return nil, err
	}
	config, err := parseShareCandidates(text, true)
	if err != nil {
		if encrypted {
			return nil, ErrAgePlaintextUnsupported
		}
		return nil, err
	}
	config, err = filterBuildableOutbounds(config)
	if err == nil {
		var raw json.RawMessage
		raw, err = marshalShareConfigJSON(config)
		if err == nil {
			return raw, nil
		}
	}
	if encrypted {
		return nil, ErrAgePlaintextUnsupported
	}
	// Builder errors can contain credentials or whole source values.
	return nil, errors.New("no valid outbound found")
}

func parseShareCandidates(links string, allowBase64 bool) (*conf.Config, error) {
	text := strings.TrimSpace(FixWindowsReturn(links))
	config := &conf.Config{}
	if strings.HasPrefix(text, "{") {
		var document struct {
			Outbounds []json.RawMessage `json:"outbounds"`
		}
		if err := json.Unmarshal([]byte(text), &document); err != nil || document.Outbounds == nil {
			return nil, errors.New("invalid share JSON outbounds")
		}
		for _, raw := range document.Outbounds {
			var outbound conf.OutboundDetourConfig
			if err := json.Unmarshal(raw, &outbound); err == nil {
				config.OutboundConfigs = append(config.OutboundConfigs, outbound)
			}
		}
		return config, nil
	}
	if hasShareSchemeLine(text) {
		for raw := range strings.SplitSeq(text, "\n") {
			line := strings.TrimSpace(raw)
			// Ignore subscription comments and text headers.
			scheme, _, found := strings.Cut(line, "://")
			if !found || strings.ContainsAny(scheme, " \t#") {
				continue
			}
			parsed, err := url.Parse(line)
			if err != nil {
				continue
			}
			outbound, err := (xrayShareLink{link: parsed, rawText: line}).outbound()
			if err == nil {
				config.OutboundConfigs = append(config.OutboundConfigs, *outbound)
			}
		}
		return config, nil
	}
	if allowBase64 {
		if decoded, err := decodeBase64Text(text); err == nil {
			return parseShareCandidates(decoded, false)
		}
	}
	return nil, errors.New("unsupported share format")
}
