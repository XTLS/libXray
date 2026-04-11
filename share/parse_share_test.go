package share

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xtls/xray-core/infra/conf"
)

const testShareUUID = "12345678-abcd-abcd-abcd-123456789abc"

func ssUserB64(cipher, password string) string {
	return base64.StdEncoding.EncodeToString([]byte(cipher + ":" + password))
}

func parseHy2Link(t *testing.T, link string) *conf.OutboundDetourConfig {
	t.Helper()
	config, err := ConvertShareLinksToXrayJson(link)
	require.NoError(t, err)
	require.Len(t, config.OutboundConfigs, 1)
	return &config.OutboundConfigs[0]
}

func TestHysteria2_Minimal(t *testing.T) {
	outbound := parseHy2Link(t, "hy2://auth@host:443?sni=example.com")

	assert.Equal(t, "hysteria", outbound.Protocol)

	var settings conf.HysteriaClientConfig
	require.NoError(t, json.Unmarshal(*outbound.Settings, &settings))
	assert.Equal(t, int32(2), settings.Version)
	assert.Equal(t, uint16(443), settings.Port)

	ss := outbound.StreamSetting
	require.NotNil(t, ss)
	assert.Equal(t, "tls", ss.Security)
	require.NotNil(t, ss.TLSSettings)
	assert.Equal(t, "example.com", ss.TLSSettings.ServerName)

	require.NotNil(t, ss.HysteriaSettings)
	assert.Equal(t, int32(2), ss.HysteriaSettings.Version)
	assert.Equal(t, "auth", ss.HysteriaSettings.Auth)

	// No FinalMask for minimal link
	assert.Nil(t, ss.FinalMask)
}

func TestHysteria2_WithBandwidth(t *testing.T) {
	outbound := parseHy2Link(t, "hy2://auth@host:443?up=100+mbps&down=200+mbps&sni=example.com")

	ss := outbound.StreamSetting
	require.NotNil(t, ss)
	require.NotNil(t, ss.FinalMask)
	require.NotNil(t, ss.FinalMask.QuicParams)

	qp := ss.FinalMask.QuicParams
	assert.Equal(t, "brutal", qp.Congestion)
	assert.Equal(t, conf.Bandwidth("100 mbps"), qp.BrutalUp)
	assert.Equal(t, conf.Bandwidth("200 mbps"), qp.BrutalDown)

	// No Salamander
	assert.Empty(t, ss.FinalMask.Udp)
}

func TestHysteria2_WithSalamander(t *testing.T) {
	outbound := parseHy2Link(t, "hy2://auth@host:443?obfs=salamander&obfs-password=secret&sni=example.com")

	ss := outbound.StreamSetting
	require.NotNil(t, ss)
	require.NotNil(t, ss.FinalMask)

	// No QuicParams
	assert.Nil(t, ss.FinalMask.QuicParams)

	// Has Salamander
	require.Len(t, ss.FinalMask.Udp, 1)
	assert.Equal(t, "salamander", ss.FinalMask.Udp[0].Type)

	var salamander conf.Salamander
	require.NoError(t, json.Unmarshal(*ss.FinalMask.Udp[0].Settings, &salamander))
	assert.Equal(t, "secret", salamander.Password)
}

func TestHysteria2_WithEverything(t *testing.T) {
	outbound := parseHy2Link(t, "hy2://auth@host:443?up=50+mbps&down=100+mbps&obfs=salamander&obfs-password=secret&ports=20000-40000&hop-interval=30&sni=example.com")

	ss := outbound.StreamSetting
	require.NotNil(t, ss)
	require.NotNil(t, ss.FinalMask)

	// QuicParams with bandwidth and port-hopping
	require.NotNil(t, ss.FinalMask.QuicParams)
	qp := ss.FinalMask.QuicParams
	assert.Equal(t, "brutal", qp.Congestion)
	assert.Equal(t, conf.Bandwidth("50 mbps"), qp.BrutalUp)
	assert.Equal(t, conf.Bandwidth("100 mbps"), qp.BrutalDown)

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
}

