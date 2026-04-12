package share

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/xtls/xray-core/infra/conf"
)

// FixWindowsReturn normalizes CRLF to LF (v2rayN exports on Windows).
func FixWindowsReturn(text string) string {
	return strings.ReplaceAll(text, "\r\n", "\n")
}

// decodeBase64Text decodes standard or URL-safe base64 (with optional missing padding).
func decodeBase64Text(text string) (string, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return "", base64.CorruptInputError(0)
	}
	if b, err := base64.StdEncoding.DecodeString(text); err == nil {
		return string(b), nil
	}
	// URL-safe alphabet, with padding
	if b, err := base64.URLEncoding.DecodeString(text); err == nil {
		return string(b), nil
	}
	// URL-safe raw + manual padding (legacy v2rayN)
	s := strings.ReplaceAll(strings.ReplaceAll(text, "-", "+"), "_", "/")
	if pad := len(s) % 4; pad != 0 {
		s += strings.Repeat("=", 4-pad)
	}
	b, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// https://github.com/XTLS/Xray-core/discussions/716

// ConvertShareLinksToXrayJson parses:
//   - a single Xray JSON object (starts with '{')
//   - plain v2rayN-style lines (vless/vmess/ss/socks/trojan/hy2…)
//   - one base64 blob that decodes to such lines
//   - Clash / Clash.Meta YAML (proxies:)
func ConvertShareLinksToXrayJson(links string) (*conf.Config, error) {
	text := strings.TrimSpace(links)
	if strings.HasPrefix(text, "{") {
		return parseXrayJSONConfig(text)
	}
	text = FixWindowsReturn(text)
	if leadingLineHasShareScheme(text) {
		return parsePlainShareLines(text)
	}
	return tryParseEncodedOrClash(text)
}

func parseXrayJSONConfig(text string) (*conf.Config, error) {
	var xray *conf.Config
	if err := json.Unmarshal([]byte(text), &xray); err != nil {
		return nil, err
	}
	if len(xray.OutboundConfigs) == 0 {
		return nil, fmt.Errorf("no valid outbounds")
	}
	return xray, nil
}

var shareSchemes = []string{
	"vless://", "vmess://", "socks://", "ss://", "trojan://",
	"hysteria2://", "hy2://",
}

func leadingLineHasShareScheme(text string) bool {
	first := text
	if i := strings.IndexByte(text, '\n'); i >= 0 {
		first = text[:i]
	}
	first = strings.TrimSpace(first)
	for _, p := range shareSchemes {
		if strings.HasPrefix(first, p) {
			return true
		}
	}
	return false
}

func parsePlainShareLines(text string) (*conf.Config, error) {
	lines := strings.Split(text, "\n")
	outbounds := make([]conf.OutboundDetourConfig, 0, len(lines))
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		u, err := url.Parse(line)
		if err != nil {
			continue
		}
		sl := xrayShareLink{link: u, rawText: line}
		ob, err := sl.outbound()
		if err != nil {
			continue
		}
		outbounds = append(outbounds, *ob)
	}
	if len(outbounds) == 0 {
		return nil, fmt.Errorf("no valid outbound found")
	}
	return &conf.Config{OutboundConfigs: outbounds}, nil
}

func tryParseEncodedOrClash(text string) (*conf.Config, error) {
	decoded, err := decodeBase64Text(text)
	if err == nil {
		return parsePlainShareLines(FixWindowsReturn(decoded))
	}
	return tryToParseClashYaml(text)
}

type xrayShareLink struct {
	link    *url.URL
	rawText string
}

func (proxy xrayShareLink) outbound() (*conf.OutboundDetourConfig, error) {
	switch proxy.link.Scheme {
	case "ss":
		return proxy.shadowsocksOutbound()
	case "vmess":
		return proxy.vmessOutbound()
	case "vless":
		return proxy.vlessOutbound()
	case "socks":
		return proxy.socksOutbound()
	case "trojan":
		return proxy.trojanOutbound()
	case "hysteria2", "hy2":
		return proxy.hysteriaOutbound()
	default:
		return nil, fmt.Errorf("unsupported link: %s", proxy.rawText)
	}
}

