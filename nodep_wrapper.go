package libXray

import (
	"github.com/xtls/libxray/nodep"
)

// Wrapper of nodep.GetFreePorts
// count means how many ports you need.
// return ports divided by ":", like "1080:1081"
func GetFreePorts(count int) string {
	return nodep.GetFreePorts(count)
}

// Convert share text to XrayJson
// support XrayJson, v2rayN plain text, v2rayN base64 text, Clash yaml, Clash.Meta yaml
func ConvertShareTextToXrayJson(textPath string, xrayPath string) string {
	return nodep.ConvertShareTextToXrayJson(textPath, xrayPath)
}

// Convert XrayJson to share links.
// VMess will generate VMessAEAD link.
func ConvertXrayJsonToShareText(xrayPath string, textPath string) string {
	return nodep.ConvertXrayJsonToShareText(xrayPath, textPath)
}