func TestHysteria2_WithTLSParams(t *testing.T) {
	outbound := parseHy2Link(t, "hy2://auth@host:443?sni=example.com&alpn=h3&fp=chrome")

	ss := outbound.StreamSetting
	require.NotNil(t, ss)
	assert.Equal(t, "tls", ss.Security)
	require.NotNil(t, ss.TLSSettings)
	assert.Equal(t, "example.com", ss.TLSSettings.ServerName)
	assert.Equal(t, "chrome", ss.TLSSettings.Fingerprint)
	require.NotNil(t, ss.TLSSettings.ALPN)
	assert.Equal(t, conf.StringList{"h3"}, *ss.TLSSettings.ALPN)
}

func TestHysteria2_PortsOnlyNoCongestion(t *testing.T) {
	outbound := parseHy2Link(t, "hy2://auth@host:443?ports=20000-40000&hop-interval=10&sni=example.com")

	ss := outbound.StreamSetting
	require.NotNil(t, ss)
	require.NotNil(t, ss.FinalMask)
	require.NotNil(t, ss.FinalMask.QuicParams)

	qp := ss.FinalMask.QuicParams
	// No Congestion when only ports are set (no bandwidth)
	assert.Empty(t, qp.Congestion)
	assert.Empty(t, string(qp.BrutalUp))
	assert.Empty(t, string(qp.BrutalDown))

	// UdpHop is set
	var portList string
	require.NoError(t, json.Unmarshal(qp.UdpHop.PortList, &portList))
	assert.Equal(t, "20000-40000", portList)
	require.NotNil(t, qp.UdpHop.Interval)
	assert.Equal(t, int32(10), qp.UdpHop.Interval.From)
	assert.Equal(t, int32(10), qp.UdpHop.Interval.To)
}

func TestHysteria2_TLSDefaultWhenSecurityOmitted(t *testing.T) {
	// When no security= param is present, hysteria2 should default to TLS
	outbound := parseHy2Link(t, "hy2://auth@host:443?sni=example.com")

	ss := outbound.StreamSetting
	require.NotNil(t, ss)
	assert.Equal(t, "tls", ss.Security)
	require.NotNil(t, ss.TLSSettings)
}

func TestFixWindowsReturn(t *testing.T) {
	in := "a\r\nb\r\nc"
	assert.Equal(t, "a\nb\nc", FixWindowsReturn(in))
}

func TestConvertShareLinksToXrayJson_XrayJSONRoundTrip(t *testing.T) {
	orig, err := ConvertShareLinksToXrayJson(
		"vless://" + testShareUUID + "@example.com:443?encryption=none&security=tls&sni=example.com&type=ws&path=%2Fp&host=cdn.example.com#tag1",
	)
	require.NoError(t, err)
	require.Len(t, orig.OutboundConfigs, 1)

	raw, err := json.Marshal(orig)
	require.NoError(t, err)

	again, err := ConvertShareLinksToXrayJson(string(raw))
	require.NoError(t, err)
	require.Len(t, again.OutboundConfigs, 1)
	assert.Equal(t, orig.OutboundConfigs[0].Protocol, again.OutboundConfigs[0].Protocol)
}

func TestConvertShareLinksToXrayJson_XrayJSONInvalid(t *testing.T) {
	_, err := ConvertShareLinksToXrayJson("{not json")
	require.Error(t, err)
}

func TestConvertShareLinksToXrayJson_XrayJSONNoOutbounds(t *testing.T) {
	_, err := ConvertShareLinksToXrayJson(`{"outbounds":[]}`)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "outbound")
}

func TestConvertShareLinksToXrayJson_Base64EncodedLines(t *testing.T) {
	lines := "trojan://secret@trojan.example.com:443?sni=trojan.example.com\n" +
		"ss://" + ssUserB64("aes-128-gcm", "pwd") + "@ss.example.com:8388#ssn"
	blob := base64.StdEncoding.EncodeToString([]byte(lines))

	cfg, err := ConvertShareLinksToXrayJson(blob)
	require.NoError(t, err)
	require.Len(t, cfg.OutboundConfigs, 2)
	assert.Equal(t, "trojan", cfg.OutboundConfigs[0].Protocol)
	assert.Equal(t, "shadowsocks", cfg.OutboundConfigs[1].Protocol)
}

func TestConvertShareLinksToXrayJson_Base64URLSafeBlob(t *testing.T) {
	inner := "vless://" + testShareUUID + "@v.example.com:443?encryption=none&security=none"
	b := base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString([]byte(inner))
	cfg, err := ConvertShareLinksToXrayJson(b)
	require.NoError(t, err)
	require.Len(t, cfg.OutboundConfigs, 1)
	assert.Equal(t, "vless", cfg.OutboundConfigs[0].Protocol)
}

