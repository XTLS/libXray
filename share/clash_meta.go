package share

import (
	"fmt"

	"gopkg.in/yaml.v3"

	"github.com/xtls/xray-core/infra/conf"
)

// https://github.com/MetaCubeX/mihomo/blob/Alpha/docs/config.yaml

type ClashYaml struct {
	Proxies []ClashProxy `yaml:"proxies,omitempty"`
}

type ClashProxy struct {
	Name       string `yaml:"name,omitempty"`
	Type       string `yaml:"type,omitempty"`
	Server     string `yaml:"server,omitempty"`
	Port       uint16 `yaml:"port,omitempty"`
	Uuid       string `yaml:"uuid,omitempty"`
	Cipher     string `yaml:"cipher,omitempty"`
	Username   string `yaml:"username,omitempty"`
	Password   string `yaml:"password,omitempty"`
	Encryption string `yaml:"encryption,omitempty"`

	Ports        string `yaml:"ports,omitempty"`
	HopInterval  int32  `yaml:"hop-interval,omitempty"`
	Up           string `yaml:"up,omitempty"`
	Down         string `yaml:"down,omitempty"`
	Obfs         string `yaml:"obfs,omitempty"`
	ObfsPassword string `yaml:"obfs-password,omitempty"`

	Udp        bool `yaml:"udp,omitempty"`
	UdpOverTcp bool `yaml:"udp-over-tcp,omitempty"`

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
	EchOpts    *ClashProxyEchOpts    `yaml:"ech-opts,omitempty"`
	WsOpts     *ClashProxyWsOpts     `yaml:"ws-opts,omitempty"`
	GrpcOpts   *ClashProxyGrpcOpts   `yaml:"grpc-opts,omitempty"`
	XhttpOpts  *ClashProxyXhttpOpts  `yaml:"xhttp-opts,omitempty"`
}

type ClashProxyEchOpts struct {
	Enable bool   `yaml:"enable,omitempty"`
	Config string `yaml:"config,omitempty"`
}

type ClashProxyRealityOpts struct {
	PublicKey string `yaml:"public-key,omitempty"`
	ShortId   string `yaml:"short-id,omitempty"`
}

