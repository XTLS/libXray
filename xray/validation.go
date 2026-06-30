package xray

// Test Xray Config.
// configPath means the config.json file path.
func TestXray(configPath string) error {
	server, err := StartXray(configPath)
	if err != nil {
		return err
	}
	err = server.Close()
	if err != nil {
		return err
	}
	return nil
}
