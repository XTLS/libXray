package share

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/xtls/xray-core/infra/conf"
)

// Convert XrayJson to share links.
// VMess will generate VMessAEAD link.
func ConvertXrayJsonToShareLinks(xrayBytes []byte) (string, error) {
	var xray conf.Config
	if err := json.Unmarshal(xrayBytes, &xray); err != nil {
		return "", err
	}

	outbounds := xray.OutboundConfigs
	if len(outbounds) == 0 {
		return "", fmt.Errorf("no valid outbounds")
	}

	links := make([]string, 0, len(outbounds))
	for _, outbound := range outbounds {
		link, err := shareLink(outbound)
		if err != nil || link == nil {
			continue
		}
		text := link.String()
		if text != "" {
			links = append(links, text)
		}
	}
	if len(links) == 0 {
		return "", fmt.Errorf("no valid outbounds")
	}
	shareText := strings.Join(links, "\n")
	return shareText, nil
}

func shareLink(proxy conf.OutboundDetourConfig) (*url.URL, error) {
	shareUrl := &url.URL{}

	switch proxy.Protocol {
	case "shadowsocks":
		err := shadowsocksLink(proxy, shareUrl)
		if err != nil {
			return nil, err
		}
	case "vmess":
		err := vmessLink(proxy, shareUrl)
		if err != nil {
			return nil, err
		}
	case "vless":
		err := vlessLink(proxy, shareUrl)
		if err != nil {
			return nil, err
		}
	case "socks":
		err := socksLink(proxy, shareUrl)
		if err != nil {
			return nil, err
		}
	case "trojan":
		err := trojanLink(proxy, shareUrl)
		if err != nil {
			return nil, err
		}
	default:
		return nil, fmt.Errorf("unsupported outbound protocol %q", proxy.Protocol)
	}
	streamSettingsQuery(proxy, shareUrl)

	return shareUrl, nil
}

func decodeOutboundSettings[T any](
	proxy conf.OutboundDetourConfig,
) (*T, error) {
	if proxy.Settings == nil {
		return nil, fmt.Errorf("missing %s outbound settings", proxy.Protocol)
	}
	raw := bytes.TrimSpace(*proxy.Settings)
	if len(raw) == 0 || bytes.Equal(raw, []byte("null")) {
		return nil, fmt.Errorf("missing %s outbound settings", proxy.Protocol)
	}

	var settings T
	if err := json.Unmarshal(raw, &settings); err != nil {
		return nil, fmt.Errorf(
			"invalid %s outbound settings: %w",
			proxy.Protocol,
			err,
		)
	}
	return &settings, nil
}

func shadowsocksLink(proxy conf.OutboundDetourConfig, link *url.URL) error {
	settings, err := decodeOutboundSettings[conf.ShadowsocksClientConfig](proxy)
	if err != nil {
		return err
	}

	link.Fragment = getOutboundName(proxy)
	link.Scheme = "ss"

	link.Host = fmt.Sprintf("%s:%d", settings.Address, settings.Port)
	if isShadowsocksAEAD2022(settings.Cipher) {
		userInfo := escapeShadowsocksUserInfo(settings.Cipher) + ":" +
			escapeShadowsocksUserInfo(settings.Password)
		link.Opaque = "//" + userInfo + "@" + link.Host
		link.Host = ""
	} else {
		password := fmt.Sprintf("%s:%s", settings.Cipher, settings.Password)
		username := base64.StdEncoding.EncodeToString([]byte(password))
		link.User = url.User(username)
	}

	return nil
}

func isShadowsocksAEAD2022(cipher string) bool {
	switch strings.ToLower(cipher) {
	case "2022-blake3-aes-128-gcm",
		"2022-blake3-aes-256-gcm",
		"2022-blake3-chacha20-poly1305":
		return true
	default:
		return false
	}
}

func escapeShadowsocksUserInfo(value string) string {
	return strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
}

func vmessLink(proxy conf.OutboundDetourConfig, link *url.URL) error {
	settings, err := decodeOutboundSettings[conf.VMessOutboundConfig](proxy)
	if err != nil {
		return err
	}

	link.Fragment = getOutboundName(proxy)
	link.Scheme = "vmess"

	link.Host = fmt.Sprintf("%s:%d", settings.Address, settings.Port)
	link.User = url.User(settings.ID)
	if len(settings.Security) > 0 {
		link.RawQuery = addQuery(link.RawQuery, "encryption", settings.Security)
	}

	return nil
}

