package share

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xtls/xray-core/infra/conf"
)

func TestShareKCPFieldsRoundTrip(t *testing.T) {
	for _, protocol := range []string{"vmess", "vless"} {
		for _, params := range []string{"", "&mtu=1350", "&tti=30", "&mtu=1350&tti=30"} {
			t.Run(protocol+params, func(t *testing.T) {
				original := protocol + "://" + testShareUUID + "@kcp.example:443?type=kcp" + params
				config, link := roundTripShareFields(t, original)
				assert.Equal(t, "kcp", link.Query().Get("type"))
				settings := config.OutboundConfigs[0].StreamSetting.KCPSettings
				if params == "" {
					assert.Nil(t, settings)
					assert.NotContains(t, link.Query(), "mtu")
					assert.NotContains(t, link.Query(), "tti")
					return
				}
				require.NotNil(t, settings)
				if strings.Contains(params, "mtu=") {
					require.NotNil(t, settings.Mtu)
					assert.EqualValues(t, 1350, *settings.Mtu)
					assert.Equal(t, "1350", link.Query().Get("mtu"))
				} else {
					assert.Nil(t, settings.Mtu)
					assert.NotContains(t, link.Query(), "mtu")
				}
				if strings.Contains(params, "tti=") {
					require.NotNil(t, settings.Tti)
					assert.EqualValues(t, 30, *settings.Tti)
					assert.Equal(t, "30", link.Query().Get("tti"))
				} else {
					assert.Nil(t, settings.Tti)
					assert.NotContains(t, link.Query(), "tti")
				}
			})
		}
	}
}

func TestShareRejectsInvalidTransportFields(t *testing.T) {
	good := "vless://" + testShareUUID + "@valid.example:443"
	for _, query := range []string{
		"type=kcp&mtu=abc", "type=kcp&mtu=-1", "type=kcp&mtu=4294967296",
		"type=kcp&mtu=20", "type=kcp&tti=9", "type=kcp&tti=1001", "type=kcp&tti=1.5",
		"type=grpc&mode=guna", "type=xhttp&extra=%7B", "type=xhttp&extra=123",
	} {
		t.Run(query, func(t *testing.T) {
			bad := "vless://" + testShareUUID + "@invalid.example:443?" + query
			_, err := ConvertShareLinksToXrayJson(bad, "")
			require.Error(t, err)
			config, err := convertShareLinksWithKeyForTest(bad+"\n"+good, "")
			require.NoError(t, err)
			require.Len(t, config.OutboundConfigs, 1)
			assert.Contains(t, string(*config.OutboundConfigs[0].Settings), "valid.example")
		})
	}
}

func TestShareXHTTPExtraRoundTrip(t *testing.T) {
	const extra = `{
		"headers":{"X-Test":"a + b / c"},"noGRPCHeader":true,
		"downloadSettings":{"network":"xhttp","xhttpSettings":{"path":"/download","headers":{"X-Download":"yes"}}},
		"futureOption":{"number":9007199254740993,"empty":null,"enabled":false}
	}`
	for _, protocol := range []string{"vmess", "vless"} {
		t.Run(protocol, func(t *testing.T) {
			query := url.Values{
				"type": {"xhttp"}, "host": {"cdn.example"}, "path": {"/path with + space"},
				"mode": {"stream-up"}, "extra": {extra},
			}
			config, link := roundTripShareFields(t, protocol+"://"+testShareUUID+"@xhttp.example:443?"+query.Encode())
			assert.JSONEq(t, extra, string(config.OutboundConfigs[0].StreamSetting.XHTTPSettings.Extra))
			assert.JSONEq(t, extra, link.Query().Get("extra"))
			assert.Contains(t, link.Query().Get("extra"), "9007199254740993")
			assert.Equal(t, query.Get("path"), link.Query().Get("path"))
			assert.NotContains(t, link.RawQuery, "+", "query values must use percent-encoded spaces")
		})
	}
}

func TestShareFinalMaskRoundTrip(t *testing.T) {
	// Core validation enables these when quicParams.debug is true.
	t.Setenv("HYSTERIA_BBR_DEBUG", "")
	t.Setenv("HYSTERIA_BRUTAL_DEBUG", "")
	const fm = `{
		"tcp":null,
		"udp":[{"type":"noise","settings":{"futureOption":{"enabled":false,"empty":null}}}],
		"quicParams":{
			"congestion":"bbr","debug":true,"bbrProfile":"conservative",
			"brutalUp":"50 mbps","brutalDown":"100 mbps","brutalDisableLossCompensation":true,
			"initStreamReceiveWindow":65536,"maxStreamReceiveWindow":131072,
			"initConnectionReceiveWindow":262144,"maxConnectionReceiveWindow":524288,
			"maxIdleTimeout":45,"keepAlivePeriod":10,"disablePathMTUDiscovery":true,
			"disableChromeParrot":true,"disableGSO":true,"maxIncomingStreams":32,"disableStatelessReset":true
		}
	}`
	config, link := roundTripShareFields(t, "vless://"+testShareUUID+"@mask.example:443?fm="+url.QueryEscape(fm))
	projected, err := json.Marshal(config.OutboundConfigs[0].StreamSetting.FinalMask)
	require.NoError(t, err)
	assert.JSONEq(t, fm, string(projected))
	assert.JSONEq(t, fm, link.Query().Get("fm"))
}