func TestConvertShareLinksToXrayJson_Shadowsocks(t *testing.T) {
	link := "ss://" + ssUserB64("chacha20-ietf-poly1305", "mypass") + "@10.0.0.1:8388#frag"
	cfg, err := ConvertShareLinksToXrayJson(link)
	require.NoError(t, err)
	require.Len(t, cfg.OutboundConfigs, 1)
	ob := cfg.OutboundConfigs[0]
	assert.Equal(t, "shadowsocks", ob.Protocol)
	var s conf.ShadowsocksClientConfig
	require.NoError(t, json.Unmarshal(*ob.Settings, &s))
	assert.Equal(t, "chacha20-ietf-poly1305", s.Cipher)
	assert.Equal(t, "mypass", s.Password)
	assert.Equal(t, uint16(8388), s.Port)
}

func TestConvertShareLinksToXrayJson_VlessWSAndTLS(t *testing.T) {
	link := "vless://" + testShareUUID + "@edge.example:443?encryption=none&type=ws&path=%2Fws&host=cdn.edge&security=tls&sni=edge.example&alpn=h2%2Ch3&fp=chrome&insecure=1"
	cfg, err := ConvertShareLinksToXrayJson(link)
	require.NoError(t, err)
	require.Len(t, cfg.OutboundConfigs, 1)
	ss := cfg.OutboundConfigs[0].StreamSetting
	require.NotNil(t, ss)
	assert.Equal(t, "tls", ss.Security)
	require.NotNil(t, ss.WSSettings)
	assert.Equal(t, "/ws", ss.WSSettings.Path)
	assert.Equal(t, "cdn.edge", ss.WSSettings.Host)
	require.NotNil(t, ss.TLSSettings)
	assert.Equal(t, "edge.example", ss.TLSSettings.ServerName)
	assert.Equal(t, "chrome", ss.TLSSettings.Fingerprint)
	assert.True(t, ss.TLSSettings.AllowInsecure)
	require.NotNil(t, ss.TLSSettings.ALPN)
	assert.Contains(t, []string(*ss.TLSSettings.ALPN), "h2")
}

func TestConvertShareLinksToXrayJson_VlessReality(t *testing.T) {
	pbk := "ZXYAbCdEfGhIjKlMnOpQrStUvWxYz0123456789ABCD"
	link := "vless://" + testShareUUID + "@reality.example:443?encryption=none&security=reality&type=tcp&sni=reality.example&pbk=" +
		pbk + "&sid=abcd&fp=qq&pqv=pqv1&spx=%2F"
	cfg, err := ConvertShareLinksToXrayJson(link)
	require.NoError(t, err)
	ss := cfg.OutboundConfigs[0].StreamSetting
	require.NotNil(t, ss)
	assert.Equal(t, "reality", ss.Security)
	require.NotNil(t, ss.REALITYSettings)
	assert.Equal(t, pbk, ss.REALITYSettings.PublicKey)
	assert.Equal(t, "abcd", ss.REALITYSettings.ShortId)
	assert.Equal(t, "qq", ss.REALITYSettings.Fingerprint)
	assert.Equal(t, "pqv1", ss.REALITYSettings.Mldsa65Verify)
	assert.Equal(t, "/", ss.REALITYSettings.SpiderX)
}

func TestConvertShareLinksToXrayJson_Trojan(t *testing.T) {
	link := "trojan://tpw@trojan.host:4443?sni=trojan.host#tname"
	cfg, err := ConvertShareLinksToXrayJson(link)
	require.NoError(t, err)
	var s conf.TrojanClientConfig
	require.NoError(t, json.Unmarshal(*cfg.OutboundConfigs[0].Settings, &s))
	assert.Equal(t, "tpw", s.Password)
	assert.Equal(t, uint16(4443), s.Port)
	ss := cfg.OutboundConfigs[0].StreamSetting
	require.NotNil(t, ss)
	assert.Equal(t, "tls", ss.Security)
}

