package libxray

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"net/url"

	"github.com/xtls/xray-core/common/platform/filesystem"
)

type xrayJson struct {
	Outbounds []xrayOutbound `json:"outbounds,omitempty"`
}

type xrayOutbound struct {
	Name           string              `json:"name,omitempty"`
	Protocol       string              `json:"protocol,omitempty"`
	Settings       *json.RawMessage    `json:"settings,omitempty"`
	StreamSettings *xrayStreamSettings `json:"streamSettings,omitempty"`
}

type xrayShadowsocks struct {
	Servers []xrayShadowsocksServer `json:"servers,omitempty"`
}

type xrayShadowsocksServer struct {
	Address  string `json:"address,omitempty"`
	Port     int    `json:"port,omitempty"`
	Method   string `json:"method,omitempty"`
	Password string `json:"password,omitempty"`
}

type xraySocks struct {
	Servers []xraySocksServer `json:"servers,omitempty"`
}

type xraySocksServer struct {
	Address string                `json:"address,omitempty"`
	Port    int                   `json:"port,omitempty"`
	Users   []xraySocksServerUser `json:"users,omitempty"`
}

type xraySocksServerUser struct {
	User string `json:"user,omitempty"`
	Pass string `json:"pass,omitempty"`
}

type xrayTrojan struct {
	Servers []xrayTrojanServer `json:"servers,omitempty"`
}

type xrayTrojanServer struct {
	Address  string `json:"address,omitempty"`
	Port     int    `json:"port,omitempty"`
	Password string `json:"password,omitempty"`
}

type xrayVLESS struct {
	Vnext []xrayVLESSVnext `json:"vnext,omitempty"`
}

type xrayVLESSVnext struct {
	Address string               `json:"address,omitempty"`
	Port    int                  `json:"port,omitempty"`
	Users   []xrayVLESSVnextUser `json:"users,omitempty"`
}

type xrayVLESSVnextUser struct {
	Id   string `json:"id,omitempty"`
	Flow string `json:"flow,omitempty"`
}

type xrayVMess struct {
	Vnext []xrayVMessVnext `json:"vnext,omitempty"`
}

type xrayVMessVnext struct {
	Address string               `json:"address,omitempty"`
	Port    int                  `json:"port,omitempty"`
	Users   []xrayVMessVnextUser `json:"users,omitempty"`
}

type xrayVMessVnextUser struct {
	Id       string `json:"id,omitempty"`
	Security string `json:"security,omitempty"`
}

type xrayStreamSettings struct {
	Network         string               `json:"network,omitempty"`
	Security        string               `json:"security,omitempty"`
	TlsSettings     *xrayTlsSettings     `json:"tlsSettings,omitempty"`
	RealitySettings *xrayRealitySettings `json:"realitySettings,omitempty"`
	TcpSettings     *xrayTcpSettings     `json:"tcpSettings,omitempty"`
	KcpSettings     *xrayKcpSettings     `json:"kcpSettings,omitempty"`
	WsSettings      *xrayWsSettings      `json:"wsSettings,omitempty"`
	HttpSettings    *xrayHttpSettings    `json:"httpSettings,omitempty"`
	QuicSettings    *xrayQuicSettings    `json:"quicSettings,omitempty"`
	GrpcSettings    *xrayGrpcSettings    `json:"grpcSettings,omitempty"`
}

type xrayTlsSettings struct {
	ServerName    string   `json:"serverName,omitempty"`
	AllowInsecure bool     `json:"allowInsecure,omitempty"`
	Alpn          []string `json:"alpn,omitempty"`
	Fingerprint   string   `json:"fingerprint,omitempty"`
}

type xrayRealitySettings struct {
	Fingerprint string `json:"fingerprint,omitempty"`
	ServerName  string `json:"serverName,omitempty"`
	PublicKey   string `json:"publicKey,omitempty"`
	ShortId     string `json:"shortId,omitempty"`
	SpiderX     string `json:"spiderX,omitempty"`
}

type xrayTcpSettings struct {
	Header *xrayTcpSettingsHeader `json:"header,omitempty"`
}

