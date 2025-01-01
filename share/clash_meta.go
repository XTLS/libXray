package share

import (
	"encoding/json"
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/xtls/xray-core/infra/conf"
	"github.com/xtls/xray-core/proxy/vless"
)

// https://github.com/MetaCubeX/mihomo/blob/Alpha/docs/config.yaml

type ClashYaml struct {
	Proxies []ClashProxy `yaml:"proxies,omitempty"`
}

type ClashProxy struct {
	Name     string `yaml:"name,omitempty"`
	Type     string `yaml:"type,omitempty"`
	Server   string `yaml:"server,omitempty"`
	Port     uint16 `yaml:"port,omitempty"`
	Uuid     string `yaml:"uuid,omitempty"`
	Cipher   string `yaml:"cipher,omitempty"`
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`

	Udp        bool `yaml:"udp,omitempty"`
	udpOverTcp bool `yaml:"udp-over-tcp,omitempty"`

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
	GrpcOpts   *ClashProxyGrpcOpts   `yaml:"grpc-opts,omitempty"`
	SsOpts     *ClashProxySsOpts     `yaml:"ss-opts,omitempty"`

	// the below are fields of hysteria2.
	// although xray doesn't support hysteria2,
	// but someone may need them.
	Ports        string `yaml:"ports,omitempty"`
	HopInterval  int    `yaml:"hop-interval,omitempty"`
	Up           string `yaml:"up,omitempty"`
	Down         string `yaml:"down,omitempty"`
	Obfs         string `yaml:"obfs,omitempty"`
	ObfsPassword string `yaml:"obfs-password,omitempty"`
}

type ClashProxyRealityOpts struct {
	PublicKey string `yaml:"public-key,omitempty"`
	ShortId   string `yaml:"short-id,omitempty"`
}

type ClashProxyPluginOpts struct {
	Mode           string `yaml:"mode,omitempty"`
	Tls            bool   `yaml:"tls,omitempty"`
	Fingerprint    string `yaml:"fingerprint,omitempty"`
	SkipCertVerify bool   `yaml:"skip-cert-verify,omitempty"`
	Host           string `yaml:"host,omitempty"`
	Path           string `yaml:"path,omitempty"`
	Mux            bool   `yaml:"mux,omitempty"`
}

type ClashProxyWsOpts struct {
	Path    string                   `yaml:"path,omitempty"`
	Headers *ClashProxyWsOptsHeaders `yaml:"headers,omitempty"`
}

type ClashProxyWsOptsHeaders struct {
	Host string `yaml:"Host,omitempty"`
}

type ClashProxyGrpcOpts struct {
	GrpcServiceName string `yaml:"grpc-service-name,omitempty"`
}

type ClashProxySsOpts struct {
	Enabled  bool   `yaml:"enabled,omitempty"`
	Method   string `yaml:"method,omitempty"`
	Password string `yaml:"password,omitempty"`
}

func tryToParseClashYaml(text string) (*conf.Config, error) {
	clashBytes := []byte(text)
	clash := ClashYaml{}

	err := yaml.Unmarshal(clashBytes, &clash)
	if err != nil {
		return nil, err
	}

	xray := clash.xrayConfig()
	return xray, nil
}

func (clash ClashYaml) xrayConfig() *conf.Config {
	xray := &conf.Config{}

	var outbounds []conf.OutboundDetourConfig
	for _, proxy := range clash.Proxies {
		if outbound, err := proxy.outbound(); err == nil {
			outbounds = append(outbounds, *outbound)
		} else {
			fmt.Println(err)
		}
	}
	xray.OutboundConfigs = outbounds

	return xray
}

func (proxy ClashProxy) outbound() (*conf.OutboundDetourConfig, error) {
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

func (proxy ClashProxy) shadowsocksOutbound() (*conf.OutboundDetourConfig, error) {
	outbound := &conf.OutboundDetourConfig{}
	outbound.Protocol = "shadowsocks"
	setOutboundName(outbound, proxy.Name)

	server := &conf.ShadowsocksServerTarget{}
	server.Address = parseAddress(proxy.Server)
	server.Port = proxy.Port
	server.Cipher = proxy.Cipher
	server.Password = proxy.Password

	var settings conf.ShadowsocksClientConfig
	settings.Servers = []*conf.ShadowsocksServerTarget{server}

	settingsRawMessage, err := convertJsonToRawMessage(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = &settingsRawMessage

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
		streamSetting := &conf.StreamConfig{}
		network := conf.TransportProtocol("websocket")
		streamSetting.Network = &network

		wsSettings := &conf.WebSocketConfig{}
		if len(proxy.PluginOpts.Host) > 0 {
			wsSettings.Host = proxy.PluginOpts.Host
		}
		if len(proxy.PluginOpts.Path) > 0 {
			wsSettings.Path = proxy.PluginOpts.Path
		}
		streamSetting.WSSettings = wsSettings

		if proxy.PluginOpts.Tls {
			tlsSettings := &conf.TLSConfig{}
			tlsSettings.Fingerprint = proxy.PluginOpts.Fingerprint
			tlsSettings.Insecure = proxy.PluginOpts.SkipCertVerify
			streamSetting.TLSSettings = tlsSettings
		}

		outbound.StreamSetting = streamSetting
	}
	return outbound, nil
}

func (proxy ClashProxy) vmessOutbound() (*conf.OutboundDetourConfig, error) {
	outbound := &conf.OutboundDetourConfig{}
	outbound.Protocol = "vmess"
	setOutboundName(outbound, proxy.Name)

	user := &conf.VMessAccount{}
	user.ID = proxy.Uuid
	user.Security = proxy.Cipher

	vnext := &conf.VMessOutboundTarget{}
	vnext.Address = parseAddress(proxy.Server)
	vnext.Port = proxy.Port

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

	streamSettings, err := proxy.streamSettings(*outbound)
	if err != nil {
		return nil, err
	}
	outbound.StreamSetting = streamSettings

	return outbound, nil
}

func (proxy ClashProxy) vlessOutbound() (*conf.OutboundDetourConfig, error) {
	outbound := &conf.OutboundDetourConfig{}
	outbound.Protocol = "vless"
	setOutboundName(outbound, proxy.Name)

	user := &vless.Account{}
	user.Id = proxy.Uuid
	user.Flow = proxy.Flow

	vnext := &conf.VLessOutboundVnext{}
	vnext.Address = parseAddress(proxy.Server)
	vnext.Port = proxy.Port

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

	streamSettings, err := proxy.streamSettings(*outbound)
	if err != nil {
		return nil, err
	}
	outbound.StreamSetting = streamSettings

	return outbound, nil
}

func (proxy ClashProxy) socksOutbound() (*conf.OutboundDetourConfig, error) {
	outbound := &conf.OutboundDetourConfig{}
	outbound.Protocol = "socks"
	setOutboundName(outbound, proxy.Name)

	user := &conf.SocksAccount{}
	user.Username = proxy.Username
	user.Password = proxy.Password

	server := &conf.SocksRemoteConfig{}
	server.Address = parseAddress(proxy.Server)
	server.Port = proxy.Port

	userRawMessage, err := convertJsonToRawMessage(user)
	if err != nil {
		return nil, err
	}
	server.Users = []json.RawMessage{userRawMessage}

	settings := &conf.SocksClientConfig{}
	settings.Servers = []*conf.SocksRemoteConfig{server}

	settingsRawMessage, err := convertJsonToRawMessage(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = &settingsRawMessage

	streamSettings, err := proxy.streamSettings(*outbound)
	if err != nil {
		return nil, err
	}
	outbound.StreamSetting = streamSettings

	return outbound, nil
}

func (proxy ClashProxy) trojanOutbound() (*conf.OutboundDetourConfig, error) {
	outbound := &conf.OutboundDetourConfig{}
	outbound.Protocol = "trojan"
	setOutboundName(outbound, proxy.Name)

	server := &conf.TrojanServerTarget{}
	server.Address = parseAddress(proxy.Server)
	server.Port = proxy.Port
	server.Password = proxy.Password

	settings := &conf.TrojanClientConfig{}
	settings.Servers = []*conf.TrojanServerTarget{server}

	settingsRawMessage, err := convertJsonToRawMessage(settings)
	if err != nil {
		return nil, err
	}
	outbound.Settings = &settingsRawMessage

	streamSettings, err := proxy.streamSettings(*outbound)
	if err != nil {
		return nil, err
	}
	outbound.StreamSetting = streamSettings

	return outbound, nil
}

func (proxy ClashProxy) streamSettings(outbound conf.OutboundDetourConfig) (*conf.StreamConfig, error) {
	streamSettings := &conf.StreamConfig{}
	network := proxy.Network
	if len(proxy.Network) == 0 {
		network = "raw"
	}
	transportProtocol := conf.TransportProtocol(network)
	streamSettings.Network = &transportProtocol

	switch network {
	case "ws":
		if proxy.WsOpts != nil {
			wsSettings := &conf.WebSocketConfig{}
			if proxy.WsOpts.Headers != nil {
				wsSettings.Host = proxy.WsOpts.Headers.Host
			}
			wsSettings.Path = proxy.WsOpts.Path
			streamSettings.WSSettings = wsSettings
		}
	case "grpc":
		if proxy.GrpcOpts != nil {
			grpcSettings := &conf.GRPCConfig{}
			grpcSettings.ServiceName = proxy.GrpcOpts.GrpcServiceName
			streamSettings.GRPCSettings = grpcSettings
		}
	}
	proxy.parseSecurity(streamSettings, outbound)
	return streamSettings, nil
}

func (proxy ClashProxy) parseSecurity(streamSettings *conf.StreamConfig, outbound conf.OutboundDetourConfig) {
	tlsSettings := &conf.TLSConfig{}
	realitySettings := &conf.REALITYConfig{}

	if proxy.Tls {
		streamSettings.Security = "tls"
	}
	if proxy.SkipCertVerify {
		tlsSettings.Insecure = true
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
		alpn := conf.StringList(proxy.Alpn)
		tlsSettings.ALPN = &alpn
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
		streamSettings.TLSSettings = tlsSettings
	case "reality":
		streamSettings.REALITYSettings = realitySettings
	}
}
