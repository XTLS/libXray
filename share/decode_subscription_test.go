package share

import (
	"errors"
	"strings"
	"testing"
)

const fullSubscriptionText = `[
  {
    "remarks": "ordinary",
    "inbounds": [{"tag":"socks","protocol":"socks","port":10808}],
    "outbounds": [{"tag":"proxy","protocol":"vless","settings":{"vnext":[{"address":"example.com","port":443,"users":[{"id":"provider-value"}]}]},"mux":{"enabled":true}}],
    "dns": {"servers":["https://dns.example/dns-query"]},
    "routing": {"rules":[{"inboundTag":["socks"],"outboundTag":"proxy"}]}
  },
  {
    "remarks": "virtual pool",
    "outbounds": [{"tag":"pool-1","protocol":"vless","streamSettings":{"sockopt":{"dialerProxy":"entry"}}},{"tag":"entry","protocol":"trojan"}],
    "routing": {"balancers":[{"tag":"pool","selector":["pool-"]}],"rules":[{"balancerTag":"pool"}]},
    "observatory": {"subjectSelector":["pool-"]},
    "futureOptions": {"preserve":true}
  }
]
`

func TestDecodeSubscriptionTextPreservesPlaintextWithoutParsing(t *testing.T) {
	for _, text := range []string{"", fullSubscriptionText, " \r\nnot a config\r\n", "dmxlc3M6Ly9leGFtcGxl"} {
		decoded, err := DecodeSubscriptionText(text, "unused-invalid-key")
		if err != nil || decoded != text {
			t.Fatalf("plaintext was changed: input length %d, output length %d, error %v", len(text), len(decoded), err)
		}
	}
}

func TestDecodeSubscriptionTextPreservesEncryptedFullConfig(t *testing.T) {
	for _, keyType := range []AgeKeyType{AgeKeyTypeX25519, AgeKeyTypeHybrid} {
		t.Run(string(keyType), func(t *testing.T) {
			pair, err := GenerateAgeKeyPair(keyType)
			if err != nil {
				t.Fatal(err)
			}
			armored := encryptAgeForTest(t, pair, fullSubscriptionText)
			decoded, err := DecodeSubscriptionText(" \r\n"+armored+" \r\n", pair.SecretKey)
			if err != nil || decoded != fullSubscriptionText {
				t.Fatalf("complete config was changed: output length %d, error %v", len(decoded), err)
			}
		})
	}
}

func TestDecodeSubscriptionTextRejectsAgeFailuresWithoutReturningText(t *testing.T) {
	pair, err := GenerateAgeKeyPair(AgeKeyTypeX25519)
	if err != nil {
		t.Fatal(err)
	}
	wrong, err := GenerateAgeKeyPair(AgeKeyTypeX25519)
	if err != nil {
		t.Fatal(err)
	}
	armored := encryptAgeForTest(t, pair, fullSubscriptionText)
	for _, test := range []struct {
		name, text, key string
		want            error
	}{
		{"missing", armored, "", ErrAgeSecretKeyMissing},
		{"invalid", armored, "invalid-key", ErrAgeSecretKeyInvalid},
		{"wrong", armored, wrong.SecretKey, ErrAgeDecryptFailed},
		{"armor", ageArmorHeader + "\ninvalid", pair.SecretKey, ErrAgeArmorMalformed},
	} {
		t.Run(test.name, func(t *testing.T) {
			decoded, err := DecodeSubscriptionText(test.text, test.key)
			if decoded != "" || !errors.Is(err, test.want) {
				t.Fatalf("output length %d, error %v, want %v", len(decoded), err, test.want)
			}
			if test.key != "" && strings.Contains(err.Error(), test.key) {
				t.Fatal("error exposed the secret key")
			}
		})
	}
}

func TestDecodeSubscriptionTextLimitsUTF8PlaintextBytes(t *testing.T) {
	// The last ASCII byte fills the remainder after the three-byte characters.
	text := strings.Repeat("界", maxAgePlaintextBytes/3) + "a"
	if len(text) != maxAgePlaintextBytes {
		t.Fatal("fixture must fill the exact byte limit")
	}
	decoded, err := DecodeSubscriptionText(text, "")
	if err != nil || decoded != text {
		t.Fatalf("exact byte limit was rejected: %v", err)
	}
	decoded, err = DecodeSubscriptionText(text+"b", "")
	if decoded != "" || !errors.Is(err, ErrAgePlaintextTooLarge) {
		t.Fatalf("oversize plaintext output length %d, error %v", len(decoded), err)
	}
}

func TestDecodeSubscriptionTextLimitsDecryptedBytes(t *testing.T) {
	pair, err := GenerateAgeKeyPair(AgeKeyTypeX25519)
	if err != nil {
		t.Fatal(err)
	}
	for _, size := range []int{maxAgePlaintextBytes, maxAgePlaintextBytes + 1} {
		text := strings.Repeat("x", size)
		decoded, err := DecodeSubscriptionText(encryptAgeForTest(t, pair, text), pair.SecretKey)
		if size == maxAgePlaintextBytes {
			if err != nil || decoded != text {
				t.Fatalf("exact decrypted byte limit was rejected: %v", err)
			}
		} else if decoded != "" || !errors.Is(err, ErrAgePlaintextTooLarge) {
			t.Fatalf("oversize decrypted output length %d, error %v", len(decoded), err)
		}
	}
}
