package share

import (
	"encoding/base64"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/xtls/xray-core/infra/conf"
)

// FixWindowsReturn normalizes CRLF to LF.
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
	// Normalize the URL-safe alphabet and restore missing padding.
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

var shareSchemes = []string{
	"vless://", "vmess://", "socks://", "ss://", "trojan://",
}

func hasShareSchemeLine(text string) bool {
	for raw := range strings.SplitSeq(text, "\n") {
		line := strings.TrimSpace(raw)
		for _, p := range shareSchemes {
			if strings.HasPrefix(line, p) {
				return true
			}
		}
	}
	return false
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

	cipher, password, err := parseShadowsocksUserInfo(proxy.link.User)
	if err != nil {
		return nil, err
	}
	settings.Cipher = cipher
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

func parseShadowsocksUserInfo(user *url.Userinfo) (string, string, error) {
	if user == nil {
		return "", "", fmt.Errorf("missing shadowsocks user info")
	}

	// SIP002 plain user info uses method:password. net/url decodes the
	// percent-encoded method and password independently.
	if password, ok := user.Password(); ok {
		cipher := user.Username()
		if cipher == "" {
			return "", "", fmt.Errorf("missing shadowsocks cipher")
		}
		return cipher, password, nil
	}

	decoded, err := decodeBase64Text(user.String())
	if err != nil {
		return "", "", err
	}
	cipher, password, ok := strings.Cut(decoded, ":")
	if !ok || cipher == "" {
		return "", "", fmt.Errorf("unsupported shadowsocks user info")
	}
	return cipher, password, nil
}

func (proxy xrayShareLink) vmessOutbound() (*conf.OutboundDetourConfig, error) {
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
