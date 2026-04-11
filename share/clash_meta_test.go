package share

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xtls/xray-core/infra/conf"
)

func clashHysteria2YAML(fields string) string {
	return fmt.Sprintf(`proxies:
  - name: test-hy2
    type: hysteria2
    server: host
    port: 443
    password: authpass
%s`, fields)
}

func parseClashHy2(t *testing.T, yaml string) *conf.OutboundDetourConfig {
	t.Helper()
	config, err := tryToParseClashYaml(yaml)
	require.NoError(t, err)
	require.Len(t, config.OutboundConfigs, 1)
	return &config.OutboundConfigs[0]
}

func parseClashYAML(t *testing.T, yaml string) *conf.Config {
	t.Helper()
	cfg, err := tryToParseClashYaml(yaml)
	require.NoError(t, err)
	return cfg
}

func TestClashHysteria2_WithBandwidthPortHoppingSalamander(t *testing.T) {
	yaml := clashHysteria2YAML(`    up: 100 mbps
    down: 200 mbps
    ports: "20000-40000"
    hop-interval: 30
    obfs: salamander
    obfs-password: secret
    sni: example.com`)

	outbound := parseClashHy2(t, yaml)
	assert.Equal(t, "hysteria", outbound.Protocol)

	var settings conf.HysteriaClientConfig
	require.NoError(t, json.Unmarshal(*outbound.Settings, &settings))
	assert.Equal(t, int32(2), settings.Version)

	ss := outbound.StreamSetting
	require.NotNil(t, ss)

	// HysteriaSettings has version + auth only
	require.NotNil(t, ss.HysteriaSettings)
	assert.Equal(t, int32(2), ss.HysteriaSettings.Version)
	assert.Equal(t, "authpass", ss.HysteriaSettings.Auth)

	// QuicParams
	require.NotNil(t, ss.FinalMask)
	require.NotNil(t, ss.FinalMask.QuicParams)
	qp := ss.FinalMask.QuicParams
	assert.Equal(t, "brutal", qp.Congestion)
	assert.Equal(t, conf.Bandwidth("100 mbps"), qp.BrutalUp)
	assert.Equal(t, conf.Bandwidth("200 mbps"), qp.BrutalDown)

	// UdpHop
	var portList string
	require.NoError(t, json.Unmarshal(qp.UdpHop.PortList, &portList))
	assert.Equal(t, "20000-40000", portList)
	require.NotNil(t, qp.UdpHop.Interval)
	assert.Equal(t, int32(30), qp.UdpHop.Interval.From)
	assert.Equal(t, int32(30), qp.UdpHop.Interval.To)

	// Salamander
	require.Len(t, ss.FinalMask.Udp, 1)
	assert.Equal(t, "salamander", ss.FinalMask.Udp[0].Type)
	var salamander conf.Salamander
	require.NoError(t, json.Unmarshal(*ss.FinalMask.Udp[0].Settings, &salamander))
	assert.Equal(t, "secret", salamander.Password)

	// TLS default
	assert.Equal(t, "tls", ss.Security)
	require.NotNil(t, ss.TLSSettings)
	assert.Equal(t, "example.com", ss.TLSSettings.ServerName)
}

func TestClashHysteria2_BandwidthOnly(t *testing.T) {
	yaml := clashHysteria2YAML(`    up: 50 mbps
    down: 100 mbps
    sni: example.com`)

	outbound := parseClashHy2(t, yaml)
	ss := outbound.StreamSetting
	require.NotNil(t, ss)

	// HysteriaSettings
	require.NotNil(t, ss.HysteriaSettings)
	assert.Equal(t, "authpass", ss.HysteriaSettings.Auth)

	// QuicParams with bandwidth, no port-hopping
	require.NotNil(t, ss.FinalMask)
	require.NotNil(t, ss.FinalMask.QuicParams)
	qp := ss.FinalMask.QuicParams
	assert.Equal(t, "brutal", qp.Congestion)
	assert.Equal(t, conf.Bandwidth("50 mbps"), qp.BrutalUp)
	assert.Equal(t, conf.Bandwidth("100 mbps"), qp.BrutalDown)

	// No Salamander
	assert.Empty(t, ss.FinalMask.Udp)

	// No UdpHop
	assert.Nil(t, qp.UdpHop.PortList)
}