func vlessLink(proxy conf.OutboundDetourConfig, link *url.URL) error {
	settings, err := decodeOutboundSettings[conf.VLessOutboundConfig](proxy)
	if err != nil {
		return err
	}

	link.Fragment = getOutboundName(proxy)
	link.Scheme = "vless"

	link.Host = fmt.Sprintf("%s:%d", settings.Address, settings.Port)
	link.User = url.User(settings.Id)
	if len(settings.Flow) > 0 {
		link.RawQuery = addQuery(link.RawQuery, "flow", settings.Flow)
	}
	if len(settings.Encryption) > 0 {
		link.RawQuery = addQuery(link.RawQuery, "encryption", settings.Encryption)
	}

	return nil
}

func socksLink(proxy conf.OutboundDetourConfig, link *url.URL) error {
	settings, err := decodeOutboundSettings[conf.SocksClientConfig](proxy)
	if err != nil {
		return err
	}

	link.Fragment = getOutboundName(proxy)
	link.Scheme = "socks"

	link.Host = fmt.Sprintf("%s:%d", settings.Address, settings.Port)
	password := fmt.Sprintf("%s:%s", settings.Username, settings.Password)
	username := base64.StdEncoding.EncodeToString([]byte(password))
	link.User = url.User(username)

	return nil
}

func trojanLink(proxy conf.OutboundDetourConfig, link *url.URL) error {
	settings, err := decodeOutboundSettings[conf.TrojanClientConfig](proxy)
	if err != nil {
		return err
	}

	link.Fragment = getOutboundName(proxy)
	link.Scheme = "trojan"

	link.Host = fmt.Sprintf("%s:%d", settings.Address, settings.Port)
	link.User = url.User(settings.Password)

	return nil
}

