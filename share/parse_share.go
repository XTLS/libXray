package share

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/xtls/xray-core/infra/conf"
	"github.com/xtls/xray-core/proxy/vless"
)

// https://github.com/XTLS/Xray-core/discussions/716
// Convert share text to XrayJson
// support v2rayN plain text, v2rayN base64 text
func ConvertShareLinksToXrayJson(links string) (*conf.Config, error) {
	text := strings.TrimSpace(links)
	if strings.HasPrefix(text, "{") {
		var xray conf.Config
		err := json.Unmarshal([]byte(text), &xray)
		if err != nil {
			return nil, err
		}

		outbounds := xray.OutboundConfigs
		if len(outbounds) == 0 {
			return nil, fmt.Errorf("no valid outbounds")
		}

		return &xray, nil
	}

	text = FixWindowsReturn(text)
	if strings.HasPrefix(text, "vless://") || strings.HasPrefix(text, "vmess://") || strings.HasPrefix(text, "socks://") || strings.HasPrefix(text, "ss://") || strings.HasPrefix(text, "trojan://") {
		xray, err := parsePlainShareText(text)
		if err != nil {
			return xray, err
		}
		return xray, nil
	} else {
		xray, err := tryParse(text)
		if err != nil {
			return nil, err
		}
		return xray, nil
	}
}

func FixWindowsReturn(text string) string {
	if strings.Contains(text, "\r\n") {
		text = strings.ReplaceAll(text, "\r\n", "\n")
	}
	return text
}

func parsePlainShareText(text string) (*conf.Config, error) {
	proxies := strings.Split(text, "\n")

	xray := &conf.Config{}
	var outbounds []conf.OutboundDetourConfig
	for _, proxy := range proxies {
		link, err := url.Parse(proxy)
		if err == nil {
			var shareLink xrayShareLink
			shareLink.link = link
			shareLink.rawText = proxy
			if outbound, err := shareLink.outbound(); err == nil {
				outbounds = append(outbounds, *outbound)
			} else {
				fmt.Println(err)
			}
		}
	}
	if len(outbounds) == 0 {
		return nil, fmt.Errorf("no valid outbound found")
	}
	xray.OutboundConfigs = outbounds
	return xray, nil
}

func tryParse(text string) (*conf.Config, error) {
	base64Text, err := decodeBase64Text(text)
	if err == nil {
		cleanText := FixWindowsReturn(base64Text)
		return parsePlainShareText(cleanText)
	}
	return tryToParseClashYaml(text)
}

func decodeBase64Text(text string) (string, error) {
	content, err := base64.StdEncoding.DecodeString(text)
	if err == nil {
		return string(content), nil
	}
	if strings.Contains(text, "-") {
		text = strings.ReplaceAll(text, "-", "+")
	}
	if strings.Contains(text, "_") {
		text = strings.ReplaceAll(text, "_", "/")
	}
	missingPadding := len(text) % 4
	if missingPadding != 0 {
		padding := strings.Repeat("=", 4-missingPadding)
		text += padding
	}
	content, err = base64.StdEncoding.DecodeString(text)
	if err != nil {
		return "", err
	}
	return string(content), nil
}

type xrayShareLink struct {
	link    *url.URL
	rawText string
}

func (proxy xrayShareLink) outbound() (*conf.OutboundDetourConfig, error) {
	switch proxy.link.Scheme {
	case "ss":
		outbound, err := proxy.shadowsocksOutbound()
		if err != nil {
			return nil, err
		}
		return outbound, nil

	case "vmess":
		outbound, err := proxy.vmessOutbound()
		if err != nil {
			return nil, err
		}
		return outbound, nil

	case "vless":
		outbound, err := proxy.vlessOutbound()
		if err != nil {
			return nil, err
		}
		return outbound, nil

	case "socks":
		outbound, err := proxy.socksOutbound()
		if err != nil {
			return nil, err
		}
		return outbound, nil
	case "trojan":
		outbound, err := proxy.trojanOutbound()
		if err != nil {
			return nil, err
		}
		return outbound, nil
	}
	return nil, fmt.Errorf("unsupport link: %s", proxy.rawText)
}

