package share

import (
	"encoding/json"
	"errors"
	"net/url"
	"strings"

	"github.com/xtls/xray-core/infra/conf"
	"gopkg.in/yaml.v3"
)

// ConvertShareLinksResult counts source candidates, not lines or changes to a subscription.
// Config contains exactly the projected, buildable outbounds counted as usable.
type ConvertShareLinksResult struct {
	Config      json.RawMessage `json:"config"`
	UsableCount int             `json:"usableCount"`
	FailedCount int             `json:"failedCount"`
}

// ConvertShareLinksToXrayJson parses share links or an Age-encrypted subscription.
// A recognized candidate container with no usable nodes returns both its counts
// and an error. Whole-document/decryption failures return no invented counts.
func ConvertShareLinksToXrayJson(links, secretKey string) (*ConvertShareLinksResult, error) {
	text, encrypted, err := decryptShareText(links, secretKey)
	if err != nil {
		return nil, err
	}
	config, candidates, err := parseShareCandidates(text, true)
	if err != nil {
		if encrypted {
			return nil, ErrAgePlaintextUnsupported
		}
		return nil, err
	}
	result := &ConvertShareLinksResult{Config: json.RawMessage(`{"outbounds":[]}`), FailedCount: candidates}
	config, err = filterBuildableOutbounds(config)
	if err == nil {
		var raw json.RawMessage
		var usable int
		raw, usable, err = marshalShareConfigJSON(config)
		if err == nil {
			result.Config, result.UsableCount, result.FailedCount = raw, usable, candidates-usable
			return result, nil
		}
	}
	if encrypted {
		return result, ErrAgePlaintextUnsupported
	}
	// Builder errors can contain credentials or whole source values. Counts do
	// not require those diagnostics; never echo rejected candidates.
	return result, errors.New("no valid outbound found")
}

func parseShareCandidates(links string, allowBase64 bool) (*conf.Config, int, error) {
	text := strings.TrimSpace(FixWindowsReturn(links))
	config := &conf.Config{}
	if strings.HasPrefix(text, "{") {
		var document struct {
			Outbounds []json.RawMessage `json:"outbounds"`
		}
		if err := json.Unmarshal([]byte(text), &document); err != nil || document.Outbounds == nil {
			return nil, 0, errors.New("invalid share JSON outbounds")
		}
		for _, raw := range document.Outbounds {
			var outbound conf.OutboundDetourConfig
			if err := json.Unmarshal(raw, &outbound); err == nil {
				config.OutboundConfigs = append(config.OutboundConfigs, outbound)
			}
		}
		return config, len(document.Outbounds), nil
	}
	if hasShareSchemeLine(text) {
		candidates := 0
		forEachLine(text, func(raw string) bool {
			line := strings.TrimSpace(raw)
			// Subscription comments/headers are not node candidates. A URI-like
			// row is one candidate, including an unsupported or malformed URI.
			scheme, _, found := strings.Cut(line, "://")
			if !found || strings.ContainsAny(scheme, " \t#") {
				return true
			}
			candidates++
			parsed, err := url.Parse(line)
			if err != nil {
				return true
			}
			outbound, err := (xrayShareLink{link: parsed, rawText: line}).outbound()
			if err == nil {
				config.OutboundConfigs = append(config.OutboundConfigs, *outbound)
			}
			return true
		})
		return config, candidates, nil
	}
	if allowBase64 {
		if decoded, err := decodeBase64Text(text); err == nil {
			return parseShareCandidates(decoded, false)
		}
	}
	if hasTopLevelClashProxiesKey(text) {
		var document struct {
			Proxies []yaml.Node `yaml:"proxies"`
		}
		if err := yaml.Unmarshal([]byte(text), &document); err != nil || document.Proxies == nil {
			return nil, 0, errors.New("invalid share YAML proxies")
		}
		for _, node := range document.Proxies {
			var proxy ClashProxy
			if err := node.Decode(&proxy); err != nil {
				continue
			}
			outbound, err := proxy.outbound()
			if err == nil {
				config.OutboundConfigs = append(config.OutboundConfigs, *outbound)
			}
		}
		return config, len(document.Proxies), nil
	}
	return nil, 0, errors.New("unsupported share format")
}
