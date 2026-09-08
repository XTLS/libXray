package share

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

const validShareOutbound = `{"protocol":"vless","tag":"Keep","settings":{"address":"example.com","port":443,"id":"12345678-abcd-abcd-abcd-123456789abc","encryption":"none"},"streamSettings":{"security":"tls","tlsSettings":{"serverName":"example.com"}}}`

func TestConvertShareLinksSkipsInvalidCandidates(t *testing.T) {
	jsonText := `{"outbounds":[` + validShareOutbound + `,{"protocol":"freedom"},{"protocol":42},null,{"protocol":"vless","settings":{"id":"invalid"}}]}`
	yamlText := `proxies:
  - {name: Keep, type: vless, server: example.com, port: 443, uuid: 12345678-abcd-abcd-abcd-123456789abc, tls: true, servername: example.com}
  - {name: Unsupported, type: unknown}
  - {name: InvalidPort, type: vless, port: invalid}
  - {name: InvalidId, type: vless, server: example.com, port: 443, uuid: invalid}
  - null
`
	for _, test := range []struct {
		name, text string
	}{
		{"links with headers", "Subscription export\n# comment\n\n" + ageTestShareLink + "\nvless://bad@example.com:443?encryption=none\nunknown://example.com"},
		{"JSON elements", jsonText},
		{"YAML elements", yamlText},
		{"base64 JSON", base64.StdEncoding.EncodeToString([]byte(jsonText))},
		{"base64 YAML", base64.RawURLEncoding.EncodeToString([]byte(yamlText))},
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

func TestConvertShareLinksAllInvalidReturnsNoData(t *testing.T) {
	for _, input := range []string{
		`{"outbounds":[{"protocol":"freedom"}]}`,
		"vless://secret-not-a-uuid@example.com:443?encryption=none",
		"proxies:\n - {type: unsupported, password: private-password}",
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