func TestClashHysteria2_SalamanderOnly(t *testing.T) {
	yaml := clashHysteria2YAML(`    obfs: salamander
    obfs-password: mysecret
    sni: example.com`)

	outbound := parseClashHy2(t, yaml)
	ss := outbound.StreamSetting
	require.NotNil(t, ss)

	require.NotNil(t, ss.FinalMask)

	// No QuicParams
	assert.Nil(t, ss.FinalMask.QuicParams)

	// Salamander
	require.Len(t, ss.FinalMask.Udp, 1)
	assert.Equal(t, "salamander", ss.FinalMask.Udp[0].Type)
	var salamander conf.Salamander
	require.NoError(t, json.Unmarshal(*ss.FinalMask.Udp[0].Settings, &salamander))
	assert.Equal(t, "mysecret", salamander.Password)
}

func TestClashHysteria2_Minimal(t *testing.T) {
	yaml := clashHysteria2YAML(`    sni: example.com`)

	outbound := parseClashHy2(t, yaml)
	assert.Equal(t, "hysteria", outbound.Protocol)

	ss := outbound.StreamSetting
	require.NotNil(t, ss)

	// HysteriaSettings
	require.NotNil(t, ss.HysteriaSettings)
	assert.Equal(t, int32(2), ss.HysteriaSettings.Version)
	assert.Equal(t, "authpass", ss.HysteriaSettings.Auth)

	// No FinalMask
	assert.Nil(t, ss.FinalMask)

	// TLS default for hysteria
	assert.Equal(t, "tls", ss.Security)
	require.NotNil(t, ss.TLSSettings)
	assert.Equal(t, "example.com", ss.TLSSettings.ServerName)
}

func TestClashShadowsocks_Plain(t *testing.T) {
	yaml := `proxies:
  - name: ss-plain
    type: ss
    server: ss.host
    port: 8388
    cipher: aes-128-gcm
    password: secret
    udp: true
    udp-over-tcp: true`
	cfg := parseClashYAML(t, yaml)
	require.Len(t, cfg.OutboundConfigs, 1)
	ob := cfg.OutboundConfigs[0]
	assert.Equal(t, "shadowsocks", ob.Protocol)
	var s conf.ShadowsocksClientConfig
	require.NoError(t, json.Unmarshal(*ob.Settings, &s))
	assert.Equal(t, "aes-128-gcm", s.Cipher)
	assert.Equal(t, "secret", s.Password)
	assert.True(t, s.UoT)
	assert.Nil(t, ob.StreamSetting)
}

func TestClashShadowsocks_V2rayPluginWebsocketTLS(t *testing.T) {
	yaml := `proxies:
  - name: ss-ws
    type: ss
    server: ss.ws
    port: 443
    cipher: aes-256-gcm
    password: p
    plugin: v2ray-plugin
    plugin-opts:
      mode: websocket
      host: cloud.cdn
      path: /ws
      tls: true
      fingerprint: chrome`
	cfg := parseClashYAML(t, yaml)
	require.Len(t, cfg.OutboundConfigs, 1)
	ss := cfg.OutboundConfigs[0].StreamSetting
	require.NotNil(t, ss)
	require.NotNil(t, ss.WSSettings)
	assert.Equal(t, "/ws", ss.WSSettings.Path)
	assert.Equal(t, "cloud.cdn", ss.WSSettings.Host)
	require.NotNil(t, ss.TLSSettings)
	assert.Equal(t, "chrome", ss.TLSSettings.Fingerprint)
}

func TestClashVmess_WsTLS(t *testing.T) {
	yaml := fmt.Sprintf(`proxies:
  - name: vm-ws
    type: vmess
    server: vm.host
    port: 443
    uuid: %s
    cipher: auto
    network: ws
    tls: true
    servername: vm.host
    alpn:
      - h2
    fingerprint: firefox
    ws-opts:
      path: /vmws
      headers:
        Host: front.cdn`, testShareUUID)
	cfg := parseClashYAML(t, yaml)
	require.Len(t, cfg.OutboundConfigs, 1)
	ss := cfg.OutboundConfigs[0].StreamSetting
	require.NotNil(t, ss.WSSettings)
	assert.Equal(t, "/vmws", ss.WSSettings.Path)
	assert.Equal(t, "front.cdn", ss.WSSettings.Host)
	assert.Equal(t, "tls", ss.Security)
	require.NotNil(t, ss.TLSSettings)
	assert.Equal(t, "vm.host", ss.TLSSettings.ServerName)
	assert.Equal(t, "firefox", ss.TLSSettings.Fingerprint)
}

func TestClashVless_Grpc(t *testing.T) {
	yaml := fmt.Sprintf(`proxies:
  - name: vl-grpc
    type: vless
    server: g.host
    port: 443
    uuid: %s
    encryption: none
    network: grpc
    tls: true
    sni: g.host
    grpc-opts:
      grpc-service-name: MySvc`, testShareUUID)
	cfg := parseClashYAML(t, yaml)
	require.Len(t, cfg.OutboundConfigs, 1)
	gs := cfg.OutboundConfigs[0].StreamSetting.GRPCSettings
	require.NotNil(t, gs)
	assert.Equal(t, "MySvc", gs.ServiceName)
}

