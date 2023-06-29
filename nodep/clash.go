package nodep

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

type ClashYaml struct {
	Proxies []ClashProxy `yaml:"proxies,omitempty"`
}

type ClashProxy struct {
	Name     string `yaml:"name,omitempty"`
	Type     string `yaml:"type,omitempty"`
	Server   string `yaml:"server,omitempty"`
	Port     int    `yaml:"port,omitempty"`
	Uuid     string `yaml:"uuid,omitempty"`
	Cipher   string `yaml:"cipher,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`

	Tls            bool     `yaml:"tls,omitempty"`
	SkipCertVerify bool     `yaml:"skip-cert-verify,omitempty"`
	Servername     string   `yaml:"servername,omitempty"`
	Sni            string   `yaml:"sni,omitempty"`
	Alpn           []string `yaml:"alpn,omitempty"`

	Fingerprint       string                 `yaml:"fingerprint,omitempty"`
	ClientFingerprint string                 `yaml:"client-fingerprint,omitempty"`
	Flow              string                 `yaml:"flow,omitempty"`
	RealityOpts       *ClashProxyRealityOpts `yaml:"reality-opts,omitempty"`

	Network    string                `yaml:"network,omitempty"`
	Plugin     string                `yaml:"plugin,omitempty"`
	PluginOpts *ClashProxyPluginOpts `yaml:"plugin-opts,omitempty"`
	WsOpts     *ClashProxyWsOpts     `yaml:"ws-opts,omitempty"`
	H2Opts     *ClashProxyH2Opts     `yaml:"h2-opts,omitempty"`
	GrpcOpts   *ClashProxyGrpcOpts   `yaml:"grpc-opts,omitempty"`
}

type ClashProxyPluginOpts struct {
	Mode           string `yaml:"mode,omitempty"`
	Tls            bool   `yaml:"tls,omitempty"`
	Fingerprint    string `yaml:"fingerprint,omitempty"`
	SkipCertVerify bool   `yaml:"skip-cert-verify,omitempty"`
	Host           string `yaml:"host,omitempty"`
	Path           string `yaml:"path,omitempty"`
}

type ClashProxyWsOpts struct {
	Path                string                   `yaml:"path,omitempty"`
	Headers             *ClashProxyWsOptsHeaders `yaml:"headers,omitempty"`
	MaxEarlyData        int                      `yaml:"max-early-data,omitempty"`
	EarlyDataHeaderName string                   `yaml:"early-data-header-name,omitempty"`
}

type ClashProxyWsOptsHeaders struct {
	Host string `yaml:"Host,omitempty"`
}

type ClashProxyH2Opts struct {
	Host []string `yaml:"host,omitempty"`
	Path string   `yaml:"path,omitempty"`
}

type ClashProxyGrpcOpts struct {
	GrpcServiceName string `yaml:"grpc-service-name,omitempty"`
}

type ClashProxyRealityOpts struct {
	PublicKey string `yaml:"public-key,omitempty"`
	ShortId   string `yaml:"short-id,omitempty"`
}

func tryConvertClashYaml(text string) (*XrayJson, error) {
	ClashBytes := []byte(text)
	Clash := ClashYaml{}

	err := yaml.Unmarshal(ClashBytes, &Clash)
	if err != nil {
		return nil, err
	}

	xray := Clash.xrayConfig()
	return &xray, nil
}

func (Clash ClashYaml) xrayConfig() XrayJson {
	var xray XrayJson

	var outbounds []XrayOutbound
	for _, proxy := range Clash.Proxies {
		if outbound, err := proxy.outbound(); err == nil {
			outbounds = append(outbounds, *outbound)
		} else {
			fmt.Println(err)
		}
	}
	xray.Outbounds = outbounds

	return xray
}

