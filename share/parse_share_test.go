package share

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xtls/xray-core/infra/conf"
)

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