type xrayTcpSettingsHeader struct {
	Type    string                        `json:"type,omitempty"`
	Request *xrayTcpSettingsHeaderRequest `json:"request,omitempty"`
}

type xrayTcpSettingsHeaderRequest struct {
	Path    []string                             `json:"path,omitempty"`
	Headers *xrayTcpSettingsHeaderRequestHeaders `json:"headers,omitempty"`
}

type xrayTcpSettingsHeaderRequestHeaders struct {
	Host []string `json:"Host,omitempty"`
}

type xrayFakeHeader struct {
	Type string `json:"type,omitempty"`
}

type xrayKcpSettings struct {
	Header *xrayFakeHeader `json:"header,omitempty"`
	Seed   string          `json:"seed,omitempty"`
}

type xrayWsSettings struct {
	Path    string                 `json:"path,omitempty"`
	Headers *xrayWsSettingsHeaders `json:"headers,omitempty"`
}

type xrayWsSettingsHeaders struct {
	Host string `json:"Host,omitempty"`
}

type xrayHttpSettings struct {
	Host []string `json:"host,omitempty"`
	Path string   `json:"path,omitempty"`
}

type xrayQuicSettings struct {
	Security string          `json:"security,omitempty"`
	Key      string          `json:"key,omitempty"`
	Header   *xrayFakeHeader `json:"header,omitempty"`
}

type xrayGrpcSettings struct {
	ServiceName string `json:"serviceName,omitempty"`
	MultiMode   bool   `json:"multiMode,omitempty"`
}

func ConvertXrayJsonToShareText(xrayPath string, textPath string) string {
	xrayBytes, err := filesystem.ReadFile(xrayPath)
	if err != nil {
		return err.Error()
	}

	var xray xrayJson

	err = json.Unmarshal(xrayBytes, &xray)
	if err != nil {
		return err.Error()
	}

	outbounds := xray.flattenOutbounds()
	if len(outbounds) == 0 {
		return "no valid outbounds"
	}

	var links []string
	for _, outbound := range outbounds {
		link, err := outbound.shareLink()
		if err == nil {
			links = append(links, link.String())
		}
	}
	if len(links) == 0 {
		return "no valid outbounds"
	}
	text := strings.Join(links, "\n")
	err = writeText(text, textPath)
	if err != nil {
		return err.Error()
	}

	return ""
}

func (xray xrayJson) flattenOutbounds() []xrayOutbound {
	var outbounds []xrayOutbound
	for _, proxy := range xray.Outbounds {
		outbounds = append(outbounds, proxy.flattenOutbounds()...)
	}
	return outbounds
}

func (proxy xrayOutbound) flattenOutbounds() []xrayOutbound {
	switch proxy.Protocol {
	case "shadowsocks":
		return proxy.shadowsocksOutbounds()
	case "vmess":
		return proxy.vmessOutbounds()
	case "vless":
		return proxy.vlessOutbounds()
	case "socks":
		return proxy.socksOutbounds()
	case "trojan":
		return proxy.trojanOutbounds()
	}
	return []xrayOutbound{}
}

func (proxy xrayOutbound) shadowsocksOutbounds() []xrayOutbound {
	var outbounds []xrayOutbound

	var settings xrayShadowsocks
	err := json.Unmarshal(*proxy.Settings, &settings)
	if err != nil {
		return outbounds
	}

	for _, server := range settings.Servers {
		var newSettings xrayShadowsocks
		newSettings.Servers = []xrayShadowsocksServer{server}
		setttingsBytes, err := json.Marshal(newSettings)
		if err == nil {
			var outbound xrayOutbound
			outbound.Protocol = proxy.Protocol
			outbound.Name = proxy.Name
			outbound.Settings = (*json.RawMessage)(&setttingsBytes)
			outbound.StreamSettings = proxy.StreamSettings

			outbounds = append(outbounds, outbound)
		}
	}
	return outbounds
}