func (proxy ClashProxy) outbound() (*XrayOutbound, error) {
	switch proxy.Type {
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

	case "socks5":
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
	return nil, fmt.Errorf("unsupport proxy type: %s", proxy.Type)
}

func (proxy ClashProxy) shadowsocksOutbound() (*XrayOutbound, error) {
	var outbound XrayOutbound
	outbound.Protocol = "shadowsocks"
	outbound.Name = proxy.Name

	var server XrayShadowsocksServer
	server.Address = proxy.Server
	server.Port = proxy.Port
	server.Method = proxy.Cipher
	server.Password = proxy.Password

	var settings XrayShadowsocks
	settings.Servers = []XrayShadowsocksServer{server}

	setttingsBytes, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = (*json.RawMessage)(&setttingsBytes)

	if len(proxy.Plugin) != 0 {
		if proxy.Plugin != "v2ray-plugin" {
			return nil, fmt.Errorf("unsupport ss plugin: obfs")
		}
		if proxy.PluginOpts == nil {
			return nil, fmt.Errorf("unsupport ss plugin-opts: nil")
		}
		if proxy.PluginOpts.Mode != "websocket" {
			return nil, fmt.Errorf("unsupport ss plugin-opts mode: %s", proxy.PluginOpts.Mode)
		}
		var streamSetting XrayStreamSettings
		streamSetting.Network = "websocket"

		var wsSettings XrayWsSettings
		if len(proxy.PluginOpts.Path) > 0 {
			wsSettings.Path = proxy.PluginOpts.Path
		}
		if len(proxy.PluginOpts.Host) > 0 {
			var headers XrayWsSettingsHeaders
			headers.Host = proxy.PluginOpts.Host
			wsSettings.Headers = &headers
		}
		streamSetting.WsSettings = &wsSettings

		if proxy.PluginOpts.Tls {
			var tlsSettings XrayTlsSettings
			tlsSettings.Fingerprint = proxy.PluginOpts.Fingerprint
			tlsSettings.AllowInsecure = proxy.PluginOpts.SkipCertVerify
			streamSetting.TlsSettings = &tlsSettings
		}

		outbound.StreamSettings = &streamSetting
	}
	return &outbound, nil
}

func (proxy ClashProxy) vmessOutbound() (*XrayOutbound, error) {
	var outbound XrayOutbound
	outbound.Protocol = "vmess"
	outbound.Name = proxy.Name

	var user XrayVMessVnextUser
	user.Id = proxy.Uuid
	user.Security = proxy.Cipher

	var vnext XrayVMessVnext
	vnext.Address = proxy.Server
	vnext.Port = proxy.Port
	vnext.Users = []XrayVMessVnextUser{user}

	var settings XrayVMess
	settings.Vnext = []XrayVMessVnext{vnext}

	setttingsBytes, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = (*json.RawMessage)(&setttingsBytes)

	streamSettings, err := proxy.streamSettings(outbound)
	if err != nil {
		return nil, err
	}
	outbound.StreamSettings = streamSettings

	return &outbound, nil
}

func (proxy ClashProxy) vlessOutbound() (*XrayOutbound, error) {
	var outbound XrayOutbound
	outbound.Protocol = "vless"
	outbound.Name = proxy.Name

	var user XrayVLESSVnextUser
	user.Id = proxy.Uuid
	user.Flow = proxy.Flow

	var vnext XrayVLESSVnext
	vnext.Address = proxy.Server
	vnext.Port = proxy.Port
	vnext.Users = []XrayVLESSVnextUser{user}

	var settings XrayVLESS
	settings.Vnext = []XrayVLESSVnext{vnext}

	setttingsBytes, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = (*json.RawMessage)(&setttingsBytes)

	streamSettings, err := proxy.streamSettings(outbound)
	if err != nil {
		return nil, err
	}
	outbound.StreamSettings = streamSettings

	return &outbound, nil
}

func (proxy ClashProxy) socksOutbound() (*XrayOutbound, error) {
	var outbound XrayOutbound
	outbound.Protocol = "socks"
	outbound.Name = proxy.Name

	var user XraySocksServerUser
	user.User = proxy.Username
	user.Pass = proxy.Password

	var server XraySocksServer
	server.Address = proxy.Server
	server.Port = proxy.Port
	server.Users = []XraySocksServerUser{user}

	var settings XraySocks
	settings.Servers = []XraySocksServer{server}

	setttingsBytes, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = (*json.RawMessage)(&setttingsBytes)

	streamSettings, err := proxy.streamSettings(outbound)
	if err != nil {
		return nil, err
	}
	outbound.StreamSettings = streamSettings

	return &outbound, nil
}

func (proxy ClashProxy) trojanOutbound() (*XrayOutbound, error) {
	var outbound XrayOutbound
	outbound.Protocol = "trojan"
	outbound.Name = proxy.Name

	var server XrayTrojanServer
	server.Address = proxy.Server
	server.Port = proxy.Port
	server.Password = proxy.Password

	var settings XrayTrojan
	settings.Servers = []XrayTrojanServer{server}

	setttingsBytes, err := json.Marshal(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = (*json.RawMessage)(&setttingsBytes)

	streamSettings, err := proxy.streamSettings(outbound)
	if err != nil {
		return nil, err
	}
	outbound.StreamSettings = streamSettings

	return &outbound, nil
}

func (proxy ClashProxy) streamSettings(outbound XrayOutbound) (*XrayStreamSettings, error) {
	var streamSettings XrayStreamSettings
	if len(proxy.Network) == 0 {
		streamSettings.Network = "tcp"
	} else {
		streamSettings.Network = proxy.Network
	}

	switch streamSettings.Network {
	case "ws":
		if proxy.WsOpts != nil {
			var wsSettings XrayWsSettings
			if proxy.WsOpts.Headers != nil {
				var headers XrayWsSettingsHeaders
				headers.Host = proxy.WsOpts.Headers.Host
				wsSettings.Headers = &headers
			}

			wsSettings.Path = proxy.WsOpts.Path

			if proxy.WsOpts.MaxEarlyData > 0 {
				return nil, fmt.Errorf("unsupport ws-opts max-early-data: %d", proxy.WsOpts.MaxEarlyData)
			}
			streamSettings.WsSettings = &wsSettings
		}
	case "h2":
		if proxy.H2Opts != nil {
			var httpSettings XrayHttpSettings
			httpSettings.Host = proxy.H2Opts.Host
			httpSettings.Path = proxy.H2Opts.Path

			streamSettings.HttpSettings = &httpSettings
		}
	case "grpc":
		if proxy.GrpcOpts != nil {
			var grpcSettings XrayGrpcSettings
			grpcSettings.ServiceName = proxy.GrpcOpts.GrpcServiceName

			streamSettings.GrpcSettings = &grpcSettings
		}
	}
	proxy.parseSecurity(&streamSettings, outbound)
	return &streamSettings, nil
}

func (proxy ClashProxy) parseSecurity(streamSettings *XrayStreamSettings, outbound XrayOutbound) {
	var tlsSettings XrayTlsSettings
	var realitySettings XrayRealitySettings

	if proxy.Tls {
		streamSettings.Security = "tls"
	}
	if proxy.SkipCertVerify {
		tlsSettings.AllowInsecure = true
	}

	if proxy.RealityOpts != nil {
		streamSettings.Security = "reality"
		realitySettings.PublicKey = proxy.RealityOpts.PublicKey
		realitySettings.ShortId = proxy.RealityOpts.ShortId
	}
	if len(proxy.Servername) > 0 {
		tlsSettings.ServerName = proxy.Servername
		realitySettings.ServerName = proxy.Servername
	}
	if len(proxy.Sni) > 0 {
		tlsSettings.ServerName = proxy.Sni
		realitySettings.ServerName = proxy.Sni
	}
	if len(proxy.Alpn) > 0 {
		tlsSettings.Alpn = proxy.Alpn
	}
	if len(proxy.Fingerprint) > 0 {
		tlsSettings.Fingerprint = proxy.Fingerprint
		realitySettings.Fingerprint = proxy.Fingerprint
	}
	if len(proxy.ClientFingerprint) > 0 {
		tlsSettings.Fingerprint = proxy.ClientFingerprint
		realitySettings.Fingerprint = proxy.ClientFingerprint
	}

	if outbound.Protocol == "trojan" && len(streamSettings.Security) == 0 {
		streamSettings.Security = "tls"
	}

	switch streamSettings.Security {
	case "tls":
		streamSettings.TlsSettings = &tlsSettings
	case "reality":
		streamSettings.RealitySettings = &realitySettings
	}
}
