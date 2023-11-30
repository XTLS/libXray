package xray

// Test Xray Config.
// datDir means the dir which geosite.dat and geoip.dat are in.
// configPath means the config.json file path.
func TestXray(datDir string, configPath string) string {
	InitEnv(datDir)
	_, err := StartXray(configPath)
	if err != nil {
		return err.Error()
	}
	return ""
}