func (proxy xrayShareLink) shadowsocksOutbound() (*conf.OutboundDetourConfig, error) {
	outbound := &conf.OutboundDetourConfig{}
	outbound.Protocol = "shadowsocks"
	setOutboundName(outbound, proxy.link.Fragment)

	server := &conf.ShadowsocksServerTarget{}
	server.Address = parseAddress(proxy.link.Hostname())
	port, err := strconv.Atoi(proxy.link.Port())
	if err != nil {
		return nil, err
	}
	server.Port = uint16(port)

	user := proxy.link.User.String()
	passwordText, err := decodeBase64Text(user)
	if err != nil {
		return nil, err
	}
	pwConfig := strings.SplitN(passwordText, ":", 2)
	if len(pwConfig) != 2 {
		return nil, fmt.Errorf("unsupport link shadowsocks password: %s", passwordText)
	}
	server.Cipher = pwConfig[0]
	server.Password = pwConfig[1]

	var settings conf.ShadowsocksClientConfig
	settings.Servers = []*conf.ShadowsocksServerTarget{server}

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
	// try vmessQrCode
	text := strings.ReplaceAll(proxy.rawText, "vmess://", "")
	base64Text, err := decodeBase64Text(text)
	if err == nil {
		return parseVMessQrCode(base64Text)
	}

	outbound := &conf.OutboundDetourConfig{}
	outbound.Protocol = "vmess"
	setOutboundName(outbound, proxy.link.Fragment)

	query := proxy.link.Query()

	user := &conf.VMessAccount{}
	id, err := url.QueryUnescape(proxy.link.User.String())
	if err != nil {
		return nil, err
	}
	user.ID = id
	security := query.Get("encryption")
	if len(security) > 0 {
		user.Security = security
	}

	vnext := &conf.VMessOutboundTarget{}
	vnext.Address = parseAddress(proxy.link.Hostname())
	port, err := strconv.Atoi(proxy.link.Port())
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

	user := &vless.Account{}
	id, err := url.QueryUnescape(proxy.link.User.String())
	if err != nil {
		return nil, err
	}
	user.Id = id
	flow := query.Get("flow")
	if len(flow) > 0 {
		user.Flow = flow
	}

	encryption := query.Get("encryption")
	if len(encryption) > 0 {
		user.Encryption = encryption
	} else {
		user.Encryption = "none"
	}

	vnext := &conf.VLessOutboundVnext{}
	vnext.Address = parseAddress(proxy.link.Hostname())
	port, err := strconv.Atoi(proxy.link.Port())
	if err != nil {
		return nil, err
	}
	vnext.Port = uint16(port)

	userRawMessage, err := convertJsonToRawMessage(user)
	if err != nil {
		return nil, err
	}
	vnext.Users = []json.RawMessage{userRawMessage}

	settings := &conf.VLessOutboundConfig{}
	settings.Vnext = []*conf.VLessOutboundVnext{vnext}

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

	users := []json.RawMessage{}

	userPassword := proxy.link.User.String()
	if len(userPassword) > 0 {
		passwordText, err := decodeBase64Text(userPassword)
		if err != nil {
			return nil, err
		}
		pwConfig := strings.SplitN(passwordText, ":", 2)
		if len(pwConfig) != 2 {
			return nil, fmt.Errorf("unsupport link socks user password: %s", passwordText)
		}

		user := &conf.SocksAccount{}
		user.Username = pwConfig[0]
		user.Password = pwConfig[1]

		userRawMessage, err := convertJsonToRawMessage(user)
		if err != nil {
			return nil, err
		}

		users = append(users, userRawMessage)
	}

	server := &conf.SocksRemoteConfig{}
	server.Address = parseAddress(proxy.link.Hostname())
	port, err := strconv.Atoi(proxy.link.Port())
	if err != nil {
		return nil, err
	}
	server.Port = uint16(port)
	server.Users = users

	settings := &conf.SocksClientConfig{}
	settings.Servers = []*conf.SocksRemoteConfig{server}

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

	server := &conf.TrojanServerTarget{}
	server.Address = parseAddress(proxy.link.Hostname())
	port, err := strconv.Atoi(proxy.link.Port())
	if err != nil {
		return nil, err
	}
	server.Port = uint16(port)

	password, err := url.QueryUnescape(proxy.link.User.String())
	if err != nil {
		return nil, err
	}
	server.Password = password

	settings := &conf.TrojanClientConfig{}
	settings.Servers = []*conf.TrojanServerTarget{server}

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

func (proxy xrayShareLink) streamSettings(link *url.URL) (*conf.StreamConfig, error) {
	query := link.Query()
	if len(query) == 0 {
		return nil, nil
	}

	streamSettings := &conf.StreamConfig{}
	network := query.Get("type")
	if len(network) == 0 {
		network = "raw"
	}
	transportProtocol := conf.TransportProtocol(network)
	streamSettings.Network = &transportProtocol

	switch network {
	case "raw", "tcp":
		headerType := query.Get("headerType")
		if headerType == "http" {
			var request XrayRawSettingsHeaderRequest
			path := query.Get("path")
			if len(path) > 0 {
				request.Path = strings.Split(path, ",")
			}
			host := query.Get("host")
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
		headerType := query.Get("headerType")
		if len(headerType) > 0 {
			var header XrayFakeHeader
			header.Type = headerType

			headerRawMessage, err := convertJsonToRawMessage(header)
			if err != nil {
				return nil, err
			}
			kcpSettings.HeaderConfig = headerRawMessage
		}
		seed := query.Get("seed")
		kcpSettings.Seed = &seed

		streamSettings.KCPSettings = kcpSettings
	case "ws", "websocket":
		wsSettings := &conf.WebSocketConfig{}
		wsSettings.Path = query.Get("path")
		wsSettings.Host = query.Get("host")

		streamSettings.WSSettings = wsSettings
	case "grpc", "gun":
		grcpSettings := &conf.GRPCConfig{}
		grcpSettings.Authority = query.Get("authority")
		grcpSettings.ServiceName = query.Get("serviceName")
		grcpSettings.MultiMode = query.Get("mode") == "multi"

		streamSettings.GRPCSettings = grcpSettings
	case "httpupgrade":
		httpupgradeSettings := &conf.HttpUpgradeConfig{}
		httpupgradeSettings.Host = query.Get("host")
		httpupgradeSettings.Path = query.Get("path")

		streamSettings.HTTPUPGRADESettings = httpupgradeSettings
	case "xhttp", "splithttp":
		xhttpSettings := &conf.SplitHTTPConfig{}
		xhttpSettings.Host = query.Get("host")
		xhttpSettings.Path = query.Get("path")
		xhttpSettings.Mode = query.Get("mode")

		extra := query.Get("extra")
		if len(extra) > 0 {
			var extraConfig conf.SplitHTTPConfig
			err := json.Unmarshal([]byte(extra), &extraConfig)
			if err != nil {
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

	err := proxy.parseSecurity(link, streamSettings)
	if err != nil {
		return nil, err
	}

	return streamSettings, nil
}

func (proxy xrayShareLink) parseSecurity(link *url.URL, streamSettings *conf.StreamConfig) error {
	query := link.Query()

	tlsSettings := &conf.TLSConfig{}
	realitySettings := &conf.REALITYConfig{}

	fp := query.Get("fp")
	tlsSettings.Fingerprint = fp
	realitySettings.Fingerprint = fp

	sni := query.Get("sni")
	tlsSettings.ServerName = sni
	realitySettings.ServerName = sni

	alpn := query.Get("alpn")
	if len(alpn) > 0 {
		alpn := conf.StringList(strings.Split(alpn, ","))
		tlsSettings.ALPN = &alpn
	}

	// https://github.com/XTLS/Xray-core/discussions/716
	// 4.4.3 allowInsecure
	// 没有这个字段。不安全的节点，不适合分享。
	// I don't like this field, but too many people ask for it.
	allowInsecure := query.Get("allowInsecure")
	if len(allowInsecure) > 0 {
		if allowInsecure == "true" || allowInsecure == "1" {
			tlsSettings.Insecure = true
		}
	}

	pbk := query.Get("pbk")
	realitySettings.PublicKey = pbk
	sid := query.Get("sid")
	realitySettings.ShortId = sid
	spx := query.Get("spx")
	realitySettings.SpiderX = spx

	security := query.Get("security")
	if len(security) == 0 {
		streamSettings.Security = "none"
	} else {
		streamSettings.Security = security
	}

	// some link omits too many params, here is some fixing
	if proxy.link.Scheme == "trojan" && streamSettings.Security == "none" {
		streamSettings.Security = "tls"
	}

	network, err := streamSettings.Network.Build()
	if err != nil {
		return err
	}
	if network == "websocket" && len(tlsSettings.ServerName) == 0 {
		if streamSettings.WSSettings != nil && len(streamSettings.WSSettings.Host) > 0 {
			tlsSettings.ServerName = streamSettings.WSSettings.Host
		}
	}

	switch streamSettings.Security {
	case "tls":
		streamSettings.TLSSettings = tlsSettings
	case "reality":
		streamSettings.REALITYSettings = realitySettings
	}
	return nil
}
