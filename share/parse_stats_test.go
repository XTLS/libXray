package share

import (
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
)

const statsValidOutbound = `{"protocol":"vless","tag":"Keep","settings":{"address":"example.com","port":443,"id":"12345678-abcd-abcd-abcd-123456789abc","encryption":"none"},"streamSettings":{"security":"tls","tlsSettings":{"serverName":"example.com"}}}`

func TestShareStatsCountActualCandidates(t *testing.T) {
	jsonText := `{"outbounds":[` + statsValidOutbound + `,{"protocol":"freedom"},{"protocol":42},null,{"protocol":"vless","settings":{"id":"invalid"}}]}`
	yamlText := `proxies:
  - {name: Keep, type: vless, server: example.com, port: 443, uuid: 12345678-abcd-abcd-abcd-123456789abc, tls: true, servername: example.com}
  - {name: Unsupported, type: unknown}
  - {name: InvalidPort, type: vless, port: invalid}
  - {name: InvalidId, type: vless, server: example.com, port: 443, uuid: invalid}
  - null
`
	for _, test := range []struct {
		name, text     string
		usable, failed int
	}{
		{"links with headers", "Subscription export\n# comment\n\n" + ageTestShareLink + "\nvless://bad@example.com:443?encryption=none\nunknown://example.com", 1, 2},
		{"JSON elements", jsonText, 1, 4},
		{"YAML elements", yamlText, 1, 4},
		{"base64 JSON", base64.StdEncoding.EncodeToString([]byte(jsonText)), 1, 4},
		{"base64 YAML", base64.RawURLEncoding.EncodeToString([]byte(yamlText)), 1, 4},
		{"base64 links", base64.StdEncoding.EncodeToString([]byte(ageTestShareLink + "\nvless://bad@example.com:443")), 1, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			result, err := ConvertShareLinksToXrayJsonWithStats(test.text, "")
			if err != nil {
				t.Fatal(err)
			}
			assertShareStats(t, result, test.usable, test.failed)
		})
	}
}

func TestShareStatsAllInvalidRetainsCountsWithoutSource(t *testing.T) {
	for _, input := range []string{
		`{"outbounds":[{"protocol":"freedom"}]}`,
		"vless://secret-not-a-uuid@example.com:443?encryption=none",
		"proxies:\n - {type: unsupported, password: private-password}",
	} {
		result, err := ConvertShareLinksToXrayJsonWithStats(input, "")
		if err == nil || err.Error() != "no valid outbound found" {
			t.Fatalf("error = %v", err)
		}
		assertShareStats(t, result, 0, 1)
	}
	result, err := ConvertShareLinksToXrayJsonWithStats(`{"outbounds":[]}`, "")
	if err == nil {
		t.Fatal("empty array succeeded")
	}
	assertShareStats(t, result, 0, 0)
}

func TestShareStatsMalformedDocumentHasNoCounts(t *testing.T) {
	for _, input := range []string{
		`{"outbounds":[`, `{"outbounds":"private-source"}`, `{"outbounds":null}`,
		"proxies: [", "proxies: private-source", "not a subscription",
	} {
		result, err := ConvertShareLinksToXrayJsonWithStats(input, "")
		if err == nil || result != nil {
			t.Fatalf("result = %+v, error = %v", result, err)
		}
		if strings.Contains(err.Error(), "private-source") {
			t.Fatal("error leaked source")
		}
	}
}

func TestShareStatsAgeCountsInnerCandidatesAndRedactsErrors(t *testing.T) {
	pair, err := GenerateAgeKeyPair(AgeKeyTypeX25519)
	if err != nil {
		t.Fatal(err)
	}
	for _, input := range []string{
		ageTestShareLink + "\nvless://secret-not-a-uuid@example.com:443",
		`{"outbounds":[` + statsValidOutbound + `,{"protocol":false}]}`,
	} {
		result, err := ConvertShareLinksToXrayJsonWithStats(encryptAgeForTest(t, pair, input), pair.SecretKey)
		if err != nil {
			t.Fatal(err)
		}
		assertShareStats(t, result, 1, 1)
	}
	for _, test := range []struct {
		text      string
		hasCounts bool
	}{
		{"vless://secret-not-a-uuid@example.com:443", true},
		{`{"outbounds":"private-source"}`, false},
	} {
		result, err := ConvertShareLinksToXrayJsonWithStats(encryptAgeForTest(t, pair, test.text), pair.SecretKey)
		if err != ErrAgePlaintextUnsupported {
			t.Fatalf("error = %v", err)
		}
		if (result != nil) != test.hasCounts {
			t.Fatalf("counts = %+v", result)
		}
		if result != nil {
			assertShareStats(t, result, 0, 1)
		}
	}
	result, err := ConvertShareLinksToXrayJsonWithStats(encryptAgeForTest(t, pair, ageTestShareLink), "private-invalid-key")
	if err != ErrAgeSecretKeyInvalid || result != nil {
		t.Fatalf("result = %+v, error = %v", result, err)
	}
}

func TestShareStatsPreservesLegacyProjectedConfig(t *testing.T) {
	config, err := ConvertShareLinksToXrayJson(ageTestShareLink)
	if err != nil {
		t.Fatal(err)
	}
	legacy, err := MarshalShareConfigJSON(config)
	if err != nil {
		t.Fatal(err)
	}
	result, err := ConvertShareLinksToXrayJsonWithStats(ageTestShareLink, "")
	if err != nil {
		t.Fatal(err)
	}
	if string(result.Config) != string(legacy) {
		t.Fatal("stats changed projected config")
	}
}

func assertShareStats(t *testing.T, result *ParseStats, usable, failed int) {
	t.Helper()
	if result == nil || result.UsableCount != usable || result.FailedCount != failed {
		t.Fatalf("stats = %+v, want usable %d failed %d", result, usable, failed)
	}
	var config struct {
		Outbounds []json.RawMessage `json:"outbounds"`
	}
	if err := json.Unmarshal(result.Config, &config); err != nil {
		t.Fatal(err)
	}
	if config.Outbounds == nil || len(config.Outbounds) != usable {
		t.Fatalf("config count = %d, want %d", len(config.Outbounds), usable)
	}
}
