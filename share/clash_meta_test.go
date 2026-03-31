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