func (proxy xrayOutbound) vmessOutbounds() []xrayOutbound {
	var outbounds []xrayOutbound

	var settings xrayVMess
	err := json.Unmarshal(*proxy.Settings, &settings)
	if err != nil {
		return outbounds
	}

	for _, vnext := range settings.Vnext {
		for _, user := range vnext.Users {
			var newVnext xrayVMessVnext
			newVnext.Address = vnext.Address
			newVnext.Port = vnext.Port
			newVnext.Users = []xrayVMessVnextUser{user}

			var newSettings xrayVMess
			newSettings.Vnext = []xrayVMessVnext{newVnext}
			setttingsBytes, err := json.Marshal(newSettings)
			if err == nil {
				var outbound xrayOutbound
				outbound.Protocol = proxy.Protocol
				outbound.Name = proxy.Name
				outbound.Settings = (*json.RawMessage)(&setttingsBytes)
				outbound.StreamSettings = proxy.StreamSettings

				outbounds = append(outbounds, outbound)
			}

		}
	}
	return outbounds
}

func (proxy xrayOutbound) vlessOutbounds() []xrayOutbound {
	var outbounds []xrayOutbound

	var settings xrayVLESS
	err := json.Unmarshal(*proxy.Settings, &settings)
	if err != nil {
		return outbounds
	}

	for _, vnext := range settings.Vnext {
		for _, user := range vnext.Users {
			var newVnext xrayVLESSVnext
			newVnext.Address = vnext.Address
			newVnext.Port = vnext.Port
			newVnext.Users = []xrayVLESSVnextUser{user}

			var newSettings xrayVLESS
			newSettings.Vnext = []xrayVLESSVnext{newVnext}
			setttingsBytes, err := json.Marshal(newSettings)
			if err == nil {
				var outbound xrayOutbound
				outbound.Protocol = proxy.Protocol
				outbound.Name = proxy.Name
				outbound.Settings = (*json.RawMessage)(&setttingsBytes)
				outbound.StreamSettings = proxy.StreamSettings

				outbounds = append(outbounds, outbound)
			}
		}
	}
	return outbounds
}

func (proxy xrayOutbound) socksOutbounds() []xrayOutbound {
	var outbounds []xrayOutbound

	var settings xraySocks
	err := json.Unmarshal(*proxy.Settings, &settings)
	if err != nil {
		return outbounds
	}

	for _, server := range settings.Servers {
		if len(server.Users) == 0 {
			var newServer xraySocksServer
			newServer.Address = server.Address
			newServer.Port = server.Port

			var newSettings xraySocks
			newSettings.Servers = []xraySocksServer{newServer}
			setttingsBytes, err := json.Marshal(newSettings)
			if err == nil {
				var outbound xrayOutbound
				outbound.Protocol = proxy.Protocol
				outbound.Name = proxy.Name
				outbound.Settings = (*json.RawMessage)(&setttingsBytes)
				outbound.StreamSettings = proxy.StreamSettings

				outbounds = append(outbounds, outbound)
			}
		} else {
			for _, user := range server.Users {
				var newServer xraySocksServer
				newServer.Address = server.Address
				newServer.Port = server.Port
				newServer.Users = []xraySocksServerUser{user}

				var newSettings xraySocks
				newSettings.Servers = []xraySocksServer{newServer}
				setttingsBytes, err := json.Marshal(newSettings)
				if err == nil {
					var outbound xrayOutbound
					outbound.Protocol = proxy.Protocol
					outbound.Name = proxy.Name
					outbound.Settings = (*json.RawMessage)(&setttingsBytes)
					outbound.StreamSettings = proxy.StreamSettings

					outbounds = append(outbounds, outbound)
				}
			}
		}

	}
	return outbounds
}

func (proxy xrayOutbound) trojanOutbounds() []xrayOutbound {
	var outbounds []xrayOutbound

	var settings xrayTrojan
	err := json.Unmarshal(*proxy.Settings, &settings)
	if err != nil {
		return outbounds
	}

	for _, server := range settings.Servers {
		var newSettings xrayTrojan
		newSettings.Servers = []xrayTrojanServer{server}
		setttingsBytes, err := json.Marshal(newSettings)
		if err == nil {
			var outbound xrayOutbound
			outbound.Protocol = proxy.Protocol
			outbound.Name = proxy.Name
			outbound.Settings = (*json.RawMessage)(&setttingsBytes)
			outbound.StreamSettings = proxy.StreamSettings

			outbounds = append(outbounds, outbound)
		}
	}
	return outbounds
}

