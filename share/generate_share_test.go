package share

import (
	"encoding/json"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xtls/xray-core/infra/conf"
)

func buildHy2Outbound(t *testing.T, auth string, host string, port uint16, streamSetting *conf.StreamConfig) conf.OutboundDetourConfig {
	t.Helper()
	settings := &conf.HysteriaClientConfig{
		Address: parseAddress(host),
		Port:    port,
	}
	settingsJSON, err := json.Marshal(settings)
	require.NoError(t, err)
	raw := json.RawMessage(settingsJSON)

	return conf.OutboundDetourConfig{
		Protocol:      "hysteria",
		Settings:      &raw,
		StreamSetting: streamSetting,
	}
}

func buildHy2StreamSettings(auth string, tls *conf.TLSConfig, finalMask *conf.FinalMask) *conf.StreamConfig {
	network := conf.TransportProtocol("hysteria")
	ss := &conf.StreamConfig{
		Network:  &network,
		Security: "tls",
	}
	ss.TLSSettings = tls
	ss.HysteriaSettings = &conf.HysteriaConfig{
		Version: 2,
		Auth:    auth,
	}
	ss.FinalMask = finalMask
	return ss
}

func TestGenerate_Hy2_WithBandwidth(t *testing.T) {
	tls := &conf.TLSConfig{ServerName: "example.com"}
	fm := &conf.FinalMask{
		QuicParams: &conf.QuicParamsConfig{
			Congestion: "brutal",
			BrutalUp:   conf.Bandwidth("100 mbps"),
			BrutalDown: conf.Bandwidth("200 mbps"),
		},
	}
	ss := buildHy2StreamSettings("auth", tls, fm)
	outbound := buildHy2Outbound(t, "auth", "host", 443, ss)

	link, err := shareLink(outbound)
	require.NoError(t, err)

	assert.Equal(t, "hysteria2", link.Scheme)
	q := link.Query()
	assert.Equal(t, "example.com", q.Get("sni"))
	assert.Equal(t, "100 mbps", q.Get("up"))
	assert.Equal(t, "200 mbps", q.Get("down"))
}

func TestGenerate_Hy2_WithSalamanderBandwidthPortHopping(t *testing.T) {
	tls := &conf.TLSConfig{ServerName: "example.com"}

	portListJSON, err := json.Marshal("20000-40000")
	require.NoError(t, err)

	fm := &conf.FinalMask{
		QuicParams: &conf.QuicParamsConfig{
			Congestion: "brutal",
			BrutalUp:   conf.Bandwidth("50 mbps"),
			BrutalDown: conf.Bandwidth("100 mbps"),
			UdpHop: conf.UdpHop{
				PortList: portListJSON,
				Interval: &conf.Int32Range{Left: 30, Right: 30, From: 30, To: 30},
			},
		},
	}

	salamander := &conf.Salamander{Password: "secret"}
	salJSON, err := json.Marshal(salamander)
	require.NoError(t, err)
	salRaw := json.RawMessage(salJSON)
	fm.Udp = []conf.Mask{{Type: "salamander", Settings: &salRaw}}

	ss := buildHy2StreamSettings("auth", tls, fm)
	outbound := buildHy2Outbound(t, "auth", "host", 443, ss)

	link, err := shareLink(outbound)
	require.NoError(t, err)

	q := link.Query()
	assert.Equal(t, "example.com", q.Get("sni"))
	assert.Equal(t, "50 mbps", q.Get("up"))
	assert.Equal(t, "100 mbps", q.Get("down"))
	assert.Equal(t, "20000-40000", q.Get("ports"))
	assert.Equal(t, "30", q.Get("hop-interval"))
	assert.Equal(t, "salamander", q.Get("obfs"))
	assert.Equal(t, "secret", q.Get("obfs-password"))
}

func TestGenerate_Hy2_WithFullTLSParams(t *testing.T) {
	alpn := conf.StringList{"h3"}
	tls := &conf.TLSConfig{
		ServerName:  "example.com",
		Fingerprint: "chrome",
		ALPN:        &alpn,
	}
	ss := buildHy2StreamSettings("auth", tls, nil)
	outbound := buildHy2Outbound(t, "auth", "host", 443, ss)

	link, err := shareLink(outbound)
	require.NoError(t, err)

	q := link.Query()
	assert.Equal(t, "example.com", q.Get("sni"))
	assert.Equal(t, "chrome", q.Get("fp"))
	assert.Equal(t, "h3", q.Get("alpn"))
}

func TestGenerate_Hy2_RoundTrip(t *testing.T) {
	original := "hy2://auth@host:443?up=50+mbps&down=100+mbps&obfs=salamander&obfs-password=secret&ports=20000-40000&hop-interval=30&sni=example.com&alpn=h3&fp=chrome"

	config, err := ConvertShareLinksToXrayJson(original)
	require.NoError(t, err)
	require.Len(t, config.OutboundConfigs, 1)

	generated, err := shareLink(config.OutboundConfigs[0])
	require.NoError(t, err)

	// Parse both URLs to compare query params (order may differ)
	origURL, err := url.Parse(original)
	require.NoError(t, err)
	origQ := origURL.Query()
	genQ := generated.Query()

	assert.Equal(t, "hysteria2", generated.Scheme)
	assert.Equal(t, origQ.Get("sni"), genQ.Get("sni"))
	assert.Equal(t, origQ.Get("fp"), genQ.Get("fp"))
	assert.Equal(t, origQ.Get("alpn"), genQ.Get("alpn"))
	assert.Equal(t, origQ.Get("up"), genQ.Get("up"))
	assert.Equal(t, origQ.Get("down"), genQ.Get("down"))
	assert.Equal(t, origQ.Get("ports"), genQ.Get("ports"))
	assert.Equal(t, origQ.Get("hop-interval"), genQ.Get("hop-interval"))
	assert.Equal(t, origQ.Get("obfs"), genQ.Get("obfs"))
	assert.Equal(t, origQ.Get("obfs-password"), genQ.Get("obfs-password"))
}
