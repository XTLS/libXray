package libxray

import (
	"encoding/json"
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