type ClashProxyPluginOpts struct {
	Mode           string             `yaml:"mode,omitempty"`
	Tls            bool               `yaml:"tls,omitempty"`
	Fingerprint    string             `yaml:"fingerprint,omitempty"`
	EchOpts        *ClashProxyEchOpts `yaml:"ech-opts,omitempty"`
	SkipCertVerify bool               `yaml:"skip-cert-verify,omitempty"`
	Host           string             `yaml:"host,omitempty"`
	Path           string             `yaml:"path,omitempty"`
	Mux            bool               `yaml:"mux,omitempty"`
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

type ClashProxyXhttpOpts struct {
	Path               string                               `yaml:"path,omitempty"`
	Host               string                               `yaml:"host,omitempty"`
	Mode               string                               `yaml:"mode,omitempty"`
	Headers            map[string]string                    `yaml:"headers,omitempty"`
	NoGrpcHeader       bool                                 `yaml:"no-grpc-header,omitempty"`
	XPaddingBytes      string                               `yaml:"x-padding-bytes,omitempty"`
	ScMaxEachPostBytes string                               `yaml:"sc-max-each-post-bytes,omitempty"`
	ReuseSettings      *ClashProxyXhttpOptsXMUX             `yaml:"reuse-settings,omitempty"`
	DownloadSettings   *ClashProxyXhttpOptsDownloadSettings `yaml:"download-settings,omitempty"`
}

type ClashProxyXhttpOptsXMUX struct {
	MaxConnections   string `yaml:"max-connections,omitempty"`
	MaxConcurrency   string `yaml:"max-concurrency,omitempty"`
	CMaxReuseTimes   string `yaml:"c-max-reuse-times,omitempty"`
	HMaxRequestTimes string `yaml:"h-max-request-times,omitempty"`
	HMaxReusableSecs string `yaml:"h-max-reusable-secs,omitempty"`
}

type ClashProxyXhttpOptsDownloadSettings struct {
	Path               string                   `yaml:"path,omitempty"`
	Host               string                   `yaml:"host,omitempty"`
	Mode               string                   `yaml:"mode,omitempty"`
	Headers            map[string]string        `yaml:"headers,omitempty"`
	NoGrpcHeader       bool                     `yaml:"no-grpc-header,omitempty"`
	XPaddingBytes      string                   `yaml:"x-padding-bytes,omitempty"`
	ScMaxEachPostBytes string                   `yaml:"sc-max-each-post-bytes,omitempty"`
	ReuseSettings      *ClashProxyXhttpOptsXMUX `yaml:"reuse-settings,omitempty"`
	// proxy part
	Server            string                 `yaml:"server,omitempty"`
	Port              uint16                 `yaml:"port,omitempty"`
	Tls               bool                   `yaml:"tls,omitempty"`
	Alpn              []string               `yaml:"alpn,omitempty"`
	EchOpts           *ClashProxyEchOpts     `yaml:"ech-opts,omitempty"`
	RealityOpts       *ClashProxyRealityOpts `yaml:"reality-opts,omitempty"`
	SkipCertVerify    bool                   `yaml:"skip-cert-verify,omitempty"`
	Fingerprint       string                 `yaml:"fingerprint,omitempty"`
	Servername        string                 `yaml:"servername,omitempty"`
	ClientFingerprint string                 `yaml:"client-fingerprint,omitempty"`
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
	case "hysteria2":
		outbound, err := proxy.hysteria2Outbound()
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

	settings := conf.ShadowsocksClientConfig{}

	settings.Address = parseAddress(proxy.Server)
	settings.Port = proxy.Port

	settings.Cipher = proxy.Cipher
	settings.Password = proxy.Password
	settings.UoT = proxy.UdpOverTcp

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

			if proxy.PluginOpts.EchOpts != nil {
				if proxy.PluginOpts.EchOpts.Enable {
					tlsSettings.ECHConfigList = proxy.PluginOpts.EchOpts.Config
				}
			}

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

	settings := conf.VMessOutboundConfig{}

	settings.Address = parseAddress(proxy.Server)
	settings.Port = proxy.Port

	settings.ID = proxy.Uuid
	settings.Security = proxy.Cipher

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

	settings := conf.VLessOutboundConfig{}

	settings.Address = parseAddress(proxy.Server)
	settings.Port = proxy.Port

	settings.Id = proxy.Uuid
	settings.Flow = proxy.Flow
	if len(proxy.Encryption) > 0 {
		settings.Encryption = proxy.Encryption
	}

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

	settings := conf.SocksClientConfig{}

	settings.Address = parseAddress(proxy.Server)
	settings.Port = proxy.Port

	settings.Username = proxy.Username
	settings.Password = proxy.Password

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

	settings := conf.TrojanClientConfig{}

	settings.Address = parseAddress(proxy.Server)
	settings.Port = proxy.Port

	settings.Password = proxy.Password

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

func (proxy ClashProxy) hysteria2Outbound() (*conf.OutboundDetourConfig, error) {
	outbound := &conf.OutboundDetourConfig{}
	outbound.Protocol = "hysteria"
	setOutboundName(outbound, proxy.Name)

	settings := conf.HysteriaClientConfig{}

	settings.Version = 2
	settings.Address = parseAddress(proxy.Server)
	settings.Port = proxy.Port

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

	// fix hysteria network
	if proxy.Type == "hysteria2" {
		network = "hysteria"
		transportProtocol = conf.TransportProtocol("hysteria")
		streamSettings.Network = &transportProtocol
	}

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
	case "xhttp":
		if proxy.XhttpOpts != nil {
			xhttpSettings, err := parseXHTTPOpts(*proxy.XhttpOpts)
			if err != nil {
				return nil, err
			}
			streamSettings.XHTTPSettings = xhttpSettings
		}
	case "hysteria":
		hysteriaSettings := &conf.HysteriaConfig{}
		hysteriaSettings.Version = 2
		hysteriaSettings.Auth = proxy.Password
		streamSettings.HysteriaSettings = hysteriaSettings

		// Build QuicParams from bandwidth and port-hopping params
		var quicParams *conf.QuicParamsConfig
		if len(proxy.Up) > 0 || len(proxy.Down) > 0 || len(proxy.Ports) > 0 {
			quicParams = &conf.QuicParamsConfig{}
			if len(proxy.Up) > 0 || len(proxy.Down) > 0 {
				quicParams.Congestion = "brutal"
			}
			if len(proxy.Up) > 0 {
				quicParams.BrutalUp = conf.Bandwidth(proxy.Up)
			}
			if len(proxy.Down) > 0 {
				quicParams.BrutalDown = conf.Bandwidth(proxy.Down)
			}
			if len(proxy.Ports) > 0 {
				udpHop := conf.UdpHop{}
				portListRawMessage, err := convertJsonToRawMessage(proxy.Ports)
				if err != nil {
					return nil, err
				}
				udpHop.PortList = portListRawMessage
				if proxy.HopInterval > 0 {
					udpHop.Interval = &conf.Int32Range{Left: proxy.HopInterval, Right: proxy.HopInterval, From: proxy.HopInterval, To: proxy.HopInterval}
				}
				quicParams.UdpHop = udpHop
			}
		}

		// Build Salamander UDP masks
		var udpMasks []conf.Mask
		if proxy.Obfs == "salamander" && len(proxy.ObfsPassword) > 0 {
			obfs := conf.Mask{}
			obfs.Type = "salamander"

			settings := &conf.Salamander{}
			settings.Password = proxy.ObfsPassword

			settingsRawMessage, err := convertJsonToRawMessage(settings)
			if err != nil {
				return nil, err
			}
			obfs.Settings = &settingsRawMessage
			udpMasks = []conf.Mask{obfs}
		}

		// Compose FinalMask from QuicParams + Salamander
		if quicParams != nil || len(udpMasks) > 0 {
			streamSettings.FinalMask = &conf.FinalMask{QuicParams: quicParams, Udp: udpMasks}
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

	if proxy.EchOpts != nil {
		if proxy.EchOpts.Enable {
			tlsSettings.ECHConfigList = proxy.EchOpts.Config
		}
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

	tlsSettings.AllowInsecure = proxy.SkipCertVerify

	if (outbound.Protocol == "trojan" || outbound.Protocol == "hysteria") && len(streamSettings.Security) == 0 {
		streamSettings.Security = "tls"
	}

	switch streamSettings.Security {
	case "tls":
		streamSettings.TLSSettings = tlsSettings
	case "reality":
		streamSettings.REALITYSettings = realitySettings
	}
}

func parseXHTTPOpts(xhttpOpts ClashProxyXhttpOpts) (*conf.SplitHTTPConfig, error) {
	xhttpSettings := &conf.SplitHTTPConfig{}
	xhttpSettings.Path = xhttpOpts.Path
	xhttpSettings.Host = xhttpOpts.Host
	xhttpSettings.Mode = xhttpOpts.Mode

	extra := &conf.SplitHTTPConfig{}
	if len(xhttpOpts.Headers) > 0 {
		extra.Headers = xhttpOpts.Headers
	}
	extra.NoGRPCHeader = xhttpOpts.NoGrpcHeader
	if len(xhttpOpts.XPaddingBytes) > 0 {
		int32Range, err := parseInt32RangeString(xhttpOpts.XPaddingBytes)
		if err != nil {
			return nil, err
		}
		extra.XPaddingBytes = *int32Range
	}
	if len(xhttpOpts.ScMaxEachPostBytes) > 0 {
		int32Range, err := parseInt32RangeString(xhttpOpts.ScMaxEachPostBytes)
		if err != nil {
			return nil, err
		}
		extra.ScMaxEachPostBytes = *int32Range
	}
	if xhttpOpts.ReuseSettings != nil {
		xmuxSettings, err := parseXHTTPXMUX(xhttpOpts.ReuseSettings)
		if err != nil {
			return nil, err
		}
		extra.Xmux = *xmuxSettings
	}
	if xhttpOpts.DownloadSettings != nil {
		downloadSettings, err := parseXHTTPDownloadSettings(xhttpOpts.DownloadSettings)
		if err != nil {
			return nil, err
		}
		extra.DownloadSettings = downloadSettings
	}

	extraRawMessage, err := convertJsonToRawMessage(extra)
	if err != nil {
		return nil, err
	}
	xhttpSettings.Extra = extraRawMessage
	return xhttpSettings, nil
}

func parseInt32RangeString(s string) (*conf.Int32Range, error) {
	left, right, err := conf.ParseRangeString(s)
	if err != nil {
		return nil, err
	}
	return &conf.Int32Range{
		Left:  int32(left),
		Right: int32(right),
		From:  int32(left),
		To:    int32(right),
	}, nil
}

func parseXHTTPXMUX(reuseSettings *ClashProxyXhttpOptsXMUX) (*conf.XmuxConfig, error) {
	xmuxSettings := &conf.XmuxConfig{}
	if len(reuseSettings.MaxConnections) > 0 {
		int32Range, err := parseInt32RangeString(reuseSettings.MaxConnections)
		if err != nil {
			return nil, err
		}
		xmuxSettings.MaxConnections = *int32Range
	}
	if len(reuseSettings.MaxConcurrency) > 0 {
		int32Range, err := parseInt32RangeString(reuseSettings.MaxConcurrency)
		if err != nil {
			return nil, err
		}
		xmuxSettings.MaxConcurrency = *int32Range
	}
	if len(reuseSettings.CMaxReuseTimes) > 0 {
		int32Range, err := parseInt32RangeString(reuseSettings.CMaxReuseTimes)
		if err != nil {
			return nil, err
		}
		xmuxSettings.CMaxReuseTimes = *int32Range
	}
	if len(reuseSettings.HMaxRequestTimes) > 0 {
		int32Range, err := parseInt32RangeString(reuseSettings.HMaxRequestTimes)
		if err != nil {
			return nil, err
		}
		xmuxSettings.HMaxRequestTimes = *int32Range
	}
	if len(reuseSettings.HMaxReusableSecs) > 0 {
		int32Range, err := parseInt32RangeString(reuseSettings.HMaxReusableSecs)
		if err != nil {
			return nil, err
		}
		xmuxSettings.HMaxReusableSecs = *int32Range
	}
	return xmuxSettings, nil
}

func parseXHTTPDownloadSettings(downloadSettings *ClashProxyXhttpOptsDownloadSettings) (*conf.StreamConfig, error) {
	streamSettings := &conf.StreamConfig{}

	streamSettings.Address = parseAddress(downloadSettings.Server)
	streamSettings.Port = downloadSettings.Port

	network := conf.TransportProtocol("xhttp")
	streamSettings.Network = &network

	xhttpSettings := &conf.SplitHTTPConfig{}
	xhttpSettings.Path = downloadSettings.Path
	xhttpSettings.Host = downloadSettings.Host
	xhttpSettings.Mode = downloadSettings.Mode

	if len(downloadSettings.Headers) > 0 {
		xhttpSettings.Headers = downloadSettings.Headers
	}
	xhttpSettings.NoGRPCHeader = downloadSettings.NoGrpcHeader
	if len(downloadSettings.XPaddingBytes) > 0 {
		int32Range, err := parseInt32RangeString(downloadSettings.XPaddingBytes)
		if err != nil {
			return nil, err
		}
		xhttpSettings.XPaddingBytes = *int32Range
	}
	if len(downloadSettings.ScMaxEachPostBytes) > 0 {
		int32Range, err := parseInt32RangeString(downloadSettings.ScMaxEachPostBytes)
		if err != nil {
			return nil, err
		}
		xhttpSettings.ScMaxEachPostBytes = *int32Range
	}
	if downloadSettings.ReuseSettings != nil {
		xmuxSettings, err := parseXHTTPXMUX(downloadSettings.ReuseSettings)
		if err != nil {
			return nil, err
		}
		xhttpSettings.Xmux = *xmuxSettings
	}
	streamSettings.XHTTPSettings = xhttpSettings

	parseXHTTPDownloadSettingsSecurity(streamSettings, downloadSettings)

	return streamSettings, nil
}

func parseXHTTPDownloadSettingsSecurity(streamSettings *conf.StreamConfig, proxy *ClashProxyXhttpOptsDownloadSettings) {
	tlsSettings := &conf.TLSConfig{}
	realitySettings := &conf.REALITYConfig{}

	if proxy.Tls {
		streamSettings.Security = "tls"
	}

	if proxy.EchOpts != nil {
		if proxy.EchOpts.Enable {
			tlsSettings.ECHConfigList = proxy.EchOpts.Config
		}
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

	tlsSettings.AllowInsecure = proxy.SkipCertVerify

	switch streamSettings.Security {
	case "tls":
		streamSettings.TLSSettings = tlsSettings
	case "reality":
		streamSettings.REALITYSettings = realitySettings
	}
}