func (proxy xrayOutbound) shareLink() (*url.URL, error) {
	var shareUrl url.URL

	switch proxy.Protocol {
	case "shadowsocks":
		err := proxy.shadowsocksLink(&shareUrl)
		if err != nil {
			return nil, err
		}
	case "vmess":
		err := proxy.vmessLink(&shareUrl)
		if err != nil {
			return nil, err
		}
	case "vless":
		err := proxy.vlessLink(&shareUrl)
		if err != nil {
			return nil, err
		}
	case "socks":
		err := proxy.socksLink(&shareUrl)
		if err != nil {
			return nil, err
		}
	case "trojan":
		err := proxy.trojanLink(&shareUrl)
		if err != nil {
			return nil, err
		}
	}
	proxy.streamSettingsLink(&shareUrl)

	return &shareUrl, nil
}

func (proxy xrayOutbound) shadowsocksLink(link *url.URL) error {
	var settings xrayShadowsocks
	err := json.Unmarshal(*proxy.Settings, &settings)
	if err != nil {
		return err
	}

	link.Fragment = proxy.Name
	link.Scheme = "ss"

	for _, server := range settings.Servers {
		link.Host = fmt.Sprintf("%s:%d", server.Address, server.Port)
		password := fmt.Sprintf("%s:%s", server.Password, server.Method)
		username := base64.StdEncoding.EncodeToString([]byte(password))
		link.User = url.User(username)
	}
	return nil
}

func (proxy xrayOutbound) vmessLink(link *url.URL) error {
	var settings xrayVMess
	err := json.Unmarshal(*proxy.Settings, &settings)
	if err != nil {
		return err
	}

	link.Fragment = proxy.Name
	link.Scheme = "vmess"

	for _, vnext := range settings.Vnext {
		link.Host = fmt.Sprintf("%s:%d", vnext.Address, vnext.Port)
		for _, user := range vnext.Users {
			link.User = url.User(user.Id)
			link.RawQuery = addQuery(link.RawQuery, "encryption", user.Security)
		}
	}
	return nil
}

func (proxy xrayOutbound) vlessLink(link *url.URL) error {
	var settings xrayVLESS
	err := json.Unmarshal(*proxy.Settings, &settings)
	if err != nil {
		return err
	}

	link.Fragment = proxy.Name
	link.Scheme = "vless"

	for _, vnext := range settings.Vnext {
		link.Host = fmt.Sprintf("%s:%d", vnext.Address, vnext.Port)
		for _, user := range vnext.Users {
			link.User = url.User(user.Id)
			if len(user.Flow) > 0 {
				link.RawQuery = addQuery(link.RawQuery, "flow", user.Flow)
			}
		}
	}
	return nil
}

func (proxy xrayOutbound) socksLink(link *url.URL) error {
	var settings xraySocks
	err := json.Unmarshal(*proxy.Settings, &settings)
	if err != nil {
		return err
	}

	link.Fragment = proxy.Name
	link.Scheme = "socks"

	for _, server := range settings.Servers {
		link.Host = fmt.Sprintf("%s:%d", server.Address, server.Port)
		if len(server.Users) == 0 {
			username := base64.StdEncoding.EncodeToString([]byte(":"))
			link.User = url.User(username)
		} else {
			for _, user := range server.Users {
				password := fmt.Sprintf("%s:%s", user.User, user.Pass)
				username := base64.StdEncoding.EncodeToString([]byte(password))
				link.User = url.User(username)
			}
		}
	}
	return nil
}

func (proxy xrayOutbound) trojanLink(link *url.URL) error {
	var settings xrayTrojan
	err := json.Unmarshal(*proxy.Settings, &settings)
	if err != nil {
		return err
	}

	for _, server := range settings.Servers {
		link.Host = fmt.Sprintf("%s:%d", server.Address, server.Port)
		link.User = url.User(server.Password)
	}
	return nil
}

