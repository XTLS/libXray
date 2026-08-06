package share

import (
	"bytes"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/metacubex/age"
	"github.com/metacubex/age/armor"
)

const ageTestShareLink = "vless://12345678-abcd-abcd-abcd-123456789abc@example.com:443?encryption=none&security=tls&sni=example.com#AgeTest"

func TestGenerateAgeKeyPair(t *testing.T) {
	for _, keyType := range []AgeKeyType{AgeKeyTypeX25519, AgeKeyTypeHybrid} {
		t.Run(string(keyType), func(t *testing.T) {
			first, err := GenerateAgeKeyPair(keyType)
			if err != nil {
				t.Fatal(err)
			}
			second, err := GenerateAgeKeyPair(keyType)
			if err != nil {
				t.Fatal(err)
			}
			if first.SecretKey == second.SecretKey || first.PublicKey == second.PublicKey {
				t.Fatal("generated age keypairs should be unique")
			}
			assertAgeRoundTrip(t, first, ageTestShareLink)
		})
	}
}

func TestGenerateAgeKeyPairRejectsUnsupportedType(t *testing.T) {
	_, err := GenerateAgeKeyPair(AgeKeyType("unsupported"))
	if !errors.Is(err, ErrAgeKeyTypeUnsupported) {
		t.Fatalf("error = %v, want %v", err, ErrAgeKeyTypeUnsupported)
	}
}

func TestConvertShareLinksToXrayJsonWithAgePlaintext(t *testing.T) {
	pair, err := GenerateAgeKeyPair(AgeKeyTypeX25519)
	if err != nil {
		t.Fatal(err)
	}
	config, err := ConvertShareLinksToXrayJsonWithAge(ageTestShareLink, pair.SecretKey)
	if err != nil {
		t.Fatal(err)
	}
	if len(config.OutboundConfigs) != 1 {
		t.Fatalf("outbounds = %d, want 1", len(config.OutboundConfigs))
	}
}

func TestConvertShareLinksToXrayJsonWithAgeEncrypted(t *testing.T) {
	for _, keyType := range []AgeKeyType{AgeKeyTypeX25519, AgeKeyTypeHybrid} {
		t.Run(string(keyType), func(t *testing.T) {
			pair, err := GenerateAgeKeyPair(keyType)
			if err != nil {
				t.Fatal(err)
			}
			armored := encryptAgeForTest(t, pair, ageTestShareLink)
			config, err := ConvertShareLinksToXrayJsonWithAge(armored, pair.SecretKey)
			if err != nil {
				t.Fatal(err)
			}
			if len(config.OutboundConfigs) != 1 {
				t.Fatalf("outbounds = %d, want 1", len(config.OutboundConfigs))
			}
		})
	}
}

func TestConvertShareLinksToXrayJsonWithAgeErrors(t *testing.T) {
	pair, err := GenerateAgeKeyPair(AgeKeyTypeX25519)
	if err != nil {
		t.Fatal(err)
	}
	armored := encryptAgeForTest(t, pair, ageTestShareLink)

	_, err = ConvertShareLinksToXrayJsonWithAge(armored, "")
	if !errors.Is(err, ErrAgeSecretKeyMissing) {
		t.Fatalf("missing key error = %v", err)
	}

	wrongPair, err := GenerateAgeKeyPair(AgeKeyTypeX25519)
	if err != nil {
		t.Fatal(err)
	}
	_, err = ConvertShareLinksToXrayJsonWithAge(armored, wrongPair.SecretKey)
	if !errors.Is(err, ErrAgeDecryptFailed) {
		t.Fatalf("wrong key error = %v", err)
	}
	if strings.Contains(err.Error(), wrongPair.SecretKey) {
		t.Fatal("decryption error contains the secret key")
	}

	_, err = ConvertShareLinksToXrayJsonWithAge(ageArmorHeader+"\ninvalid", pair.SecretKey)
	if !errors.Is(err, ErrAgeArmorMalformed) {
		t.Fatalf("malformed armor error = %v", err)
	}
}

func TestConvertShareLinksToXrayJsonWithAgeRejectsLargePlaintext(t *testing.T) {
	pair, err := GenerateAgeKeyPair(AgeKeyTypeX25519)
	if err != nil {
		t.Fatal(err)
	}
	armored := encryptAgeForTest(
		t,
		pair,
		strings.Repeat("x", maxAgePlaintextBytes+1),
	)
	_, err = ConvertShareLinksToXrayJsonWithAge(armored, pair.SecretKey)
	if !errors.Is(err, ErrAgePlaintextTooLarge) {
		t.Fatalf("large plaintext error = %v", err)
	}
}

func TestConvertShareLinksToXrayJsonWithAgeSanitizesParserErrors(t *testing.T) {
	pair, err := GenerateAgeKeyPair(AgeKeyTypeX25519)
	if err != nil {
		t.Fatal(err)
	}
	sensitivePlaintext := "unsupported://user:password@example.com"
	armored := encryptAgeForTest(t, pair, sensitivePlaintext)
	_, err = ConvertShareLinksToXrayJsonWithAge(armored, pair.SecretKey)
	if !errors.Is(err, ErrAgePlaintextUnsupported) {
		t.Fatalf("unsupported plaintext error = %v", err)
	}
	if strings.Contains(err.Error(), sensitivePlaintext) || strings.Contains(err.Error(), pair.SecretKey) {
		t.Fatal("age parser error contains sensitive input")
	}
}

func assertAgeRoundTrip(t *testing.T, pair *AgeKeyPair, plaintext string) {
	t.Helper()
	armored := encryptAgeForTest(t, pair, plaintext)
	identity, _, err := parseNativeAgeIdentity(pair.SecretKey)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := age.Decrypt(armor.NewReader(strings.NewReader(armored)), identity)
	if err != nil {
		t.Fatal(err)
	}
	decrypted, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if string(decrypted) != plaintext {
		t.Fatalf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func encryptAgeForTest(t *testing.T, pair *AgeKeyPair, plaintext string) string {
	t.Helper()
	_, recipient, err := parseNativeAgeIdentity(pair.SecretKey)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	armored := armor.NewWriter(&output)
	writer, err := age.Encrypt(armored, recipient)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(writer, plaintext); err != nil {
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
