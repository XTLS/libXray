package nodep

import (
	"encoding/json"
)

type XrayJson struct {
	Outbounds []XrayOutbound `json:"outbounds,omitempty"`
}

// Keep same with Xray Config, but add name for sharing.
type XrayOutbound struct {
	Name           string              `json:"name,omitempty"`
	Protocol       string              `json:"protocol,omitempty"`
	Settings       *json.RawMessage    `json:"settings,omitempty"`
	StreamSettings *XrayStreamSettings `json:"streamSettings,omitempty"`
}

type XrayShadowsocks struct {
	Servers []XrayShadowsocksServer `json:"servers,omitempty"`
}

type XrayShadowsocksServer struct {
	Address  string `json:"address,omitempty"`
	Port     int    `json:"port,omitempty"`
	Method   string `json:"method,omitempty"`
	Password string `json:"password,omitempty"`
}

type XraySocks struct {
	Servers []XraySocksServer `json:"servers,omitempty"`
}

type XraySocksServer struct {
	Address string                `json:"address,omitempty"`
	Port    int                   `json:"port,omitempty"`
	Users   []XraySocksServerUser `json:"users,omitempty"`
}

type XraySocksServerUser struct {
	User string `json:"user,omitempty"`
	Pass string `json:"pass,omitempty"`
}

type XrayTrojan struct {
	Servers []XrayTrojanServer `json:"servers,omitempty"`
}

type XrayTrojanServer struct {
	Address  string `json:"address,omitempty"`
	Port     int    `json:"port,omitempty"`
	Password string `json:"password,omitempty"`
}

type XrayVLESS struct {
	Vnext []XrayVLESSVnext `json:"vnext,omitempty"`
}

type XrayVLESSVnext struct {
	Address string               `json:"address,omitempty"`
	Port    int                  `json:"port,omitempty"`
	Users   []XrayVLESSVnextUser `json:"users,omitempty"`
}

type XrayVLESSVnextUser struct {
	Id         string `json:"id,omitempty"`
	Flow       string `json:"flow,omitempty"`
	Encryption string `json:"encryption,omitempty"`
}

type XrayVMess struct {
	Vnext []XrayVMessVnext `json:"vnext,omitempty"`
}

type XrayVMessVnext struct {
	Address string               `json:"address,omitempty"`
	Port    int                  `json:"port,omitempty"`
	Users   []XrayVMessVnextUser `json:"users,omitempty"`
}

type XrayVMessVnextUser struct {
	Id       string `json:"id,omitempty"`
	Security string `json:"security,omitempty"`
}

type XrayStreamSettings struct {
	Network             string                   `json:"network,omitempty"`
	Security            string                   `json:"security,omitempty"`
	TlsSettings         *XrayTlsSettings         `json:"tlsSettings,omitempty"`
	RealitySettings     *XrayRealitySettings     `json:"realitySettings,omitempty"`
	TcpSettings         *XrayTcpSettings         `json:"tcpSettings,omitempty"`
	KcpSettings         *XrayKcpSettings         `json:"kcpSettings,omitempty"`
	WsSettings          *XrayWsSettings          `json:"wsSettings,omitempty"`
	HttpSettings        *XrayHttpSettings        `json:"httpSettings,omitempty"`
	GrpcSettings        *XrayGrpcSettings        `json:"grpcSettings,omitempty"`
	HttpupgradeSettings *XrayHttpupgradeSettings `json:"httpupgradeSettings,omitempty"`
	SplithttpSettings   *XraySplithttpSettings   `json:"splithttpSettings,omitempty"`
}

type XrayTlsSettings struct {
	ServerName    string   `json:"serverName,omitempty"`
	AllowInsecure bool     `json:"allowInsecure,omitempty"`
	Alpn          []string `json:"alpn,omitempty"`
	Fingerprint   string   `json:"fingerprint,omitempty"`
}

type XrayRealitySettings struct {
	Fingerprint string `json:"fingerprint,omitempty"`
	ServerName  string `json:"serverName,omitempty"`
	PublicKey   string `json:"publicKey,omitempty"`
	ShortId     string `json:"shortId,omitempty"`
	SpiderX     string `json:"spiderX,omitempty"`
}

type XrayTcpSettings struct {
	Header *XrayTcpSettingsHeader `json:"header,omitempty"`
}

type XrayTcpSettingsHeader struct {
	Type    string                        `json:"type,omitempty"`
	Request *XrayTcpSettingsHeaderRequest `json:"request,omitempty"`
}

type XrayTcpSettingsHeaderRequest struct {
	Path    []string                             `json:"path,omitempty"`
	Headers *XrayTcpSettingsHeaderRequestHeaders `json:"headers,omitempty"`
}

type XrayTcpSettingsHeaderRequestHeaders struct {
	Host []string `json:"Host,omitempty"`
}

type XrayFakeHeader struct {
	Type string `json:"type,omitempty"`
}

type XrayKcpSettings struct {
	Header *XrayFakeHeader `json:"header,omitempty"`
	Seed   string          `json:"seed,omitempty"`
}

type XrayWsSettings struct {
	Path string `json:"path,omitempty"`
	Host string `json:"host,omitempty"`
}

type XrayHttpSettings struct {
	Host []string `json:"host,omitempty"`
	Path string   `json:"path,omitempty"`
}

