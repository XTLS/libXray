package nodep

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// https://github.com/XTLS/Xray-core/discussions/716
// Convert share text to XrayJson
// support v2rayN plain text, v2rayN base64 text
func ConvertShareLinksToXrayJson(links string) (*XrayJson, error) {
	text := strings.TrimSpace(links)
	if strings.HasPrefix(text, "{") {
		var xray XrayJson
		err := json.Unmarshal([]byte(text), &xray)
		if err != nil {
			return nil, err
		}

		outbounds := xray.FlattenOutbounds()
		if len(outbounds) == 0 {
			return nil, fmt.Errorf("no valid outbounds")
		}
		xray.Outbounds = outbounds

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

func parsePlainShareText(text string) (*XrayJson, error) {
	proxies := strings.Split(text, "\n")

	var xray XrayJson
	var outbounds []XrayOutbound
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
	xray.Outbounds = outbounds
	return &xray, nil
}

func tryParse(text string) (*XrayJson, error) {
	base64Text, err := decodeBase64Text(text)
	if err == nil {
		cleanText := FixWindowsReturn(base64Text)
		return parsePlainShareText(cleanText)
	}
	return tryConvertClashYaml(text)
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

func (proxy xrayShareLink) outbound() (*XrayOutbound, error) {
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

func (proxy xrayShareLink) shadowsocksOutbound() (*XrayOutbound, error) {
	var outbound XrayOutbound
	outbound.Protocol = "shadowsocks"
	outbound.Name = proxy.link.Fragment

	var server XrayShadowsocksServer
	server.Address = proxy.link.Hostname()
	port, err := strconv.Atoi(proxy.link.Port())
	if err != nil {
		return nil, err
	}
	server.Port = port

	user := proxy.link.User.String()
	passwordText, err := decodeBase64Text(user)
	if err != nil {
		return nil, err
	}
	pwConfig := strings.SplitN(passwordText, ":", 2)
	if len(pwConfig) != 2 {
		return nil, fmt.Errorf("unsupport link shadowsocks password: %s", passwordText)
	}
	server.Method = pwConfig[0]
	server.Password = pwConfig[1]

	var settings XrayShadowsocks
	settings.Servers = []XrayShadowsocksServer{server}

	setttingsBytes, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = (*json.RawMessage)(&setttingsBytes)

	outbound.StreamSettings = proxy.streamSettings(proxy.link)

	return &outbound, nil
}

func (proxy xrayShareLink) vmessOutbound() (*XrayOutbound, error) {
	// try vmessQrCode
	text := strings.ReplaceAll(proxy.rawText, "vmess://", "")
	base64Text, err := decodeBase64Text(text)
	if err == nil {
		return parseVMessQrCode(base64Text)
	}

	var outbound XrayOutbound
	outbound.Protocol = "vmess"
	outbound.Name = proxy.link.Fragment

	query := proxy.link.Query()

	var user XrayVMessVnextUser
	id, err := url.QueryUnescape(proxy.link.User.String())
	if err != nil {
		return nil, err
	}
	user.Id = id
	security := query.Get("encryption")
	if len(security) > 0 {
		user.Security = security
	}

	var vnext XrayVMessVnext
	vnext.Address = proxy.link.Hostname()
	port, err := strconv.Atoi(proxy.link.Port())
	if err != nil {
		return nil, err
	}
	vnext.Port = port
	vnext.Users = []XrayVMessVnextUser{user}

	var settings XrayVMess
	settings.Vnext = []XrayVMessVnext{vnext}

	setttingsBytes, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = (*json.RawMessage)(&setttingsBytes)

	outbound.StreamSettings = proxy.streamSettings(proxy.link)

	return &outbound, nil
}

func (proxy xrayShareLink) vlessOutbound() (*XrayOutbound, error) {
	var outbound XrayOutbound
	outbound.Protocol = "vless"
	outbound.Name = proxy.link.Fragment

	query := proxy.link.Query()

	var user XrayVLESSVnextUser
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

	var vnext XrayVLESSVnext
	vnext.Address = proxy.link.Hostname()
	port, err := strconv.Atoi(proxy.link.Port())
	if err != nil {
		return nil, err
	}
	vnext.Port = port
	vnext.Users = []XrayVLESSVnextUser{user}

	var settings XrayVLESS
	settings.Vnext = []XrayVLESSVnext{vnext}

	setttingsBytes, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = (*json.RawMessage)(&setttingsBytes)

	outbound.StreamSettings = proxy.streamSettings(proxy.link)

	return &outbound, nil
}

func (proxy xrayShareLink) socksOutbound() (*XrayOutbound, error) {
	var outbound XrayOutbound
	outbound.Protocol = "socks"
	outbound.Name = proxy.link.Fragment

	userPassword := proxy.link.User.String()
	passwordText, err := decodeBase64Text(userPassword)
	if err != nil {
		return nil, err
	}
	pwConfig := strings.SplitN(passwordText, ":", 2)
	if len(pwConfig) != 2 {
		return nil, fmt.Errorf("unsupport link socks user password: %s", passwordText)
	}
	var user XraySocksServerUser
	user.User = pwConfig[0]
	user.Pass = pwConfig[1]

	var server XraySocksServer
	server.Address = proxy.link.Hostname()
	port, err := strconv.Atoi(proxy.link.Port())
	if err != nil {
		return nil, err
	}
	server.Port = port
	server.Users = []XraySocksServerUser{user}

	var settings XraySocks
	settings.Servers = []XraySocksServer{server}

	setttingsBytes, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = (*json.RawMessage)(&setttingsBytes)

	outbound.StreamSettings = proxy.streamSettings(proxy.link)

	return &outbound, nil
}

func (proxy xrayShareLink) trojanOutbound() (*XrayOutbound, error) {
	var outbound XrayOutbound
	outbound.Protocol = "trojan"
	outbound.Name = proxy.link.Fragment

	var server XrayTrojanServer
	server.Address = proxy.link.Hostname()
	port, err := strconv.Atoi(proxy.link.Port())
	if err != nil {
		return nil, err
	}
	server.Port = port
	password, err := url.QueryUnescape(proxy.link.User.String())
	if err != nil {
		return nil, err
	}
	server.Password = password

	var settings XrayTrojan
	settings.Servers = []XrayTrojanServer{server}

	setttingsBytes, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = (*json.RawMessage)(&setttingsBytes)

	outbound.StreamSettings = proxy.streamSettings(proxy.link)

	return &outbound, nil
}

func (proxy xrayShareLink) streamSettings(link *url.URL) *XrayStreamSettings {
	query := link.Query()
	var streamSettings XrayStreamSettings
	if len(query) == 0 {
		return &streamSettings
	}
	network := query.Get("type")
	if len(network) == 0 {
		streamSettings.Network = "tcp"
	} else {
		streamSettings.Network = network
	}

	switch streamSettings.Network {
	case "tcp":
		headerType := query.Get("headerType")
		if headerType == "http" {
			var request XrayTcpSettingsHeaderRequest
			path := query.Get("path")
			if len(path) > 0 {
				request.Path = strings.Split(path, ",")
			}
			host := query.Get("host")
			if len(host) > 0 {
				var headers XrayTcpSettingsHeaderRequestHeaders
				headers.Host = strings.Split(host, ",")
				request.Headers = &headers
			}
			var header XrayTcpSettingsHeader
			header.Type = headerType
			header.Request = &request

			var tcpSettings XrayTcpSettings
			tcpSettings.Header = &header

			streamSettings.TcpSettings = &tcpSettings
		}
	case "kcp", "mkcp":
		var kcpSettings XrayKcpSettings
		headerType := query.Get("headerType")
		if len(headerType) > 0 {
			var header XrayFakeHeader
			header.Type = headerType
			kcpSettings.Header = &header
		}
		kcpSettings.Seed = query.Get("seed")

		streamSettings.KcpSettings = &kcpSettings
	case "ws", "websocket":
		var wsSettings XrayWsSettings
		wsSettings.Path = query.Get("path")
		wsSettings.Host = query.Get("host")
		streamSettings.WsSettings = &wsSettings
	case "grpc", "gun":
		var grcpSettings XrayGrpcSettings
		grcpSettings.Authority = query.Get("authority")
		grcpSettings.ServiceName = query.Get("serviceName")
		grcpSettings.MultiMode = query.Get("mode") == "multi"

		streamSettings.GrpcSettings = &grcpSettings
	case "h2", "http":
		var httpSettings XrayHttpSettings
		host := query.Get("host")
		httpSettings.Host = strings.Split(host, ",")
		httpSettings.Path = query.Get("path")

		streamSettings.HttpSettings = &httpSettings
	case "httpupgrade":
		var httpupgradeSettings XrayHttpupgradeSettings
		httpupgradeSettings.Host = query.Get("host")
		httpupgradeSettings.Path = query.Get("path")

		streamSettings.HttpupgradeSettings = &httpupgradeSettings
	case "splithttp":
		var splithttpSettings XraySplithttpSettings
		splithttpSettings.Host = query.Get("host")
		splithttpSettings.Path = query.Get("path")

		streamSettings.SplithttpSettings = &splithttpSettings
	}

	proxy.parseSecurity(link, &streamSettings)

	return &streamSettings
}

func (proxy xrayShareLink) parseSecurity(link *url.URL, streamSettings *XrayStreamSettings) {
	query := link.Query()

	var tlsSettings XrayTlsSettings
	var realitySettings XrayRealitySettings

	fp := query.Get("fp")
	tlsSettings.Fingerprint = fp
	realitySettings.Fingerprint = fp

	sni := query.Get("sni")
	tlsSettings.ServerName = sni
	realitySettings.ServerName = sni

	alpn := query.Get("alpn")
	if len(alpn) > 0 {
		tlsSettings.Alpn = strings.Split(alpn, ",")
	}

	// https://github.com/XTLS/Xray-core/discussions/716
	// 4.4.3 allowInsecure
	// 没有这个字段。不安全的节点，不适合分享。
	// I don't like this field, but too many people ask for it.
	allowInsecure := query.Get("allowInsecure")
	if len(allowInsecure) > 0 {
		if allowInsecure == "true" || allowInsecure == "1" {
			tlsSettings.AllowInsecure = true
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
	if streamSettings.Network == "ws" && len(tlsSettings.ServerName) == 0 {
		if streamSettings.WsSettings != nil && len(streamSettings.WsSettings.Host) > 0 {
			tlsSettings.ServerName = streamSettings.WsSettings.Host
		}
	}

	switch streamSettings.Security {
	case "tls":
		streamSettings.TlsSettings = &tlsSettings
	case "reality":
		streamSettings.RealitySettings = &realitySettings
	}
}
