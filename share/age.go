package share

import (
	"errors"
	"io"
	"strings"

	"github.com/metacubex/age"
	"github.com/metacubex/age/armor"
	"github.com/xtls/xray-core/infra/conf"
)

const (
	ageArmorHeader       = "-----BEGIN AGE ENCRYPTED FILE-----"
	maxAgePlaintextBytes = 16 * 1024 * 1024
)

var (
	ErrAgeSecretKeyMissing     = errors.New("missing age secret key")
	ErrAgeSecretKeyInvalid     = errors.New("invalid or unsupported age secret key")
	ErrAgeDecryptFailed        = errors.New("unable to decrypt age subscription")
	ErrAgeArmorMalformed       = errors.New("malformed age armor")
	ErrAgePlaintextTooLarge    = errors.New("decrypted subscription exceeds the 16 MiB size limit")
	ErrAgePlaintextUnsupported = errors.New("decrypted subscription is unsupported")
	ErrAgeKeyTypeUnsupported   = errors.New("unsupported age key type")
)

type AgeKeyType string

const (
	AgeKeyTypeX25519 AgeKeyType = "x25519"
	AgeKeyTypeHybrid AgeKeyType = "hybrid"
)

type AgeKeyPair struct {
	SecretKey string
	PublicKey string
}

func GenerateAgeKeyPair(keyType AgeKeyType) (*AgeKeyPair, error) {
	switch keyType {
	case "", AgeKeyTypeX25519:
		identity, err := age.GenerateX25519Identity()
		if err != nil {
			return nil, errors.New("failed to generate age keypair")
		}
		return &AgeKeyPair{
			SecretKey: identity.String(),
			PublicKey: identity.Recipient().String(),
		}, nil
	case AgeKeyTypeHybrid:
		identity, err := age.GenerateHybridIdentity()
		if err != nil {
			return nil, errors.New("failed to generate age keypair")
		}
		return &AgeKeyPair{
			SecretKey: identity.String(),
			PublicKey: identity.Recipient().String(),
		}, nil
	default:
		return nil, ErrAgeKeyTypeUnsupported
	}
}

func ConvertShareLinksToXrayJsonWithAge(links, secretKey string) (*conf.Config, error) {
	text := strings.TrimSpace(FixWindowsReturn(links))
	if !strings.HasPrefix(text, ageArmorHeader) {
		return ConvertShareLinksToXrayJson(links)
	}
	if strings.TrimSpace(secretKey) == "" {
		return nil, ErrAgeSecretKeyMissing
	}

	identity, _, err := parseNativeAgeIdentity(secretKey)
	if err != nil {
		return nil, err
	}
	reader, err := age.Decrypt(armor.NewReader(strings.NewReader(text)), identity)
	if err != nil {
		var noMatch *age.NoIdentityMatchError
		if errors.As(err, &noMatch) {
			return nil, ErrAgeDecryptFailed
		}
		return nil, ErrAgeArmorMalformed
	}

	plaintext, err := io.ReadAll(io.LimitReader(reader, maxAgePlaintextBytes+1))
	if err != nil {
		return nil, ErrAgeArmorMalformed
	}
	if len(plaintext) > maxAgePlaintextBytes {
		return nil, ErrAgePlaintextTooLarge
	}
	config, err := ConvertShareLinksToXrayJson(string(plaintext))
	if err != nil {
		return nil, ErrAgePlaintextUnsupported
	}
	return config, nil
}

func parseNativeAgeIdentity(secretKey string) (age.Identity, age.Recipient, error) {
	key := strings.TrimSpace(secretKey)
	if key == "" || strings.ContainsAny(key, "\r\n") {
		return nil, nil, ErrAgeSecretKeyInvalid
	}

	if strings.HasPrefix(key, "AGE-SECRET-KEY-1") {
		identity, err := age.ParseX25519Identity(key)
		if err != nil {
			return nil, nil, ErrAgeSecretKeyInvalid
		}
		return identity, identity.Recipient(), nil
	}
	if strings.HasPrefix(key, "AGE-SECRET-KEY-PQ-1") {
		identity, err := age.ParseHybridIdentity(key)
		if err != nil {
			return nil, nil, ErrAgeSecretKeyInvalid
		}
		return identity, identity.Recipient(), nil
	}
	return nil, nil, ErrAgeSecretKeyInvalid
}