type XrayGrpcSettings struct {
	Authority   string `json:"authority,omitempty"`
	ServiceName string `json:"serviceName,omitempty"`
	MultiMode   bool   `json:"multiMode,omitempty"`
}

type XrayHttpupgradeSettings struct {
	Path string `json:"path,omitempty"`
	Host string `json:"host,omitempty"`
}

type XraySplithttpSettings struct {
	Path string `json:"path,omitempty"`
	Host string `json:"host,omitempty"`
}

func (xray XrayJson) FlattenOutbounds() []XrayOutbound {
	var outbounds []XrayOutbound
	for _, proxy := range xray.Outbounds {
		outbounds = append(outbounds, proxy.flattenOutbounds()...)
	}
	return outbounds
}

func (proxy XrayOutbound) flattenOutbounds() []XrayOutbound {
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
	return []XrayOutbound{}
}

func (proxy XrayOutbound) shadowsocksOutbounds() []XrayOutbound {
	var outbounds []XrayOutbound

	var settings XrayShadowsocks
	err := json.Unmarshal(*proxy.Settings, &settings)
	if err != nil {
		return outbounds
	}

	for _, server := range settings.Servers {
		var newSettings XrayShadowsocks
		newSettings.Servers = []XrayShadowsocksServer{server}
		setttingsBytes, err := json.Marshal(newSettings)
		if err == nil {
			var outbound XrayOutbound
			outbound.Protocol = proxy.Protocol
			outbound.Name = proxy.Name
			outbound.Settings = (*json.RawMessage)(&setttingsBytes)
			outbound.StreamSettings = proxy.StreamSettings

			outbounds = append(outbounds, outbound)
		}
	}
	return outbounds
}

func (proxy XrayOutbound) vmessOutbounds() []XrayOutbound {
	var outbounds []XrayOutbound

	var settings XrayVMess
	err := json.Unmarshal(*proxy.Settings, &settings)
	if err != nil {
		return outbounds
	}

	for _, vnext := range settings.Vnext {
		for _, user := range vnext.Users {
			var newVnext XrayVMessVnext
			newVnext.Address = vnext.Address
			newVnext.Port = vnext.Port
			newVnext.Users = []XrayVMessVnextUser{user}

			var newSettings XrayVMess
			newSettings.Vnext = []XrayVMessVnext{newVnext}
			setttingsBytes, err := json.Marshal(newSettings)
			if err == nil {
				var outbound XrayOutbound
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

func (proxy XrayOutbound) vlessOutbounds() []XrayOutbound {
	var outbounds []XrayOutbound

	var settings XrayVLESS
	err := json.Unmarshal(*proxy.Settings, &settings)
	if err != nil {
		return outbounds
	}

	for _, vnext := range settings.Vnext {
		for _, user := range vnext.Users {
			var newVnext XrayVLESSVnext
			newVnext.Address = vnext.Address
			newVnext.Port = vnext.Port
			newVnext.Users = []XrayVLESSVnextUser{user}

			var newSettings XrayVLESS
			newSettings.Vnext = []XrayVLESSVnext{newVnext}
			setttingsBytes, err := json.Marshal(newSettings)
			if err == nil {
				var outbound XrayOutbound
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

func (proxy XrayOutbound) socksOutbounds() []XrayOutbound {
	var outbounds []XrayOutbound

	var settings XraySocks
	err := json.Unmarshal(*proxy.Settings, &settings)
	if err != nil {
		return outbounds
	}

	for _, server := range settings.Servers {
		if len(server.Users) == 0 {
			var newServer XraySocksServer
			newServer.Address = server.Address
			newServer.Port = server.Port

			var newSettings XraySocks
			newSettings.Servers = []XraySocksServer{newServer}
			setttingsBytes, err := json.Marshal(newSettings)
			if err == nil {
				var outbound XrayOutbound
				outbound.Protocol = proxy.Protocol
				outbound.Name = proxy.Name
				outbound.Settings = (*json.RawMessage)(&setttingsBytes)
				outbound.StreamSettings = proxy.StreamSettings

				outbounds = append(outbounds, outbound)
			}
		} else {
			for _, user := range server.Users {
				var newServer XraySocksServer
				newServer.Address = server.Address
				newServer.Port = server.Port
				newServer.Users = []XraySocksServerUser{user}

				var newSettings XraySocks
				newSettings.Servers = []XraySocksServer{newServer}
				setttingsBytes, err := json.Marshal(newSettings)
				if err == nil {
					var outbound XrayOutbound
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

func (proxy XrayOutbound) trojanOutbounds() []XrayOutbound {
	var outbounds []XrayOutbound

	var settings XrayTrojan
	err := json.Unmarshal(*proxy.Settings, &settings)
	if err != nil {
		return outbounds
	}

	for _, server := range settings.Servers {
		var newSettings XrayTrojan
		newSettings.Servers = []XrayTrojanServer{server}
		setttingsBytes, err := json.Marshal(newSettings)
		if err == nil {
			var outbound XrayOutbound
			outbound.Protocol = proxy.Protocol
			outbound.Name = proxy.Name
			outbound.Settings = (*json.RawMessage)(&setttingsBytes)
			outbound.StreamSettings = proxy.StreamSettings

			outbounds = append(outbounds, outbound)
		}
	}
	return outbounds
}