func TestShareSecurityFieldsRoundTrip(t *testing.T) {
	for name, query := range map[string]url.Values{
		"tls": {
			"security": {"tls"}, "fp": {"chrome"}, "sni": {"tls.example"}, "alpn": {"h2,http/1.1"},
			"ech": {"config+with/padding="}, "pcs": {strings.Repeat("01", 32) + "," + strings.Repeat("02", 32)},
			"vcn": {"one.example,two.example"},
		},
		"reality": {
			"security": {"reality"}, "fp": {"chrome"}, "sni": {"reality.example"},
			"pbk": {base64.RawURLEncoding.EncodeToString(make([]byte, 32))}, "sid": {"abcd"},
			"pqv": {base64.RawURLEncoding.EncodeToString(make([]byte, 1952))}, "spx": {"/path?key=a + b"},
			"flow": {"xtls-rprx-vision"},
		},
	} {
		t.Run(name, func(t *testing.T) {
			_, link := roundTripShareFields(t, "vless://"+testShareUUID+"@security.example:443?"+query.Encode())
			for key, values := range query {
				assert.Equal(t, values, link.Query()[key], key)
			}
		})
	}
}

func TestShareDefaultSNIUsesRemoteHost(t *testing.T) {
	for _, protocol := range []string{"vmess", "vless"} {
		t.Run(protocol, func(t *testing.T) {
			config, link := roundTripShareFields(t, protocol+"://"+testShareUUID+"@remote.example:443?type=ws&host=cdn.example&security=tls")
			assert.Equal(t, "remote.example", config.OutboundConfigs[0].StreamSetting.TLSSettings.ServerName)
			assert.Equal(t, "remote.example", link.Query().Get("sni"))
		})
	}
}

func TestGenerateShareTransportAliases(t *testing.T) {
	for name, stream := range map[string]string{
		"tcp":       `{"network":"tcp","tcpSettings":{"header":{"type":"http","request":{"path":["/a"],"headers":{"Host":["cdn.example"]}}}}}`,
		"raw":       `{"network":"raw","rawSettings":{"header":{"type":"http","request":{"path":["/a"],"headers":{"Host":["cdn.example"]}}}}}`,
		"websocket": `{"network":"websocket","wsSettings":{"path":"/a","host":"cdn.example"}}`,
		"splithttp": `{"network":"splithttp","splithttpSettings":{"path":"/a","host":"cdn.example","mode":"stream-up","extra":{"futureOption":true}}}`,
		"method":    `{"network":"ws","method":"tcp","rawSettings":{"header":{"type":"http","request":{"path":["/a"],"headers":{"Host":["cdn.example"]}}}}}`,
	} {
		t.Run(name, func(t *testing.T) {
			input := `{"outbounds":[{"protocol":"vless","settings":{"address":"remote.example","port":443,"id":"` + testShareUUID + `","encryption":"none"},"streamSettings":` + stream + `}]}`
			text, err := ConvertXrayJsonToShareLinks([]byte(input))
			require.NoError(t, err)
			link, err := url.Parse(text)
			require.NoError(t, err)
			assert.Equal(t, "/a", link.Query().Get("path"))
			assert.Equal(t, "cdn.example", link.Query().Get("host"))
			if name == "tcp" || name == "raw" || name == "method" {
				assert.Equal(t, "tcp", link.Query().Get("type"))
				assert.Equal(t, "http", link.Query().Get("headerType"))
			}
			if name == "splithttp" {
				assert.Equal(t, "xhttp", link.Query().Get("type"))
				assert.JSONEq(t, `{"futureOption":true}`, link.Query().Get("extra"))
			}
			_, err = ConvertShareLinksToXrayJson(text, "")
			require.NoError(t, err)
		})
	}
}

func roundTripShareFields(t *testing.T, original string) (*conf.Config, *url.URL) {
	t.Helper()
	raw, err := ConvertShareLinksToXrayJson(original+"#Field%20test", "")
	require.NoError(t, err)
	text, err := ConvertXrayJsonToShareLinks(raw)
	require.NoError(t, err)
	link, err := url.Parse(text)
	require.NoError(t, err)
	again, err := ConvertShareLinksToXrayJson(text, "")
	require.NoError(t, err)
	assert.JSONEq(t, string(raw), string(again))
	var config conf.Config
	require.NoError(t, json.Unmarshal(raw, &config))
	require.Len(t, config.OutboundConfigs, 1)
	return &config, link
}
