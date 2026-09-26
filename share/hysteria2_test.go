package share

import (
	"encoding/base64"
	"encoding/json"
	"net"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/xtls/xray-core/infra/conf"
	"github.com/xtls/xray-core/transport/internet/finalmask"
)

func TestHysteria2ImportAndRoundTrip(t *testing.T) {
	for _, test := range []struct {
		name, text, host, auth string
		port                   uint16
	}{
		{"default TLS and port", "hy2://password@example.com", "example.com", "password", 443},
		{"empty auth", "hysteria2://example.com/", "example.com", "", 443},
		{"userpass", "hysteria2://user:p%3Aa%25ss%40word@example.com:8443/", "example.com", "user:p:a%ss@word", 8443},
		{"literal plus", "hy2://a+b%2Bc%252B@example.com#Test%20%E6%B5%8B%E8%AF%95", "example.com", "a+b+c%2B", 443},
		{"IPv6 default", "hy2://auth@[2001:db8::1]", "2001:db8::1", "auth", 443},
		{"IPv6 port", "hysteria2://auth@[2001:db8::2]:8443?sni=server.example", "2001:db8::2", "auth", 8443},
	} {
		t.Run(test.name, func(t *testing.T) {
			data, err := ConvertShareLinksToXrayJson(test.text, "")
			require.NoError(t, err)
			var config conf.Config
			require.NoError(t, json.Unmarshal(data, &config))
			require.Len(t, config.OutboundConfigs, 1)
			outbound := config.OutboundConfigs[0]
			settings, err := decodeOutboundSettings[conf.HysteriaClientConfig](outbound)
			require.NoError(t, err)
			assert.Equal(t, parseAddress(test.host), settings.Address)
			assert.Equal(t, test.port, settings.Port)
			assert.Equal(t, int32(2), settings.Version)
			assert.Equal(t, "hysteria", outbound.Protocol)
			assert.Equal(t, "tls", outbound.StreamSetting.Security)
			assert.Equal(t, test.auth, outbound.StreamSetting.HysteriaSettings.Auth)
			assert.Equal(t, int32(2), outbound.StreamSetting.HysteriaSettings.Version)
			requireProjectedOutboundsBuild(t, data)
			generated, err := ConvertXrayJsonToShareLinks(data)
			require.NoError(t, err)
			assert.True(t, strings.HasPrefix(generated, "hysteria2://"))
			again, err := ConvertShareLinksToXrayJson(generated, "")
			require.NoError(t, err)
			// Nameless nodes acquire the existing protocol-name fallback on export.
			if outbound.Tag == "" {
				outbound.Tag = "hysteria"
				config.OutboundConfigs[0] = outbound
				data, err = marshalShareConfigJSON(&config)
				require.NoError(t, err)
			}
			assert.JSONEq(t, string(data), string(again))
		})
	}
}

func TestHysteria2TLSAndMaskRoundTrip(t *testing.T) {
	for _, authority := range []string{"example.com:443,5000-5002", "[2001:db8::1]:5000-5002"} {
		input := "hy2://auth@" + authority + "/?sni=server.example&obfs=salamander&obfs-password=pass%2Bword&hop-interval=5&fp=chrome&alpn=h3&ech=aGVsbG8%3D&pcs=" + strings.Repeat("ab", 32) + "&vcn=server.example#Node"
		data, err := ConvertShareLinksToXrayJson(input, "")
		require.NoError(t, err)
		generated, err := ConvertXrayJsonToShareLinks(data)
		require.NoError(t, err)
		assert.Contains(t, generated, "@"+authority)
		assert.NotContains(t, generated, "fm=")
		assert.NotContains(t, generated, "security=")
		assert.NotContains(t, generated, "type=")
		again, err := ConvertShareLinksToXrayJson(generated, "")
		require.NoError(t, err)
		assert.JSONEq(t, string(data), string(again))
	}
}