func streamSettingsQuery(proxy conf.OutboundDetourConfig, link *url.URL) {
	streamSettings := proxy.StreamSetting
	if streamSettings == nil {
		return
	}
	query := link.RawQuery

	network := "raw"
	if streamSettings.Network != nil {
		network = string(*streamSettings.Network)
	}
	if streamSettings.Method != nil {
		network = string(*streamSettings.Method)
	}
	if canonical, ok := canonicalShareNetwork(network); ok {
		network = canonical
	}

	shareNetwork := network
	if shareNetwork == "raw" {
		shareNetwork = "tcp"
	}
	query = addQuery(query, "type", shareNetwork)

	security := streamSettings.Security
	if security == "" {
		security = "none"
	}
	query = addQuery(query, "security", security)

	switch network {
	case "raw":
		rawSettings := streamSettings.RAWSettings
		if rawSettings == nil {
			rawSettings = streamSettings.TCPSettings
		}
		if rawSettings == nil {
			break
		}

		headerConfig := rawSettings.HeaderConfig
		if headerConfig == nil {
			break
		}
		var header XrayRawSettingsHeader
		err := json.Unmarshal(headerConfig, &header)
		if err != nil {
			break
		}

		headerType := header.Type
		if len(headerType) > 0 {
			query = addQuery(query, "headerType", headerType)
			if header.Request == nil {
				break
			}
			path := header.Request.Path
			if len(path) > 0 {
				query = addQuery(query, "path", strings.Join(path, ","))
			}
			if header.Request.Headers == nil {
				break
			}
			host := header.Request.Headers.Host
			if len(host) > 0 {
				query = addQuery(query, "host", strings.Join(host, ","))
			}
		}
	case "kcp":
		if settings := streamSettings.KCPSettings; settings != nil {
			if settings.Mtu != nil {
				query = addQuery(query, "mtu", strconv.FormatUint(uint64(*settings.Mtu), 10))
			}
			if settings.Tti != nil {
				query = addQuery(query, "tti", strconv.FormatUint(uint64(*settings.Tti), 10))
			}
		}
	case "ws":
		if streamSettings.WSSettings == nil {
			break
		}
		path := streamSettings.WSSettings.Path
		if len(path) > 0 {
			query = addQuery(query, "path", path)
		}
		host := streamSettings.WSSettings.Host
		if len(host) > 0 {
			query = addQuery(query, "host", host)
		}
	case "grpc":
		if streamSettings.GRPCSettings == nil {
			break
		}
		mode := streamSettings.GRPCSettings.MultiMode
		if mode {
			query = addQuery(query, "mode", "multi")
		} else {
			query = addQuery(query, "mode", "gun")
		}
		serviceName := streamSettings.GRPCSettings.ServiceName
		if len(serviceName) > 0 {
			query = addQuery(query, "serviceName", serviceName)
		}
		authority := streamSettings.GRPCSettings.Authority
		if len(authority) > 0 {
			query = addQuery(query, "authority", authority)
		}
	case "httpupgrade":
		if streamSettings.HTTPUPGRADESettings == nil {
			break
		}
		host := streamSettings.HTTPUPGRADESettings.Host
		if len(host) > 0 {
			query = addQuery(query, "host", host)
		}
		path := streamSettings.HTTPUPGRADESettings.Path
		if len(path) > 0 {
			query = addQuery(query, "path", path)
		}
	case "xhttp":
		xhttpSettings := streamSettings.XHTTPSettings
		if xhttpSettings == nil {
			xhttpSettings = streamSettings.SplitHTTPSettings
		}
		if xhttpSettings == nil {
			break
		}
		host := xhttpSettings.Host
		if len(host) > 0 {
			query = addQuery(query, "host", host)
		}
		path := xhttpSettings.Path
		if len(path) > 0 {
			query = addQuery(query, "path", path)
		}
		mode := xhttpSettings.Mode
		if len(mode) > 0 {
			query = addQuery(query, "mode", mode)
		}
		extra := xhttpSettings.Extra
		if extra != nil {
			query = addQuery(query, "extra", string(extra))
		}
	}

	switch streamSettings.Security {
	case "tls":
		if streamSettings.TLSSettings == nil {
			break
		}
		fp := streamSettings.TLSSettings.Fingerprint
		if len(fp) > 0 {
			query = addQuery(query, "fp", fp)
		}
		sni := streamSettings.TLSSettings.ServerName
		if len(sni) > 0 {
			query = addQuery(query, "sni", sni)
		}
		alpn := streamSettings.TLSSettings.ALPN
		if alpn != nil && len(*alpn) > 0 {
			query = addQuery(query, "alpn", strings.Join(*alpn, ","))
		}
		ech := streamSettings.TLSSettings.ECHConfigList
		if len(ech) > 0 {
			query = addQuery(query, "ech", ech)
		}
		pcs := streamSettings.TLSSettings.PinnedPeerCertSha256
		if len(pcs) > 0 {
			query = addQuery(query, "pcs", pcs)
		}
		vcn := streamSettings.TLSSettings.VerifyPeerCertByName
		if len(vcn) > 0 {
			query = addQuery(query, "vcn", vcn)
		}
	case "reality":
		if streamSettings.REALITYSettings == nil {
			break
		}
		fp := streamSettings.REALITYSettings.Fingerprint
		if len(fp) > 0 {
			query = addQuery(query, "fp", fp)
		}
		sni := streamSettings.REALITYSettings.ServerName
		if len(sni) > 0 {
			query = addQuery(query, "sni", sni)
		}
		pbk := streamSettings.REALITYSettings.Password
		if pbk == "" {
			pbk = streamSettings.REALITYSettings.PublicKey
		}
		if len(pbk) > 0 {
			query = addQuery(query, "pbk", pbk)
		}
		sid := streamSettings.REALITYSettings.ShortId
		if len(sid) > 0 {
			query = addQuery(query, "sid", sid)
		}
		pqv := streamSettings.REALITYSettings.Mldsa65Verify
		if len(pqv) > 0 {
			query = addQuery(query, "pqv", pqv)
		}
		spx := streamSettings.REALITYSettings.SpiderX
		if len(spx) > 0 {
			query = addQuery(query, "spx", spx)
		}
	}

	if streamSettings.FinalMask != nil {
		finalMask := streamSettings.FinalMask
		fmBytes, err := json.Marshal(finalMask)
		if err == nil {
			query = addQuery(query, "fm", string(fmBytes))
		}
	}

	link.RawQuery = query
}

func addQuery(rawQuery string, key, value string) string {
	v, err := url.ParseQuery(rawQuery)
	if err != nil {
		newPart := key + "=" + strings.ReplaceAll(url.QueryEscape(value), "+", "%20")
		if rawQuery == "" {
			return newPart
		}
		return rawQuery + "&" + newPart
	}
	v.Add(key, value)
	return strings.ReplaceAll(v.Encode(), "+", "%20")
}
