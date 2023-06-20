package libxray

import (
	"fmt"
	"strings"

	"encoding/base64"
	"encoding/json"
	"net/url"
	"strconv"

	"github.com/xtls/xray-core/common/platform/filesystem"
)

func ParseShareText(textPath string, xrayPath string) string {
	textBytes, err := filesystem.ReadFile(textPath)
	if err != nil {
		return err.Error()
	}
	text := string(textBytes)
	text = strings.TrimSpace(text)

	if strings.HasPrefix(text, "{") {
		var xray xrayJson

		err = json.Unmarshal(textBytes, &xray)
		if err != nil {
			return err.Error()
		}

		outbounds := xray.flattenOutbounds()
		if len(outbounds) == 0 {
			return "no valid outbounds"
		}
		xray.Outbounds = outbounds

		err = writeXrayJson(&xray, xrayPath)
		if err != nil {
			return err.Error()
		}
		return ""
	}

	text = checkWindowsReturn(text)
	if strings.HasPrefix(text, "vless://") || strings.HasPrefix(text, "vmess://") || strings.HasPrefix(text, "socks://") || strings.HasPrefix(text, "ss://") || strings.HasPrefix(text, "trojan://") {
		xray, err := parsePlainShareText(text)
		if err != nil {
			return err.Error()
		}
		err = writeXrayJson(xray, xrayPath)
		if err != nil {
			return err.Error()
		}
	} else {
		xray, err := tryParse(text)
		if err != nil {
			return err.Error()
		}
		err = writeXrayJson(xray, xrayPath)
		if err != nil {
			return err.Error()
		}
	}

	return ""
}

func checkWindowsReturn(text string) string {
	if strings.Contains(text, "\r\n") {
		text = strings.ReplaceAll(text, "\r\n", "\n")
	}
	return text
}

