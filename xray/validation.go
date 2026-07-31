package xray

// Test Xray Config.
// xrayJSON is the serialized Xray JSON configuration.
func TestXray(xrayJSON string) error {
	server, err := newXrayInstance(xrayJSON)
	if err != nil {
		return err
	}
	err = server.Close()
	if err != nil {
		return err
	}
	return nil
}
