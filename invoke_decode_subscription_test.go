package libXray

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/metacubex/age"
	"github.com/metacubex/age/armor"
	"github.com/xtls/libxray/share"
)

func TestInvokeDecodeSubscriptionTextPreservesWholeDocument(t *testing.T) {
	const config = `{"remarks":"完整配置","outbounds":[{"tag":"proxy","protocol":"future-protocol","mux":{"enabled":true}}],"dns":{"servers":["fakedns"]},"routing":{"balancers":[{"tag":"pool","selector":["proxy"]}]},"observatory":{"subjectSelector":["proxy"]}}`
	for _, text := range []string{"", config, "[\n" + config + ",\n" + config + "\n]\n", "not JSON"} {
		response := invokeForTest(t, LibXrayMethodDecodeSubscriptionText, DecodeSubscriptionTextRequest{Text: text})
		if !response.Success || response.Err != "" {
			t.Fatalf("decode failed: %s", response.Err)
		}
		var data map[string]json.RawMessage
		if err := json.Unmarshal(response.Data, &data); err != nil {
			t.Fatal(err)
		}
		if len(data) != 1 || data["text"] == nil {
			t.Fatal("decode response must contain only text, including empty text")
		}
		if decoded := decodeDataObject[DecodeSubscriptionTextResponse](t, response); decoded.Text != text {
			t.Fatal("decode response projected or changed the source")
		}
	}
}

func TestInvokeDecodeSubscriptionTextDecryptsWithoutProjection(t *testing.T) {
	pair, err := share.GenerateAgeKeyPair(share.AgeKeyTypeX25519)
	if err != nil {
		t.Fatal(err)
	}
	const text = `[{"outbounds":[{"protocol":"unknown","streamSettings":{"sockopt":{"dialerProxy":"entry"}}}],"routing":{"rules":[]},"dns":{"hosts":{"example.com":"203.0.113.1"}}}]`
	response := invokeForTest(t, LibXrayMethodDecodeSubscriptionText, DecodeSubscriptionTextRequest{
		Text: encryptDecodeTextForTest(t, pair.PublicKey, text),
		Age:  &AgeDecryptConfig{SecretKey: pair.SecretKey},
	})
	if !response.Success || decodeDataObject[DecodeSubscriptionTextResponse](t, response).Text != text {
		t.Fatalf("full subscription decode failed: %s", response.Err)
	}
	for _, payload := range []any{
		DecodeSubscriptionTextRequest{Text: "-----BEGIN AGE ENCRYPTED FILE-----\ninvalid"},
		map[string]any{"text": []string{"invalid field type"}},
	} {
		failed := invokeForTest(t, LibXrayMethodDecodeSubscriptionText, payload)
		if failed.Success || string(failed.Data) != "null" || failed.Err == "" {
			t.Fatal("failed decoding must return null data and an error")
		}
	}
}

func TestInvokeDecodeSubscriptionTextKeepsEnvelopeLimits(t *testing.T) {
	pair, err := share.GenerateAgeKeyPair(share.AgeKeyTypeX25519)
	if err != nil {
		t.Fatal(err)
	}
	// The armor fits the request limit; JSON's HTML escaping expands the decoded
	// response beyond its envelope limit without exceeding the plaintext limit.
	text := strings.Repeat("<", 3*1024*1024)
	response := invokeForTest(t, LibXrayMethodDecodeSubscriptionText, DecodeSubscriptionTextRequest{
		Text: encryptDecodeTextForTest(t, pair.PublicKey, text),
		Age:  &AgeDecryptConfig{SecretKey: pair.SecretKey},
	})
	if response.Success || string(response.Data) != "null" || response.Err != invokeJSONSizeLimitMessage("response") {
		t.Fatalf("oversize decoded envelope error = %s", response.Err)
	}
}

func encryptDecodeTextForTest(t *testing.T, publicKey, text string) string {
	t.Helper()
	recipient, err := age.ParseX25519Recipient(publicKey)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	armored := armor.NewWriter(&output)
	writer, err := age.Encrypt(armored, recipient)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(writer, text); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := armored.Close(); err != nil {
		t.Fatal(err)
	}
	return output.String()
}
