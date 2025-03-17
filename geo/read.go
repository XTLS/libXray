package geo

// Read all geo files in config file.
// configPath means where xray config file is.
func ReadGeoFiles(xrayBytes []byte) ([]string, []string) {
	domain, ip := loadXrayConfig(xrayBytes)
	domainCodes := filterAndStrip(domain, "geosite")
	domainFiles := []string{}
	for key := range domainCodes {
		domainFiles = append(domainFiles, key)
	}

	ipCodes := filterAndStrip(ip, "geoip")
	ipFiles := []string{}
	for key := range ipCodes {
		ipFiles = append(ipFiles, key)
	}

	return domainFiles, ipFiles
}