func TestConvertShareLinksToXrayJson_SocksWithAuth(t *testing.T) {
	u := base64.StdEncoding.EncodeToString([]byte("socksuser:sockspass"))
	link := "socks://" + u + "@127.0.0.1:1081#sk"
	cfg, err := ConvertShareLinksToXrayJson(link)
	require.NoError(t, err)
	var s conf.SocksClientConfig
	require.NoError(t, json.Unmarshal(*cfg.OutboundConfigs[0].Settings, &s))
	assert.Equal(t, "socksuser", s.Username)
	assert.Equal(t, "sockspass", s.Password)
}

func TestConvertShareLinksToXrayJson_VmessPlainURL(t *testing.T) {
	link := "vmess://" + testShareUUID + "@vm.example:443?encryption=auto&type=tcp&headerType=http&path=%2Fpath1%2C%2Fpath2&host=h1%2Ch2"
	cfg, err := ConvertShareLinksToXrayJson(link)
	require.NoError(t, err)
	var s conf.VMessOutboundConfig
	require.NoError(t, json.Unmarshal(*cfg.OutboundConfigs[0].Settings, &s))
	assert.Equal(t, testShareUUID, s.ID)
	assert.Equal(t, "auto", s.Security)
	ss := cfg.OutboundConfigs[0].StreamSetting
	require.NotNil(t, ss)
	require.NotNil(t, ss.RAWSettings)
}

func TestConvertShareLinksToXrayJson_VmessBase64QR(t *testing.T) {
	qr := `{"ps":"qrname","add":"vm.add","port":"8443","id":"` + testShareUUID + `","scy":"auto","net":"ws","host":"ws.host","path":"/w","tls":"tls","sni":"tls.sni","alpn":"h2,h3","fp":"safari"}`
	b64 := base64.StdEncoding.EncodeToString([]byte(qr))
	link := "vmess://" + b64
	cfg, err := ConvertShareLinksToXrayJson(link)
	require.NoError(t, err)
	var s conf.VMessOutboundConfig
	require.NoError(t, json.Unmarshal(*cfg.OutboundConfigs[0].Settings, &s))
	assert.Equal(t, testShareUUID, s.ID)
	ss := cfg.OutboundConfigs[0].StreamSetting
	require.NotNil(t, ss)
	require.NotNil(t, ss.WSSettings)
	assert.Equal(t, "/w", ss.WSSettings.Path)
	assert.Equal(t, "tls", ss.Security)
	require.NotNil(t, ss.TLSSettings)
	assert.Equal(t, "tls.sni", ss.TLSSettings.ServerName)
}

func TestConvertShareLinksToXrayJson_TransportKcpGrpcHttpUpgradeXhttp(t *testing.T) {
	t.Run("kcp", func(t *testing.T) {
		link := "vless://" + testShareUUID + "@k.example:443?encryption=none&type=kcp&headerType=srtp&seed=myseed"
		cfg, err := ConvertShareLinksToXrayJson(link)
		require.NoError(t, err)
		ss := cfg.OutboundConfigs[0].StreamSetting
		require.NotNil(t, ss.KCPSettings)
		require.NotNil(t, ss.KCPSettings.Seed)
		assert.Equal(t, "myseed", *ss.KCPSettings.Seed)
	})

	t.Run("grpc", func(t *testing.T) {
		link := "vless://" + testShareUUID + "@g.example:443?encryption=none&type=grpc&serviceName=svc&authority=auth.here&mode=multi"
		cfg, err := ConvertShareLinksToXrayJson(link)
		require.NoError(t, err)
		gs := cfg.OutboundConfigs[0].StreamSetting.GRPCSettings
		require.NotNil(t, gs)
		assert.Equal(t, "svc", gs.ServiceName)
		assert.Equal(t, "auth.here", gs.Authority)
		assert.True(t, gs.MultiMode)
	})

	t.Run("httpupgrade", func(t *testing.T) {
		link := "vless://" + testShareUUID + "@hu.example:443?encryption=none&type=httpupgrade&path=%2Fup&host=hu.host"
		cfg, err := ConvertShareLinksToXrayJson(link)
		require.NoError(t, err)
		h := cfg.OutboundConfigs[0].StreamSetting.HTTPUPGRADESettings
		require.NotNil(t, h)
		assert.Equal(t, "/up", h.Path)
		assert.Equal(t, "hu.host", h.Host)
	})

	t.Run("xhttp_extra", func(t *testing.T) {
		extra := `{"host":"xh.extra"}`
		link := "vless://" + testShareUUID + "@xh.example:443?encryption=none&type=xhttp&path=%2Fx&host=xh.host&mode=stream-up&extra=" +
			url.QueryEscape(extra)
		cfg, err := ConvertShareLinksToXrayJson(link)
		require.NoError(t, err)
		x := cfg.OutboundConfigs[0].StreamSetting.XHTTPSettings
		require.NotNil(t, x)
		assert.Equal(t, "stream-up", x.Mode)
		require.NotNil(t, x.Extra)
	})
}