func TestHysteria2LegacyPortQueriesAndBandwidth(t *testing.T) {
	for _, key := range []string{"ports", "mport"} {
		config, err := convertShareLinksWithKeyForTest("hy2://auth@host:443?"+key+"=5000-5002&up=50%20mbps&down=100%20mbps", "")
		require.NoError(t, err)
		mask := config.OutboundConfigs[0].StreamSetting.FinalMask
		require.NotNil(t, mask)
		assert.Equal(t, conf.Bandwidth("50 mbps"), mask.QuicParams.BrutalUp)
		var hop conf.UDPHop
		require.NoError(t, json.Unmarshal(*mask.Udp[0].Settings, &hop))
		assert.Equal(t, "intervalLocal,intervalRemote", hop.Mode)
		assert.Equal(t, int32(30), hop.Interval.From)
		link, err := shareLink(config.OutboundConfigs[0])
		require.NoError(t, err)
		assert.Contains(t, link.String(), "host:5000-5002")
		assert.Empty(t, link.Query().Get("up"))
		assert.Empty(t, link.Query().Get("down"))
	}
}

func TestHysteria2RejectsInvalidOrUnsupportedLinks(t *testing.T) {
	for _, suffix := range []string{
		"", ":443", "example.com:", "example.com:0", "example.com:65536", "example.com:-1",
		"example.com:5002-5000", "example.com:443,", "example.com:443,,444", "example.com:env:PORT",
		"2001:db8::1", "[example.com]:443", "[2001:db8::1", "example.com/path",
		"example.com?insecure=1", "example.com?allowInsecure=true", "example.com?insecure=invalid",
		"example.com?insecure=0&insecure=1", "example.com?pinSHA256=" + strings.Repeat("ab", 32),
		"example.com?insecure=1&pinSHA256=" + strings.Repeat("ab", 32),
		"example.com?security=none", "example.com?security=reality",
		"example.com?obfs=gecko&obfs-password=secret", "example.com?obfs=salamander",
		"example.com?obfs=salamander&obfs-password=abc", "example.com?obfs-password=secret",
		"example.com?pcs=not-a-pin", "example.com?hop-interval=5",
		"example.com:443,444?hop-interval=0", "example.com:443,444?hop-interval=4",
		"example.com:443,444?hop-interval=no", "example.com:443,444?ports=445,446",
		"example.com?ports=443,444&mport=445,446", "example.com?ports=",
		"example.com?obfs=salamander&obfs-password=%zz", "example.com?up=not-bandwidth",
	} {
		t.Run(suffix, func(t *testing.T) {
			data, err := ConvertShareLinksToXrayJson("hy2://private-password@"+suffix, "")
			require.Error(t, err)
			assert.Nil(t, data)
			assert.NotContains(t, err.Error(), "private-password")
		})
	}
	for _, value := range []string{"0", "false"} {
		_, err := ConvertShareLinksToXrayJson("hy2://auth@host?insecure="+value, "")
		require.NoError(t, err)
	}
}

func TestHysteria2SubscriptionsKeepOrder(t *testing.T) {
	text := "hy2://one@first.example#First\n" + ageTestShareLink + "\nhy2://bad@bad.example?insecure=1\nhysteria2://last@last.example#Last"
	for _, input := range []string{text, base64.StdEncoding.EncodeToString([]byte(text)), base64.RawURLEncoding.EncodeToString([]byte(text))} {
		config, err := convertShareLinksWithKeyForTest(input, "")
		require.NoError(t, err)
		require.Len(t, config.OutboundConfigs, 3)
		assert.Equal(t, "First", config.OutboundConfigs[0].Tag)
		assert.Equal(t, "vless", config.OutboundConfigs[1].Protocol)
		assert.Equal(t, "Last", config.OutboundConfigs[2].Tag)
	}
}