func (proxy xrayOutbound) streamSettingsLink(link *url.URL) {
	streamSettings := proxy.StreamSettings
	if streamSettings == nil {
		return
	}
	query := link.RawQuery

	if len(streamSettings.Network) == 0 {
		streamSettings.Network = "tcp"
	}
	query = addQuery(query, "type", streamSettings.Network)

	if len(streamSettings.Security) == 0 {
		streamSettings.Security = "none"
	}
	query = addQuery(query, "security", streamSettings.Security)

	switch streamSettings.Network {
	case "tcp":
		headerType := streamSettings.TcpSettings.Header.Type
		if len(headerType) > 0 {
			query = addQuery(query, "headerType", headerType)
			host := streamSettings.TcpSettings.Header.Request.Headers.Host
			if len(host) > 0 {
				query = addQuery(query, "host", strings.Join(host, ","))
			}
			path := streamSettings.TcpSettings.Header.Request.Path
			if len(path) > 0 {
				query = addQuery(query, "path", strings.Join(path, ","))
			}
		}
	case "kcp":
		headerType := streamSettings.KcpSettings.Header.Type
		if len(headerType) > 0 {
			query = addQuery(query, "headerType", headerType)
		}
		seed := streamSettings.KcpSettings.Seed
		if len(headerType) > 0 {
			query = addQuery(query, "seed", seed)
		}
	case "ws":
		host := streamSettings.WsSettings.Headers.Host
		if len(host) > 0 {
			query = addQuery(query, "host", host)
		}
		path := streamSettings.WsSettings.Path
		if len(path) > 0 {
			query = addQuery(query, "path", path)
		}
	case "grpc":
		mode := streamSettings.GrpcSettings.MultiMode
		if mode {
			query = addQuery(query, "mode", "multi")
		} else {
			query = addQuery(query, "mode", "gun")
		}
		serviceName := streamSettings.GrpcSettings.ServiceName
		if len(serviceName) > 0 {
			query = addQuery(query, "serviceName", serviceName)
		}
	case "quic":
		headerType := streamSettings.KcpSettings.Header.Type
		if len(headerType) > 0 {
			query = addQuery(query, "headerType", headerType)
		}
		quicSecurity := streamSettings.QuicSettings.Security
		if len(quicSecurity) > 0 {
			query = addQuery(query, "quicSecurity", quicSecurity)
		}
		key := streamSettings.QuicSettings.Key
		if len(key) > 0 {
			query = addQuery(query, "key", key)
		}
	case "http":
		host := streamSettings.HttpSettings.Host
		if len(host) > 0 {
			query = addQuery(query, "host", strings.Join(host, ","))
		}
		path := streamSettings.HttpSettings.Path
		if len(path) > 0 {
			query = addQuery(query, "path", path)
		}
	}

	switch streamSettings.Security {
	case "tls":
		fp := streamSettings.TlsSettings.Fingerprint
		if len(fp) > 0 {
			query = addQuery(query, "fp", fp)
		}
		sni := streamSettings.TlsSettings.ServerName
		if len(sni) > 0 {
			query = addQuery(query, "sni", sni)
		}
		alpn := streamSettings.TlsSettings.Alpn
		if len(alpn) > 0 {
			query = addQuery(query, "alpn", strings.Join(alpn, ","))
		}
	case "reality":
		fp := streamSettings.RealitySettings.Fingerprint
		if len(fp) > 0 {
			query = addQuery(query, "fp", fp)
		}
		sni := streamSettings.RealitySettings.ServerName
		if len(sni) > 0 {
			query = addQuery(query, "sni", sni)
		}
		pbk := streamSettings.RealitySettings.PublicKey
		if len(pbk) > 0 {
			query = addQuery(query, "pbk", pbk)
		}
		sid := streamSettings.RealitySettings.ShortId
		if len(sid) > 0 {
			query = addQuery(query, "sid", sid)
		}
		spx := streamSettings.RealitySettings.SpiderX
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