func (proxy xrayShareLink) shadowsocksOutbound() (*conf.OutboundDetourConfig, error) {
	outbound := &conf.OutboundDetourConfig{}
	outbound.Protocol = "shadowsocks"
	setOutboundName(outbound, proxy.link.Fragment)

	settings := &conf.ShadowsocksClientConfig{}
	settings.Address = parseAddress(proxy.link.Hostname())
	port, err := strconv.Atoi(proxy.link.Port())
	if err != nil {
		return nil, err
	}
	settings.Port = uint16(port)

	user := proxy.link.User.String()
	passwordText, err := decodeBase64Text(user)
	if err != nil {
		return nil, err
	}
	pwConfig := strings.SplitN(passwordText, ":", 2)
	if len(pwConfig) != 2 {
		return nil, fmt.Errorf("unsupported shadowsocks link password: %s", passwordText)
	}
	settings.Cipher = pwConfig[0]
	settings.Password = pwConfig[1]

	settingsRawMessage, err := convertJsonToRawMessage(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = &settingsRawMessage

	streamSettings, err := proxy.streamSettings(proxy.link)
	if err != nil {
		return nil, err
	}
	outbound.StreamSetting = streamSettings
	return outbound, nil
}

func (proxy xrayShareLink) vmessOutbound() (*conf.OutboundDetourConfig, error) {
	text := strings.ReplaceAll(proxy.rawText, "vmess://", "")
	if base64Text, err := decodeBase64Text(text); err == nil {
		return parseVMessQrCode(base64Text)
	}

	outbound := &conf.OutboundDetourConfig{}
	outbound.Protocol = "vmess"
	setOutboundName(outbound, proxy.link.Fragment)
	query := proxy.link.Query()

	settings := conf.VMessOutboundConfig{}
	settings.Address = parseAddress(proxy.link.Hostname())
	port, err := strconv.Atoi(proxy.link.Port())
	if err != nil {
		return nil, err
	}
	settings.Port = uint16(port)

	id, err := url.QueryUnescape(proxy.link.User.String())
	if err != nil {
		return nil, err
	}
	settings.ID = id
	if security := query.Get("encryption"); security != "" {
		settings.Security = security
	}

	settingsRawMessage, err := convertJsonToRawMessage(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = &settingsRawMessage

	streamSettings, err := proxy.streamSettings(proxy.link)
	if err != nil {
		return nil, err
	}
	outbound.StreamSetting = streamSettings
	return outbound, nil
}

func (proxy xrayShareLink) vlessOutbound() (*conf.OutboundDetourConfig, error) {
	outbound := &conf.OutboundDetourConfig{}
	outbound.Protocol = "vless"
	setOutboundName(outbound, proxy.link.Fragment)
	query := proxy.link.Query()

	settings := &conf.VLessOutboundConfig{}
	settings.Address = parseAddress(proxy.link.Hostname())
	port, err := strconv.Atoi(proxy.link.Port())
	if err != nil {
		return nil, err
	}
	settings.Port = uint16(port)

	id, err := url.QueryUnescape(proxy.link.User.String())
	if err != nil {
		return nil, err
	}
	settings.Id = id
	if flow := query.Get("flow"); flow != "" {
		settings.Flow = flow
	}
	if enc := query.Get("encryption"); enc != "" {
		settings.Encryption = enc
	} else {
		settings.Encryption = "none"
	}

	settingsRawMessage, err := convertJsonToRawMessage(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = &settingsRawMessage

	streamSettings, err := proxy.streamSettings(proxy.link)
	if err != nil {
		return nil, err
	}
	outbound.StreamSetting = streamSettings
	return outbound, nil
}

func (proxy xrayShareLink) socksOutbound() (*conf.OutboundDetourConfig, error) {
	outbound := &conf.OutboundDetourConfig{}
	outbound.Protocol = "socks"
	setOutboundName(outbound, proxy.link.Fragment)

	settings := &conf.SocksClientConfig{}
	settings.Address = parseAddress(proxy.link.Hostname())
	port, err := strconv.Atoi(proxy.link.Port())
	if err != nil {
		return nil, err
	}
	settings.Port = uint16(port)

	if userPassword := proxy.link.User.String(); userPassword != "" {
		passwordText, err := decodeBase64Text(userPassword)
		if err != nil {
			return nil, err
		}
		pwConfig := strings.SplitN(passwordText, ":", 2)
		if len(pwConfig) != 2 {
			return nil, fmt.Errorf("unsupported socks link user:password: %s", passwordText)
		}
		settings.Username = pwConfig[0]
		settings.Password = pwConfig[1]
	}

	settingsRawMessage, err := convertJsonToRawMessage(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = &settingsRawMessage

	streamSettings, err := proxy.streamSettings(proxy.link)
	if err != nil {
		return nil, err
	}
	outbound.StreamSetting = streamSettings
	return outbound, nil
}

func (proxy xrayShareLink) trojanOutbound() (*conf.OutboundDetourConfig, error) {
	outbound := &conf.OutboundDetourConfig{}
	outbound.Protocol = "trojan"
	setOutboundName(outbound, proxy.link.Fragment)

	settings := &conf.TrojanClientConfig{}
	settings.Address = parseAddress(proxy.link.Hostname())
	port, err := strconv.Atoi(proxy.link.Port())
	if err != nil {
		return nil, err
	}
	settings.Port = uint16(port)

	password, err := url.QueryUnescape(proxy.link.User.String())
	if err != nil {
		return nil, err
	}
	settings.Password = password

	settingsRawMessage, err := convertJsonToRawMessage(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = &settingsRawMessage

	streamSettings, err := proxy.streamSettings(proxy.link)
	if err != nil {
		return nil, err
	}
	outbound.StreamSetting = streamSettings
	return outbound, nil
}

func (proxy xrayShareLink) hysteriaOutbound() (*conf.OutboundDetourConfig, error) {
	outbound := &conf.OutboundDetourConfig{}
	outbound.Protocol = "hysteria"
	setOutboundName(outbound, proxy.link.Fragment)

	settings := &conf.HysteriaClientConfig{}
	settings.Version = 2
	settings.Address = parseAddress(proxy.link.Hostname())
	port, err := strconv.Atoi(proxy.link.Port())
	if err != nil {
		return nil, err
	}
	settings.Port = uint16(port)

	settingsRawMessage, err := convertJsonToRawMessage(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = &settingsRawMessage

	streamSettings := &conf.StreamConfig{}
	streamSettings.Network = new(conf.TransportProtocol("hysteria"))

	hysteriaSettings := &conf.HysteriaConfig{}
	hysteriaSettings.Version = 2
	auth, err := url.QueryUnescape(proxy.link.User.String())
	if err != nil {
		return nil, err
	}
	hysteriaSettings.Auth = auth
	streamSettings.HysteriaSettings = hysteriaSettings

	query := proxy.link.Query()
	var hopPtr *int32
	if hopStr := query.Get("hop-interval"); hopStr != "" {
		interval, pErr := strconv.ParseInt(hopStr, 10, 32)
		if pErr != nil {
			return nil, pErr
		}
		hopPtr = new(int32(interval))
	}
	finalMask, mErr := buildHy2FinalMask(
		query.Get("up"), query.Get("down"), query.Get("ports"),
		hopPtr, query.Get("obfs"), query.Get("obfs-password"),
	)
	if mErr != nil {
		return nil, mErr
	}
	streamSettings.FinalMask = finalMask

	if err := proxy.parseSecurityFromURL(proxy.link, streamSettings); err != nil {
		return nil, err
	}
	outbound.StreamSetting = streamSettings
	return outbound, nil
}
