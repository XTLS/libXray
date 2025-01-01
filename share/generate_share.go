package share

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/xtls/xray-core/infra/conf"
	"github.com/xtls/xray-core/proxy/vless"
)

// Convert XrayJson to share links.
// VMess will generate VMessAEAD link.
func ConvertXrayJsonToShareLinks(xrayBytes []byte) (string, error) {
	var xray conf.Config

	err := json.Unmarshal(xrayBytes, &xray)
	if err != nil {
		return "", err
	}

	outbounds := xray.OutboundConfigs
	if len(outbounds) == 0 {
		return "", fmt.Errorf("no valid outbounds")
	}

	var links []string
	for _, outbound := range outbounds {
		link, err := shareLink(outbound)
		if err == nil {
			links = append(links, link.String())
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
	}
	streamSettingsQuery(proxy, shareUrl)

	return shareUrl, nil
}

func shadowsocksLink(proxy conf.OutboundDetourConfig, link *url.URL) error {
	var settings conf.ShadowsocksClientConfig
	err := json.Unmarshal(*proxy.Settings, &settings)
	if err != nil {
		return err
	}

	link.Fragment = getOutboundName(proxy)
	link.Scheme = "ss"

	if len(settings.Servers) > 0 {
		server := settings.Servers[0]
		link.Host = fmt.Sprintf("%s:%d", server.Address, server.Port)
		password := fmt.Sprintf("%s:%s", server.Cipher, server.Password)
		username := base64.StdEncoding.EncodeToString([]byte(password))
		link.User = url.User(username)
	}
	return nil
}

func vmessLink(proxy conf.OutboundDetourConfig, link *url.URL) error {
	var settings conf.VMessOutboundConfig
	err := json.Unmarshal(*proxy.Settings, &settings)
	if err != nil {
		return err
	}

	link.Fragment = getOutboundName(proxy)
	link.Scheme = "vmess"

	if len(settings.Receivers) > 0 {
		vnext := settings.Receivers[0]
		link.Host = fmt.Sprintf("%s:%d", vnext.Address, vnext.Port)
		if len(vnext.Users) > 0 {
			user := vnext.Users[0]
			var account conf.VMessAccount
			err := json.Unmarshal(user, &account)
			if err != nil {
				return err
			}
			link.User = url.User(account.ID)
			link.RawQuery = addQuery(link.RawQuery, "encryption", account.Security)
		}
	}
	return nil
}

func vlessLink(proxy conf.OutboundDetourConfig, link *url.URL) error {
	var settings conf.VLessOutboundConfig
	err := json.Unmarshal(*proxy.Settings, &settings)
	if err != nil {
		return err
	}

	link.Fragment = getOutboundName(proxy)
	link.Scheme = "vless"

	if len(settings.Vnext) > 0 {
		vnext := settings.Vnext[0]
		link.Host = fmt.Sprintf("%s:%d", vnext.Address, vnext.Port)
		if len(vnext.Users) > 0 {
			user := vnext.Users[0]
			var account vless.Account
			err := json.Unmarshal(user, &account)
			if err != nil {
				return err
			}
			link.User = url.User(account.Id)
			if len(account.Flow) > 0 {
				link.RawQuery = addQuery(link.RawQuery, "flow", account.Flow)
			}
		}
	}
	return nil
}

func socksLink(proxy conf.OutboundDetourConfig, link *url.URL) error {
	var settings conf.SocksClientConfig
	err := json.Unmarshal(*proxy.Settings, &settings)
	if err != nil {
		return err
	}

	link.Fragment = getOutboundName(proxy)
	link.Scheme = "socks"

	if len(settings.Servers) > 0 {
		server := settings.Servers[0]
		link.Host = fmt.Sprintf("%s:%d", server.Address, server.Port)
		if len(server.Users) == 0 {
			username := base64.StdEncoding.EncodeToString([]byte(":"))
			link.User = url.User(username)
		} else {
			user := server.Users[0]
			var account conf.SocksAccount
			err := json.Unmarshal(user, &account)
			if err != nil {
				return err
			}
			password := fmt.Sprintf("%s:%s", account.Username, account.Password)
			username := base64.StdEncoding.EncodeToString([]byte(password))
			link.User = url.User(username)
		}
	}
	return nil
}

func trojanLink(proxy conf.OutboundDetourConfig, link *url.URL) error {
	var settings conf.TrojanClientConfig
	err := json.Unmarshal(*proxy.Settings, &settings)
	if err != nil {
		return err
	}

	link.Fragment = getOutboundName(proxy)
	link.Scheme = "trojan"

	if len(settings.Servers) > 0 {
		server := settings.Servers[0]
		link.Host = fmt.Sprintf("%s:%d", server.Address, server.Port)
		link.User = url.User(server.Password)
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
		// https://github.com/XTLS/Xray-core/discussions/716
		// 4.4.3 allowInsecure
		// 没有这个字段。不安全的节点，不适合分享。
		// I don't like this field, but too many people ask for it.
		allowInsecure := streamSettings.TLSSettings.Insecure
		if allowInsecure {
			query = addQuery(query, "allowInsecure", "1")
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
		pbk := streamSettings.REALITYSettings.PublicKey
		if len(pbk) > 0 {
			query = addQuery(query, "pbk", pbk)
		}
		sid := streamSettings.REALITYSettings.ShortId
		if len(sid) > 0 {
			query = addQuery(query, "sid", sid)
		}
		spx := streamSettings.REALITYSettings.SpiderX
		if len(spx) > 0 {
			query = addQuery(query, "spx", spx)
		}
	}

	link.RawQuery = query
}

func addQuery(query string, key string, value string) string {
	newQuery := fmt.Sprintf("%s=%s", key, url.QueryEscape(value))
	if len(query) == 0 {
		return newQuery
	} else {
		return fmt.Sprintf("%s&%s", query, newQuery)
	}
}
