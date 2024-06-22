package nodep

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"
)

type clashYaml struct {
	Proxies []clashProxy `yaml:"proxies,omitempty"`
}

type clashProxy struct {
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
	RealityOpts       *clashProxyRealityOpts `yaml:"reality-opts,omitempty"`

	Network    string                `yaml:"network,omitempty"`
	Plugin     string                `yaml:"plugin,omitempty"`
	PluginOpts *clashProxyPluginOpts `yaml:"plugin-opts,omitempty"`
	WsOpts     *clashProxyWsOpts     `yaml:"ws-opts,omitempty"`
	H2Opts     *clashProxyH2Opts     `yaml:"h2-opts,omitempty"`
	GrpcOpts   *clashProxyGrpcOpts   `yaml:"grpc-opts,omitempty"`
}

type clashProxyPluginOpts struct {
	Mode           string `yaml:"mode,omitempty"`
	Tls            bool   `yaml:"tls,omitempty"`
	Fingerprint    string `yaml:"fingerprint,omitempty"`
	SkipCertVerify bool   `yaml:"skip-cert-verify,omitempty"`
	Host           string `yaml:"host,omitempty"`
	Path           string `yaml:"path,omitempty"`
}

type clashProxyWsOpts struct {
	Path                string                   `yaml:"path,omitempty"`
	Headers             *clashProxyWsOptsHeaders `yaml:"headers,omitempty"`
	MaxEarlyData        int                      `yaml:"max-early-data,omitempty"`
	EarlyDataHeaderName string                   `yaml:"early-data-header-name,omitempty"`
}

type clashProxyWsOptsHeaders struct {
	Host string `yaml:"Host,omitempty"`
}

type clashProxyH2Opts struct {
	Host []string `yaml:"host,omitempty"`
	Path string   `yaml:"path,omitempty"`
}

type clashProxyGrpcOpts struct {
	GrpcServiceName string `yaml:"grpc-service-name,omitempty"`
}

type clashProxyRealityOpts struct {
	PublicKey string `yaml:"public-key,omitempty"`
	ShortId   string `yaml:"short-id,omitempty"`
}

func tryConvertClashYaml(text string) (*XrayJson, error) {
	clashBytes := []byte(text)
	clash := clashYaml{}

	err := yaml.Unmarshal(clashBytes, &clash)
	if err != nil {
		return nil, err
	}

	xray := clash.xrayConfig()
	return &xray, nil
}

func (clash clashYaml) xrayConfig() XrayJson {
	var xray XrayJson

	var outbounds []XrayOutbound
	for _, proxy := range clash.Proxies {
		if outbound, err := proxy.outbound(); err == nil {
			outbounds = append(outbounds, *outbound)
		} else {
			fmt.Println(err)
		}
	}
	xray.Outbounds = outbounds

	return xray
}

func (proxy clashProxy) outbound() (*XrayOutbound, error) {
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

func (proxy clashProxy) shadowsocksOutbound() (*XrayOutbound, error) {
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
			wsSettings.Host = proxy.PluginOpts.Host
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

func (proxy clashProxy) vmessOutbound() (*XrayOutbound, error) {
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

func (proxy clashProxy) vlessOutbound() (*XrayOutbound, error) {
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

func (proxy clashProxy) socksOutbound() (*XrayOutbound, error) {
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

func (proxy clashProxy) trojanOutbound() (*XrayOutbound, error) {
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

func (proxy clashProxy) streamSettings(outbound XrayOutbound) (*XrayStreamSettings, error) {
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
				wsSettings.Host = proxy.WsOpts.Headers.Host
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

func (proxy clashProxy) parseSecurity(streamSettings *XrayStreamSettings, outbound XrayOutbound) {
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
