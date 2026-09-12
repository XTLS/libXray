package share

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xtls/xray-core/infra/conf"
)

func TestConvertXrayJsonToShareLinks_RoundTripProtocols(t *testing.T) {
	cases := []string{
		"vless://" + testShareUUID + "@r1.example:443?encryption=none&security=tls&sni=r1.example&type=ws&path=%2Fr&host=h.r1",
		"trojan://trpass@r2.example:443?sni=r2.example",
		"ss://" + ssUserB64("aes-128-gcm", "pw") + "@r3.example:8389",
		"vmess://" + testShareUUID + "@r4.example:443?encryption=none&type=tcp",
		"socks://" + base64.StdEncoding.EncodeToString([]byte("u:p")) + "@127.0.0.1:1090",
		"vmess://" + testShareUUID + "@vm.example:443?encryption=auto&type=ws&path=%2Fws&security=tls#VMessAEAD",
	}
	for _, link := range cases {
		t.Run(link[:12], func(t *testing.T) {
			cfg, err := convertShareLinksWithKeyForTest(link, "")
			require.NoError(t, err)
			out, err := json.Marshal(cfg)
			require.NoError(t, err)
			text, err := ConvertXrayJsonToShareLinks(out)
			require.NoError(t, err)
			assert.NotEmpty(t, text)
			if cfg.OutboundConfigs[0].Protocol == "vmess" {
				assert.Contains(t, text, "vmess://"+testShareUUID+"@")
			}
			again, err := convertShareLinksWithKeyForTest(text, "")
			require.NoError(t, err)
			require.Len(t, again.OutboundConfigs, 1)
			assert.Equal(t, cfg.OutboundConfigs[0].Protocol, again.OutboundConfigs[0].Protocol)
		})
	}
}

func TestGenerate_KCPIgnoresSeedAndHeader(t *testing.T) {
	config, err := convertShareLinksForTest(
		"vless://" + testShareUUID + "@kcp.example:443?encryption=none&type=kcp",
	)
	require.NoError(t, err)

	seed := "legacy-seed"
	header := json.RawMessage(`{"type":"srtp"}`)
	config.OutboundConfigs[0].StreamSetting.KCPSettings = &conf.KCPConfig{
		Seed:         &seed,
		HeaderConfig: header,
	}

	link, err := shareLink(config.OutboundConfigs[0])
	require.NoError(t, err)
	assert.Equal(t, "kcp", link.Query().Get("type"))
	assert.Empty(t, link.Query().Get("seed"))
	assert.Empty(t, link.Query().Get("headerType"))
}

func TestGenerate_ShadowsocksAEAD2022PlainUserInfo(t *testing.T) {
	const original = "ss://2022-blake3-aes-256-gcm:" +
		"YctPZ6U7xPPcU%2Bgp3u%2B0tx%2FtRizJN9K8y%2BuKlW2qjlI%3D" +
		"@192.168.100.1:8888#Example3"

	config, err := convertShareLinksForTest(original)
	require.NoError(t, err)
	require.Len(t, config.OutboundConfigs, 1)

	generated, err := shareLink(config.OutboundConfigs[0])
	require.NoError(t, err)
	assert.Equal(t, original, generated.String())
}

func TestGenerate_ShadowsocksLegacyBase64UserInfo(t *testing.T) {
	original := "ss://" + ssUserB64("aes-128-gcm", "password") +
		"@ss.example.com:8388#Legacy"

	config, err := convertShareLinksForTest(original)
	require.NoError(t, err)
	require.Len(t, config.OutboundConfigs, 1)

	generated, err := shareLink(config.OutboundConfigs[0])
	require.NoError(t, err)
	assert.Equal(t, original, generated.String())
}

func TestConvertXrayJsonToShareLinks_Errors(t *testing.T) {
	_, err := ConvertXrayJsonToShareLinks([]byte(`{"outbounds":[]}`))
	require.Error(t, err)

	_, err = ConvertXrayJsonToShareLinks([]byte(`{`))
	require.Error(t, err)

	_, err = ConvertXrayJsonToShareLinks([]byte(`null`))
	require.Error(t, err)

	_, err = ConvertXrayJsonToShareLinks([]byte(`{
		"outbounds": [{"protocol": "shadowsocks"}]
	}`))
	require.Error(t, err)

	_, err = ConvertXrayJsonToShareLinks([]byte(`{
		"outbounds": [{"protocol": "shadowsocks", "settings": null}]
	}`))
	require.Error(t, err)

	_, err = ConvertXrayJsonToShareLinks([]byte(`{
		"outbounds": [{"protocol": "freedom", "tag": "direct"}]
	}`))
	require.Error(t, err)
}

func TestConvertXrayJsonToShareLinksRejectsHysteria(t *testing.T) {
	links, err := ConvertXrayJsonToShareLinks([]byte(`{
		"outbounds": [{
			"protocol": "hysteria",
			"settings": {"version": 2, "address": "example.com", "port": 443},
			"streamSettings": {
				"network": "hysteria", "security": "tls",
				"hysteriaSettings": {"version": 2, "auth": "password"}
			}
		}]
	}`))
	require.EqualError(t, err, "no valid outbounds")
	assert.Empty(t, links)
}

func TestConvertXrayJsonToShareLinksSkipsUnsupportedOutbounds(t *testing.T) {
	links, err := ConvertXrayJsonToShareLinks([]byte(`{
		"outbounds": [
			{
				"protocol": "trojan",
				"settings": {
					"address": "example.com",
					"port": 443,
					"password": "password"
				}
			},
			{"protocol": "freedom", "tag": "direct"},
			{"protocol": "hysteria", "settings": {"version": 2, "address": "example.com", "port": 443}}
		]
	}`))
	require.NoError(t, err)
	assert.Equal(t, "trojan://password@example.com:443#trojan", links)
}

func TestConvertXrayJsonToShareLinks_IgnoresSendThroughForName(t *testing.T) {
	cfg, err := convertShareLinksForTest(`trojan://pw@tag.example:443`)
	require.NoError(t, err)
	ob := cfg.OutboundConfigs[0]
	sendThrough := "127.0.0.1"
	ob.SendThrough = &sendThrough
	ob.Tag = "named-by-tag"
	out, err := json.Marshal(&conf.Config{OutboundConfigs: []conf.OutboundDetourConfig{ob}})
	require.NoError(t, err)
	links, err := ConvertXrayJsonToShareLinks(out)
	require.NoError(t, err)
	assert.Contains(t, links, "#named-by-tag")
}