func parsePlainShareText(text string) (*xrayJson, error) {
	proxies := strings.Split(text, "\n")

	var xray xrayJson
	var outbounds []xrayOutbound
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

func tryParse(text string) (*xrayJson, error) {
	base64Text, err := decodeBase64Text(text)
	if err == nil {
		cleanText := checkWindowsReturn(base64Text)
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

func writeXrayJson(xray *xrayJson, xrayPath string) error {
	xrayBytes, err := json.Marshal(xray)
	if err != nil {
		return err
	}

	return writeBytes(xrayBytes, xrayPath)
}

type xrayShareLink struct {
	link    *url.URL
	rawText string
}

func (proxy xrayShareLink) outbound() (*xrayOutbound, error) {
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
	return nil, fmt.Errorf("unsupport link type: %s", proxy.link.Scheme)
}

func (proxy xrayShareLink) shadowsocksOutbound() (*xrayOutbound, error) {
	var outbound xrayOutbound
	outbound.Protocol = "shadowsocks"
	outbound.Name = proxy.link.Fragment

	var server xrayShadowsocksServer
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
	pwConfig := strings.Split(passwordText, ":")
	if len(pwConfig) != 2 {
		return nil, fmt.Errorf("unsupport link shadowsocks password: %s", passwordText)
	}
	server.Method = pwConfig[0]
	server.Password = pwConfig[1]

	var settings xrayShadowsocks
	settings.Servers = []xrayShadowsocksServer{server}

	setttingsBytes, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = (*json.RawMessage)(&setttingsBytes)

	outbound.StreamSettings = proxy.streamSettings(proxy.link.Query())

	return &outbound, nil
}

func (proxy xrayShareLink) vmessOutbound() (*xrayOutbound, error) {
	// try vmessQrCode
	text := strings.ReplaceAll(proxy.rawText, "vmess://", "")
	base64Text, err := decodeBase64Text(text)
	if err == nil {
		return parseVMessQrCode(base64Text)
	}

	var outbound xrayOutbound
	outbound.Protocol = "vmess"
	outbound.Name = proxy.link.Fragment

	query := proxy.link.Query()

	var user xrayVMessVnextUser
	user.Id = proxy.link.User.String()
	security := query.Get("encryption")
	if len(security) > 0 {
		user.Security = security
	}

	var vnext xrayVMessVnext
	vnext.Address = proxy.link.Hostname()
	port, err := strconv.Atoi(proxy.link.Port())
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

	outbound.StreamSettings = proxy.streamSettings(query)

	return &outbound, nil
}

func (proxy xrayShareLink) vlessOutbound() (*xrayOutbound, error) {
	var outbound xrayOutbound
	outbound.Protocol = "vless"
	outbound.Name = proxy.link.Fragment

	query := proxy.link.Query()

	var user xrayVLESSVnextUser
	user.Id = proxy.link.User.String()
	flow := query.Get("flow")
	if len(flow) > 0 {
		user.Flow = flow
	}

	var vnext xrayVLESSVnext
	vnext.Address = proxy.link.Hostname()
	port, err := strconv.Atoi(proxy.link.Port())
	if err != nil {
		return nil, err
	}
	vnext.Port = port
	vnext.Users = []xrayVLESSVnextUser{user}

	var settings xrayVLESS
	settings.Vnext = []xrayVLESSVnext{vnext}

	setttingsBytes, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = (*json.RawMessage)(&setttingsBytes)

	outbound.StreamSettings = proxy.streamSettings(query)

	return &outbound, nil
}

func (proxy xrayShareLink) socksOutbound() (*xrayOutbound, error) {
	var outbound xrayOutbound
	outbound.Protocol = "socks"
	outbound.Name = proxy.link.Fragment

	userPassword := proxy.link.User.String()
	passwordText, err := decodeBase64Text(userPassword)
	if err != nil {
		return nil, err
	}
	pwConfig := strings.Split(passwordText, ":")
	if len(pwConfig) != 2 {
		return nil, fmt.Errorf("unsupport link socks user password: %s", passwordText)
	}
	var user xraySocksServerUser
	user.User = pwConfig[0]
	user.Pass = pwConfig[1]

	var server xraySocksServer
	server.Address = proxy.link.Hostname()
	port, err := strconv.Atoi(proxy.link.Port())
	if err != nil {
		return nil, err
	}
	server.Port = port
	server.Users = []xraySocksServerUser{user}

	var settings xraySocks
	settings.Servers = []xraySocksServer{server}

	setttingsBytes, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = (*json.RawMessage)(&setttingsBytes)

	outbound.StreamSettings = proxy.streamSettings(proxy.link.Query())

	return &outbound, nil
}

func (proxy xrayShareLink) trojanOutbound() (*xrayOutbound, error) {
	var outbound xrayOutbound
	outbound.Protocol = "trojan"
	outbound.Name = proxy.link.Fragment

	var server xrayTrojanServer
	server.Address = proxy.link.Hostname()
	port, err := strconv.Atoi(proxy.link.Port())
	if err != nil {
		return nil, err
	}
	server.Port = port
	server.Password = proxy.link.User.String()

	var settings xrayTrojan
	settings.Servers = []xrayTrojanServer{server}

	setttingsBytes, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = (*json.RawMessage)(&setttingsBytes)

	outbound.StreamSettings = proxy.streamSettings(proxy.link.Query())

	return &outbound, nil
}

func (proxy xrayShareLink) streamSettings(query url.Values) *xrayStreamSettings {
	var streamSettings xrayStreamSettings
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
			var request xrayTcpSettingsHeaderRequest
			path := query.Get("path")
			if len(path) > 0 {
				request.Path = strings.Split(path, ",")
			}
			host := query.Get("host")
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
		headerType := query.Get("headerType")
		if len(headerType) > 0 {
			var header xrayFakeHeader
			header.Type = headerType
			kcpSettings.Header = &header
		}
		seed := query.Get("seed")
		kcpSettings.Seed = seed

		streamSettings.KcpSettings = &kcpSettings
	case "ws":
		var wsSettings xrayWsSettings
		path := query.Get("path")
		wsSettings.Path = path
		host := query.Get("host")
		if len(host) > 0 {
			var headers xrayWsSettingsHeaders
			headers.Host = host
			wsSettings.Headers = &headers
		}

		streamSettings.WsSettings = &wsSettings
	case "grpc":
		var grcpSettings xrayGrpcSettings
		serviceName := query.Get("serviceName")
		grcpSettings.ServiceName = serviceName
		mode := query.Get("mode")
		grcpSettings.MultiMode = mode == "multi"

		streamSettings.GrpcSettings = &grcpSettings
	case "quic":
		var quicSettings xrayQuicSettings
		headerType := query.Get("headerType")
		if len(headerType) > 0 {
			var header xrayFakeHeader
			header.Type = headerType
			quicSettings.Header = &header
		}
		quicSecurity := query.Get("quicSecurity")
		quicSettings.Security = quicSecurity
		key := query.Get("key")
		quicSettings.Key = key

		streamSettings.QuicSettings = &quicSettings
	case "http":
		var httpSettings xrayHttpSettings
		host := query.Get("host")
		httpSettings.Host = strings.Split(host, ",")
		path := query.Get("path")
		httpSettings.Path = path

		streamSettings.HttpSettings = &httpSettings
	}

	proxy.parseSecurity(query, &streamSettings)

	return &streamSettings
}

func (proxy xrayShareLink) parseSecurity(query url.Values, streamSettings *xrayStreamSettings) {
	var tlsSettings xrayTlsSettings
	var realitySettings xrayRealitySettings

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

	switch streamSettings.Security {
	case "tls":
		streamSettings.TlsSettings = &tlsSettings
	case "reality":
		streamSettings.RealitySettings = &realitySettings
	}
}
