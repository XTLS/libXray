package share

import (
	"encoding/base64"
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xtls/xray-core/infra/conf"
)

func TestMarshalShareConfigProjectsSupportedFields(t *testing.T) {
	const input = `{
		"log":{"loglevel":"warning"},
		"outbounds":[
			{"protocol":"freedom","settings":{}},
			{
				"protocol":"vless",
				"settings":{"address":"drop.example","port":443,"id":"12345678-abcd-abcd-abcd-123456789abc","encryption":"none"},
				"streamSettings":{"network":"quic","security":"tls","tlsSettings":{"fingerprint":"chrome"}}
			},
			{
				"protocol":"vless",
				"settings":{"address":"drop.example","port":443,"id":"12345678-abcd-abcd-abcd-123456789abc","encryption":"none"},
				"streamSettings":{"network":"raw","security":"xtls"}
			},
			{
				"protocol":"VLESS",
				"sendThrough":"127.0.0.1",
				"tag":"Node name",
				"settings":{
					"address":"example.com","port":443,"id":"12345678-abcd-abcd-abcd-123456789abc",
					"flow":"","encryption":"none","level":1,"email":"drop@example.com","seed":"drop","reverse":{}
				},
				"streamSettings":{
					"address":"drop.example","port":8443,"network":"splithttp","security":"REALITY",
					"xhttpSettings":{
						"host":"cdn.example.com","path":"/xhttp","mode":"stream-up","headers":{"X-Drop":"yes"},
						"extra":{"maxConcurrency":"1-2","empty":null}
					},
					"realitySettings":{
						"show":false,"target":null,"dest":null,"serverNames":null,"privateKey":"",
						"serverName":"example.com","fingerprint":"chrome","password":"",
						"publicKey":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","shortId":"abcd","spiderX":"/"
					},
					"sockopt":{"dialerProxy":"drop"}
				},
				"proxySettings":{"tag":"drop"},"mux":{"enabled":true},"targetStrategy":"UseIP"
			}
		]
	}`

	var config conf.Config
	require.NoError(t, json.Unmarshal([]byte(input), &config))
	raw, err := marshalShareConfigJSON(&config)
	require.NoError(t, err)

	const expected = `{
		"outbounds":[{
			"protocol":"vless",
			"sendThrough":"127.0.0.1",
			"tag":"Node name",
			"settings":{"address":"example.com","port":443,"id":"12345678-abcd-abcd-abcd-123456789abc","encryption":"none"},
			"streamSettings":{
				"network":"xhttp","security":"reality",
				"xhttpSettings":{"host":"cdn.example.com","path":"/xhttp","mode":"stream-up","extra":{"maxConcurrency":"1-2","empty":null}},
				"realitySettings":{"serverName":"example.com","fingerprint":"chrome","password":"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA","shortId":"abcd","spiderX":"/"}
			}
		}]
	}`
	assert.JSONEq(t, expected, string(raw))
	requireProjectedOutboundsBuild(t, raw)
}

func TestConvertShareLinksNativeJSONPreservesHysteriaPortHopping(t *testing.T) {
	raw, err := ConvertShareLinksToXrayJson(`{
		"outbounds": [{
			"protocol": "hysteria",
			"settings": {"version": 2, "address": "host", "port": 443},
			"streamSettings": {
				"network": "hysteria", "security": "tls",
				"tlsSettings": {"serverName": "example.com", "fingerprint": "chrome"},
				"hysteriaSettings": {"version": 2, "auth": "auth"},
				"finalmask": {
					"quicParams": {"congestion": "brutal", "brutalUp": "50 mbps", "brutalDown": "100 mbps"},
					"udp": [{"type": "udphop", "settings": {"mode": "intervalRemote", "remotePorts": "20000-40000", "interval": "30"}}]
				}
			}
		}]
	}`, "")
	require.NoError(t, err)

	var document map[string]any
	require.NoError(t, json.Unmarshal(raw, &document))
	outbound := document["outbounds"].([]any)[0].(map[string]any)
	stream := outbound["streamSettings"].(map[string]any)
	finalMask := stream["finalmask"].(map[string]any)
	quicParams := finalMask["quicParams"].(map[string]any)
	assert.NotContains(t, quicParams, "udpHop")
	mask := finalMask["udp"].([]any)[0].(map[string]any)
	assert.Equal(t, "udphop", mask["type"])
	udpHop := mask["settings"].(map[string]any)
	assert.Equal(t, "intervalRemote", udpHop["mode"])
	assert.Equal(t, "20000-40000", udpHop["remotePorts"])
	assert.Equal(t, "30", udpHop["interval"])
	requireProjectedOutboundsBuild(t, raw)
}

func TestMarshalShareConfigKeepsKCPWithoutSettings(t *testing.T) {
	link := "vmess://" + testShareUUID + "@kcp.host:8391?type=kcp&seed=seedval&headerType=wireguard#k"
	config, err := convertShareLinksForTest(link)
	require.NoError(t, err)
	raw, err := marshalShareConfigJSON(config)
	require.NoError(t, err)

	assert.Contains(t, string(raw), `"network":"kcp"`)
	assert.NotContains(t, string(raw), "kcpSettings")
	assert.NotContains(t, string(raw), "seedval")
	assert.NotContains(t, string(raw), "wireguard")
	requireProjectedOutboundsBuild(t, raw)
}

func TestMarshalShareConfigSupportedProtocolsBuild(t *testing.T) {
	tests := map[string]string{
		"shadowsocks": "ss://" + ssUserB64("chacha20-ietf-poly1305", "password") + "@10.0.0.1:8388",
		"vmess":       "vmess://" + testShareUUID + "@vm.example:443?encryption=auto&type=raw",
		"vless":       "vless://" + testShareUUID + "@10.0.0.1:443?encryption=none&security=none",
		"socks":       "socks://" + base64.StdEncoding.EncodeToString([]byte("user:password")) + "@127.0.0.1:1080",
		"trojan":      "trojan://password@trojan.example:443?sni=trojan.example",
	}
	for protocol, link := range tests {
		t.Run(protocol, func(t *testing.T) {
			config, err := convertShareLinksForTest(link)
			require.NoError(t, err)
			raw, err := marshalShareConfigJSON(config)
			require.NoError(t, err)
			requireProjectedOutboundsBuild(t, raw)
		})
	}
}

func requireProjectedOutboundsBuild(t *testing.T, raw json.RawMessage) {
	t.Helper()
	var config conf.Config
	require.NoError(t, json.Unmarshal(raw, &config))
	require.NotEmpty(t, config.OutboundConfigs)
	for index := range config.OutboundConfigs {
		_, err := config.OutboundConfigs[index].Build()
		require.NoError(t, err)
	}
}