func TestClashVless_XhttpExtraDownloadAndRanges(t *testing.T) {
	yaml := fmt.Sprintf(`proxies:
  - name: vl-xh
    type: vless
    server: xh.host
    port: 443
    uuid: %s
    encryption: none
    network: xhttp
    tls: true
    sni: xh.host
    client-fingerprint: edge
    ech-opts:
      enable: true
      config: echcfg123
    skip-cert-verify: true
    xhttp-opts:
      path: /xh
      host: xh.h
      mode: stream-one
      no-grpc-header: true
      x-padding-bytes: 100-500
      sc-max-each-post-bytes: 1000-2000
      headers:
        X-Custom: z
      reuse-settings:
        max-connections: 1-8
        max-concurrency: 2-4
        c-max-reuse-times: 0-0
        h-max-request-times: 1-1
        h-max-reusable-secs: 30-60
      download-settings:
        server: dl.host
        port: 8443
        tls: true
        servername: dl.host
        alpn:
          - h3
        path: /dl
        host: dl.h
        fingerprint: ios
        reality-opts:
          public-key: AbCdEfGhIjKlMnOpQrStUvWxYz0123456789012
          short-id: "0123456789abcdef"`, testShareUUID)
	cfg := parseClashYAML(t, yaml)
	require.Len(t, cfg.OutboundConfigs, 1)
	ob := cfg.OutboundConfigs[0]
	ss := ob.StreamSetting
	require.NotNil(t, ss.XHTTPSettings)
	x := ss.XHTTPSettings
	assert.Equal(t, "/xh", x.Path)
	assert.Equal(t, "stream-one", x.Mode)
	require.NotNil(t, x.Extra)
	var extra conf.SplitHTTPConfig
	require.NoError(t, json.Unmarshal(x.Extra, &extra))
	assert.True(t, extra.NoGRPCHeader)
	assert.Equal(t, "z", extra.Headers["X-Custom"])
	require.NotNil(t, extra.DownloadSettings)
	dl := extra.DownloadSettings
	require.NotNil(t, dl.XHTTPSettings)
	assert.Equal(t, "/dl", dl.XHTTPSettings.Path)
	assert.Equal(t, "reality", dl.Security)
	require.NotNil(t, dl.REALITYSettings)
}

func TestClashSocks5_WithAuth(t *testing.T) {
	yaml := `proxies:
  - name: sk
    type: socks5
    server: 127.0.0.1
    port: 1080
    username: su
    password: sp`
	cfg := parseClashYAML(t, yaml)
	require.Len(t, cfg.OutboundConfigs, 1)
	var s conf.SocksClientConfig
	require.NoError(t, json.Unmarshal(*cfg.OutboundConfigs[0].Settings, &s))
	assert.Equal(t, "su", s.Username)
	assert.Equal(t, "sp", s.Password)
}

func TestClashTrojan_TlsFromType(t *testing.T) {
	yaml := `proxies:
  - name: tr
    type: trojan
    server: tr.host
    port: 443
    password: trpass
    sni: tr.host
    skip-cert-verify: true`
	cfg := parseClashYAML(t, yaml)
	require.Len(t, cfg.OutboundConfigs, 1)
	ss := cfg.OutboundConfigs[0].StreamSetting
	require.NotNil(t, ss)
	assert.Equal(t, "tls", ss.Security)
	require.NotNil(t, ss.TLSSettings)
	assert.True(t, ss.TLSSettings.AllowInsecure)
}

func TestClashVless_Reality(t *testing.T) {
	yaml := fmt.Sprintf(`proxies:
  - name: vl-reality
    type: vless
    server: r.host
    port: 443
    uuid: %s
    encryption: none
    network: tcp
    reality-opts:
      public-key: XYAbCdEfGhIjKlMnOpQrStUvWxYz0123456789AB
      short-id: abcdef01
    client-fingerprint: qq
    servername: r.host`, testShareUUID)
	cfg := parseClashYAML(t, yaml)
	ss := cfg.OutboundConfigs[0].StreamSetting
	assert.Equal(t, "reality", ss.Security)
	require.NotNil(t, ss.REALITYSettings)
	assert.Equal(t, "XYAbCdEfGhIjKlMnOpQrStUvWxYz0123456789AB", ss.REALITYSettings.PublicKey)
}
