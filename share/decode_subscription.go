package share

// DecodeSubscriptionText decrypts Age armor or returns the original plaintext.
// It does not parse or validate the subscription's format or Xray configuration.
func DecodeSubscriptionText(text, secretKey string) (string, error) {
	plaintext, _, err := decryptShareText(text, secretKey)
	if err != nil {
		return "", err
	}
	if len(plaintext) > maxAgePlaintextBytes {
		return "", ErrAgePlaintextTooLarge
	}
	return plaintext, nil
}