func TestConvertShareLinksToXrayJson_FinalMaskQuery(t *testing.T) {
	fm := `{"udp":[{"type":"test-mask"}]}`
	link := "vless://" + testShareUUID + "@fm.example:443?encryption=none&type=tcp&fm=" + url.QueryEscape(fm)
	cfg, err := ConvertShareLinksToXrayJson(link)
	require.NoError(t, err)
	ss := cfg.OutboundConfigs[0].StreamSetting
	require.NotNil(t, ss.FinalMask)
	require.Len(t, ss.FinalMask.Udp, 1)
	assert.Equal(t, "test-mask", ss.FinalMask.Udp[0].Type)
}

func TestConvertShareLinksToXrayJson_Hysteria2InvalidHop(t *testing.T) {
	_, err := ConvertShareLinksToXrayJson("hy2://auth@host:443?hop-interval=notint&sni=x.com")
	require.Error(t, err)
}

func TestConvertShareLinksToXrayJson_MultiLineSkipsBad(t *testing.T) {
	bad := "vmess://" + testShareUUID + "@bad.example:notaport?encryption=none"
	good := "vless://" + testShareUUID + "@ok.example:443?encryption=none"
	cfg, err := ConvertShareLinksToXrayJson(bad + "\n\n" + good)
	require.NoError(t, err)
	require.Len(t, cfg.OutboundConfigs, 1)
	assert.Equal(t, "vless", cfg.OutboundConfigs[0].Protocol)
}

func TestConvertShareLinksToXrayJson_RawClashYAML(t *testing.T) {
	yaml := `proxies:
  - name: clash-ss
    type: ss
    server: c.example
    port: 8390
    cipher: aes-256-gcm
    password: yamlpw`
	cfg, err := ConvertShareLinksToXrayJson(yaml)
	require.NoError(t, err)
	require.Len(t, cfg.OutboundConfigs, 1)
	assert.Equal(t, "shadowsocks", cfg.OutboundConfigs[0].Protocol)
}

func TestConvertShareLinksToXrayJson_VmessQRGrpcAndKcp(t *testing.T) {
	t.Run("grpc", func(t *testing.T) {
		qr := `{"ps":"g","add":"grpc.host","port":"443","id":"` + testShareUUID + `","net":"grpc","path":"svcname","type":"multi"}`
		link := "vmess://" + base64.StdEncoding.EncodeToString([]byte(qr))
		cfg, err := ConvertShareLinksToXrayJson(link)
		require.NoError(t, err)
		gs := cfg.OutboundConfigs[0].StreamSetting.GRPCSettings
		require.NotNil(t, gs)
		assert.Equal(t, "svcname", gs.ServiceName)
		assert.True(t, gs.MultiMode)
	})

	t.Run("kcp", func(t *testing.T) {
		qr := `{"ps":"k","add":"kcp.host","port":"8391","id":"` + testShareUUID + `","net":"kcp","path":"seedval","type":"wireguard"}`
		link := "vmess://" + base64.StdEncoding.EncodeToString([]byte(qr))
		cfg, err := ConvertShareLinksToXrayJson(link)
		require.NoError(t, err)
		ks := cfg.OutboundConfigs[0].StreamSetting.KCPSettings
		require.NotNil(t, ks)
		require.NotNil(t, ks.Seed)
		assert.Equal(t, "seedval", *ks.Seed)
	})
}

func TestConvertShareLinksToXrayJson_ShadowsocksWithStreamQuery(t *testing.T) {
	link := "ss://" + ssUserB64("aes-128-gcm", "p") + "@ss-ws.example:443?type=ws&path=%2Fws&host=cdn.ws&security=tls&sni=ss-ws.example"
	cfg, err := ConvertShareLinksToXrayJson(link)
	require.NoError(t, err)
	ss := cfg.OutboundConfigs[0].StreamSetting
	require.NotNil(t, ss.WSSettings)
	assert.Equal(t, "tls", ss.Security)
}