func TestHysteria2ExportRejectsUnrepresentableSecurityAndMasks(t *testing.T) {
	for _, mutate := range []func(*conf.StreamConfig){
		func(s *conf.StreamConfig) { s.TLSSettings.AllowInsecure = true },
		func(s *conf.StreamConfig) { s.TLSSettings.DisableSystemRoot = true },
		func(s *conf.StreamConfig) { s.TLSSettings.MinVersion = "1.3" },
		func(s *conf.StreamConfig) { s.Method = new(conf.TransportProtocol("ws")) },
		func(s *conf.StreamConfig) { s.FinalMask = &conf.FinalMask{Udp: []conf.Mask{{Type: "noise"}}} },
		func(s *conf.StreamConfig) {
			raw := json.RawMessage(`{"password":"secret","packetSize":"100-200"}`)
			s.FinalMask = &conf.FinalMask{Udp: []conf.Mask{{Type: "salamander", Settings: &raw}}}
		},
		func(s *conf.StreamConfig) {
			raw := json.RawMessage(`{"mode":"intervalRemote","interval":30,"remotePorts":"443,444"}`)
			s.FinalMask = &conf.FinalMask{Udp: []conf.Mask{{Type: "udphop", Settings: &raw}}}
		},
	} {
		config, err := convertShareLinksForTest("hy2://auth@host?sni=host")
		require.NoError(t, err)
		mutate(config.OutboundConfigs[0].StreamSetting)
		_, err = shareLink(config.OutboundConfigs[0])
		require.Error(t, err)
	}
}

// Exercise packet replies, not just mask construction: the remote-only hopping
// mode used by the former converter never starts the pinned core's read loop.
func TestHysteria2PortHoppingReceivesAfterHop(t *testing.T) {
	if testing.Short() {
		t.Skip("waits for a real local UDP hop")
	}
	server, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = server.Close() })
	go func() {
		buf := make([]byte, 64)
		for {
			n, addr, err := server.ReadFrom(buf)
			if err != nil {
				return
			}
			_, _ = server.WriteTo([]byte(addr.String()+":"+string(buf[:n])), addr)
		}
	}()
	port := server.LocalAddr().(*net.UDPAddr).Port
	config, err := convertShareLinksForTest("hy2://auth@127.0.0.1:" + strconv.Itoa(port) + "," + strconv.Itoa(port) + "?hop-interval=5")
	require.NoError(t, err)
	mask := config.OutboundConfigs[0].StreamSetting.FinalMask.Udp[0]
	built, err := mask.Build(false)
	require.NoError(t, err)
	raw, err := net.ListenPacket("udp", "127.0.0.1:0")
	require.NoError(t, err)
	t.Cleanup(func() { _ = raw.Close() })
	conn, err := finalmask.NewUdpmaskManager([]finalmask.Udpmask{built.(finalmask.Udpmask)}).WrapPacketConnClient(raw)
	require.NoError(t, err)
	t.Cleanup(func() { _ = conn.Close() })
	exchange := func() string {
		t.Helper()
		require.NoError(t, conn.SetDeadline(time.Now().Add(2*time.Second)))
		_, err := conn.WriteTo([]byte("echo"), server.LocalAddr())
		require.NoError(t, err)
		buf := make([]byte, 128)
		n, _, err := conn.ReadFrom(buf)
		require.NoError(t, err)
		assert.True(t, strings.HasSuffix(string(buf[:n]), ":echo"))
		return string(buf[:n])
	}
	first := exchange()
	// Clear the first exchange's deadline before the replacement socket inherits it.
	require.NoError(t, conn.SetDeadline(time.Time{}))
	time.Sleep(5500 * time.Millisecond)
	second := exchange()
	assert.NotEqual(t, first, second, "the local UDP socket should change after hopping")
}

func TestHysteria2ShareDoesNotUseQueryEscapingForUserinfo(t *testing.T) {
	data, err := ConvertShareLinksToXrayJson("hy2://user%3Apassword%2B%25@example.com", "")
	require.NoError(t, err)
	text, err := ConvertXrayJsonToShareLinks(data)
	require.NoError(t, err)
	u, err := url.Parse(text)
	require.NoError(t, err)
	assert.Equal(t, "user:password+%", u.User.Username())
}
