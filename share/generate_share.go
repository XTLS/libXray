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
	case "hysteria":
		err := hysteriaLink(proxy, shareUrl)
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

func hysteriaLink(proxy conf.OutboundDetourConfig, link *url.URL) error {
	settings, err := decodeOutboundSettings[conf.HysteriaClientConfig](proxy)
	if err != nil {
		return err
	}

	link.Fragment = getOutboundName(proxy)
	link.Scheme = "hysteria2"

	link.Host = fmt.Sprintf("%s:%d", settings.Address, settings.Port)

	if proxy.StreamSetting != nil && proxy.StreamSetting.HysteriaSettings != nil {
		link.User = url.User(proxy.StreamSetting.HysteriaSettings.Auth)
	}

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

	if network == "hysteria" {
		// TLS params
		if streamSettings.TLSSettings != nil {
			sni := streamSettings.TLSSettings.ServerName
			if len(sni) > 0 {
				query = addQuery(query, "sni", sni)
			}
			fp := streamSettings.TLSSettings.Fingerprint
			if len(fp) > 0 {
				query = addQuery(query, "fp", fp)
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
			if streamSettings.TLSSettings.AllowInsecure {
				query = addQuery(query, "insecure", "1")
			}
		}

		// QuicParams (bandwidth + port-hopping)
		if streamSettings.FinalMask != nil && streamSettings.FinalMask.QuicParams != nil {
			qp := streamSettings.FinalMask.QuicParams
			if len(qp.BrutalUp) > 0 {
				query = addQuery(query, "up", string(qp.BrutalUp))
			}
			if len(qp.BrutalDown) > 0 {
				query = addQuery(query, "down", string(qp.BrutalDown))
			}
			if len(qp.UdpHop.PortList.Range) > 0 {
				query = addQuery(query, "ports", qp.UdpHop.PortList.String())
			}
			if qp.UdpHop.Interval.From != 0 || qp.UdpHop.Interval.To != 0 {
				query = addQuery(query, "hop-interval", strconv.FormatInt(int64(qp.UdpHop.Interval.From), 10))
			}
		}

		// Salamander
		if streamSettings.FinalMask != nil && len(streamSettings.FinalMask.Udp) > 0 {
			mask := streamSettings.FinalMask.Udp[0]
			if mask.Settings != nil {
				var obfs conf.Salamander
				err := json.Unmarshal(*mask.Settings, &obfs)
				if err == nil {
					query = addQuery(query, "obfs", "salamander")
					query = addQuery(query, "obfs-password", obfs.Password)
				}
			}
		}
		link.RawQuery = query
		return
	}

	query = addQuery(query, "type", network)

	if len(streamSettings.Security) == 0 {
		streamSettings.Security = "none"
	}
	query = addQuery(query, "security", streamSettings.Security)

	switch network {
	case "raw":
		if streamSettings.RAWSettings == nil {
			break
		}

		headerConfig := streamSettings.RAWSettings.HeaderConfig
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
		if streamSettings.KCPSettings == nil {
			break
		}
		seed := streamSettings.KCPSettings.Seed
		if seed != nil && len(*seed) > 0 {
			query = addQuery(query, "seed", *seed)
		}

		headerConfig := streamSettings.KCPSettings.HeaderConfig
		if headerConfig == nil {
			break
		}
		var header XrayFakeHeader
		err := json.Unmarshal(headerConfig, &header)
		if err != nil {
			break
		}

		headerType := header.Type
		if len(headerType) > 0 {
			query = addQuery(query, "headerType", headerType)
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
		if streamSettings.XHTTPSettings == nil {
			break
		}
		host := streamSettings.XHTTPSettings.Host
		if len(host) > 0 {
			query = addQuery(query, "host", host)
		}
		path := streamSettings.XHTTPSettings.Path
		if len(path) > 0 {
			query = addQuery(query, "path", path)
		}
		mode := streamSettings.XHTTPSettings.Mode
		if len(mode) > 0 {
			query = addQuery(query, "mode", mode)
		}
		extra := streamSettings.XHTTPSettings.Extra
		if extra != nil {
			var extraConfig conf.SplitHTTPConfig
			err := json.Unmarshal(extra, &extraConfig)
			if err == nil {
				extraBytes, err := json.Marshal(extraConfig)
				if err == nil {
					query = addQuery(query, "extra", string(extraBytes))
				}
			}
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
		if streamSettings.TLSSettings.AllowInsecure {
			query = addQuery(query, "insecure", "1")
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
		newPart := key + "=" + url.QueryEscape(value)
		if rawQuery == "" {
			return newPart
		}
		return rawQuery + "&" + newPart
	}
	v.Add(key, value)
	return v.Encode()
}
