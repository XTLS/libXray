package share

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

const validShareOutbound = `{"protocol":"vless","tag":"Keep","settings":{"address":"example.com","port":443,"id":"12345678-abcd-abcd-abcd-123456789abc","encryption":"none"},"streamSettings":{"security":"tls","tlsSettings":{"serverName":"example.com"}}}`

const legacyVMessQRCodeJSON = `{"v":"2","ps":"Legacy","add":"vm.example","port":"443","id":"` + testShareUUID + `","aid":"0","scy":"auto","net":"ws","path":"/ws","tls":"tls"}`

func TestConvertShareLinksSkipsInvalidCandidates(t *testing.T) {
	jsonText := `{"outbounds":[` + validShareOutbound + `,{"protocol":"freedom"},{"protocol":42},null,{"protocol":"vless","settings":{"id":"invalid"}}]}`
	for _, test := range []struct {
		name, text string
	}{
		{"links with headers", "Subscription export\n# comment\n\n" + ageTestShareLink + "\nvless://bad@example.com:443?encryption=none\nunknown://example.com"},
		{"links with removed protocols", "hy2://auth@hy.example:443\n" + ageTestShareLink + "\nhysteria2://auth@hy.example:443"},
		{"links with VMessQrCode", "vmess://" + base64.StdEncoding.EncodeToString([]byte(legacyVMessQRCodeJSON)) + "\n" + ageTestShareLink},
		{"JSON elements", jsonText},
		{"base64 JSON", base64.StdEncoding.EncodeToString([]byte(jsonText))},
		{"base64 links", base64.StdEncoding.EncodeToString([]byte(ageTestShareLink + "\nvless://bad@example.com:443"))},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := ConvertShareLinksToXrayJson(test.text, "")
			if err != nil {
				t.Fatal(err)
			}
			assertShareOutbounds(t, result, 1)
		})
	}
}

func TestConvertShareLinksRejectsRemovedFormats(t *testing.T) {
	for _, input := range []struct {
		name, text string
	}{
		{"Clash YAML", "proxies:\n  - {name: Node, type: vless, server: example.com, port: 443, uuid: " + testShareUUID + "}"},
		{"Mihomo JSON", `{"proxies":[{"name":"Node","type":"vless","server":"example.com","port":443,"uuid":"` + testShareUUID + `"}]}`},
		{"Hysteria2 URI", "hysteria2://auth@hy.example:443?sni=hy.example"},
		{"Hy2 URI", "hy2://auth@hy.example:443?sni=hy.example"},
		{"VMessQrCode", "vmess://" + base64.StdEncoding.EncodeToString([]byte(legacyVMessQRCodeJSON))},
		{"VMessQrCode URL-safe", "vmess://" + base64.RawURLEncoding.EncodeToString([]byte(legacyVMessQRCodeJSON))},
	} {
		t.Run(input.name, func(t *testing.T) {
			for _, encoding := range []struct {
				name, text string
			}{
				{"plaintext", input.text},
				{"base64", base64.StdEncoding.EncodeToString([]byte(input.text))},
				{"base64 URL", base64.RawURLEncoding.EncodeToString([]byte(input.text))},
			} {
				t.Run(encoding.name, func(t *testing.T) {
					result, err := ConvertShareLinksToXrayJson(encoding.text, "")
					if err == nil || result != nil {
						t.Fatalf("result = %s, error = %v", result, err)
					}
				})
			}
		})
	}
}

func TestConvertShareLinksAllInvalidReturnsNoData(t *testing.T) {
	for _, input := range []string{
		`{"outbounds":[{"protocol":"freedom"}]}`,
		"vless://secret-not-a-uuid@example.com:443?encryption=none",
	} {
		result, err := ConvertShareLinksToXrayJson(input, "")
		if err == nil || err.Error() != "no valid outbound found" {
			t.Fatalf("error = %v", err)
		}
		if result != nil {
			t.Fatalf("unexpected data: %s", result)
		}
	}
	result, err := ConvertShareLinksToXrayJson(`{"outbounds":[]}`, "")
	if err == nil {
		t.Fatal("empty array succeeded")
	}
	if result != nil {
		t.Fatalf("unexpected data: %s", result)
	}
}

func TestConvertShareLinksMalformedDocumentReturnsNoData(t *testing.T) {
	for _, input := range []string{
		`{"outbounds":[`, `{"outbounds":"private-source"}`, `{"outbounds":null}`,
		"proxies: [", "proxies: private-source", "not a subscription",
	} {
		result, err := ConvertShareLinksToXrayJson(input, "")
		if err == nil || result != nil {
			t.Fatalf("result = %+v, error = %v", result, err)
		}
		if strings.Contains(err.Error(), "private-source") {
			t.Fatal("error leaked source")
		}
	}
}

func TestConvertShareLinksAgeSkipsInvalidCandidatesAndRedactsErrors(t *testing.T) {
	pair, err := GenerateAgeKeyPair(AgeKeyTypeX25519)
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{
		ageTestShareLink + "\nvless://secret-not-a-uuid@example.com:443",
		`{"outbounds":[` + validShareOutbound + `,{"protocol":false}]}`,
	} {
		result, err := ConvertShareLinksToXrayJson(encryptAgeForTest(t, pair, input), pair.SecretKey)
		if err != nil {
			t.Fatal(err)
		}
		assertShareOutbounds(t, result, 1)
	}
	for _, input := range []string{
		"vless://secret-not-a-uuid@example.com:443",
		`{"outbounds":"private-source"}`,
		"proxies:\n  - {type: vless, server: example.com, port: 443, uuid: " + testShareUUID + "}",
		"hysteria2://private-password@hy.example:443",
		"hy2://private-password@hy.example:443",
		"vmess://" + base64.StdEncoding.EncodeToString([]byte(legacyVMessQRCodeJSON)),
	} {
		result, err := ConvertShareLinksToXrayJson(encryptAgeForTest(t, pair, input), pair.SecretKey)
		if err != ErrAgePlaintextUnsupported {
			t.Fatalf("error = %v", err)
		}
		if result != nil {
			t.Fatalf("unexpected data: %s", result)
		}
	}
	result, err := ConvertShareLinksToXrayJson(encryptAgeForTest(t, pair, ageTestShareLink), "private-invalid-key")
	if err != ErrAgeSecretKeyInvalid || result != nil {
		t.Fatalf("result = %+v, error = %v", result, err)
	}
}

func assertShareOutbounds(t *testing.T, result json.RawMessage, count int) {
	t.Helper()
	var config struct {
		Outbounds []json.RawMessage `json:"outbounds"`
	}
	if err := json.Unmarshal(result, &config); err != nil {
		t.Fatal(err)
	}
	if len(config.Outbounds) != count {
		t.Fatalf("outbounds = %d, want %d", len(config.Outbounds), count)
	}
}
