package libXray

import (
	"encoding/json"
	"path"
	"strings"

	"github.com/xtls/libxray/nodep"
	"github.com/xtls/xray-core/app/router"
	"github.com/xtls/xray-core/common/platform/filesystem"
	"google.golang.org/protobuf/proto"
)

type geoCode struct {
	Domain []string `json:"domain,omitempty"`
	Ip     []string `json:"ip,omitempty"`
}

// Read geo data and cut the codes we need.
// datDir means the dir which geosite.dat and geoip.dat are in.
// dstDir means the dir which new geosite.dat and new geoip.dat are in.
//
// This function is used to reduce memory when init instance.
// You can cut the country codes which rules and nameservers contain.
func CutGeoData(datDir string, dstDir string) string {
	geoCodePath := path.Join(dstDir, "geoCode.json")
	codeBytes, err := filesystem.ReadFile(geoCodePath)
	if err != nil {
		return err.Error()
	}

	var code geoCode

	if err := json.Unmarshal(codeBytes, &code); err != nil {
		return err.Error()
	}

	if err := cutGeoSite(datDir, code.Domain, dstDir); err != nil {
		return err.Error()
	}
	if err := cutGeoIP(datDir, code.Ip, dstDir); err != nil {
		return err.Error()
	}
	return ""
}

func cutGeoSite(datDir string, codes []string, dstDir string) error {
	datPath := path.Join(datDir, "geosite.dat")
	geositeBytes, err := filesystem.ReadFile(datPath)
	if err != nil {
		return err
	}
	var geositeList router.GeoSiteList
	if err := proto.Unmarshal(geositeBytes, &geositeList); err != nil {
		return err
	}

	var newEntry []*router.GeoSite
	for _, site := range geositeList.Entry {
		if containsCountryCode(codes, site.CountryCode) {
			newEntry = append(newEntry, site)
		}
	}
	var newGeositeList router.GeoSiteList
	newGeositeList.Entry = newEntry
	newDatBytes, err := proto.Marshal(&newGeositeList)
	if err != nil {
		return err
	}
	dstPath := path.Join(dstDir, "geosite.dat")
	if err := nodep.WriteBytes(newDatBytes, dstPath); err != nil {
		return err
	}

	return nil
}

func cutGeoIP(datDir string, codes []string, dstDir string) error {
	datPath := path.Join(datDir, "geoip.dat")
	geoipBytes, err := filesystem.ReadFile(datPath)
	if err != nil {
		return err
	}
	var geoipList router.GeoIPList
	if err := proto.Unmarshal(geoipBytes, &geoipList); err != nil {
		return err
	}

	var newEntry []*router.GeoIP
	for _, ip := range geoipList.Entry {
		if containsCountryCode(codes, ip.CountryCode) {
			newEntry = append(newEntry, ip)
		}
	}
	var newGeoipList router.GeoIPList
	newGeoipList.Entry = newEntry
	newDatBytes, err := proto.Marshal(&newGeoipList)
	if err != nil {
		return err
	}
	dstPath := path.Join(dstDir, "geoip.dat")
	if err := nodep.WriteBytes(newDatBytes, dstPath); err != nil {
		return err
	}

	return nil
}

func containsCountryCode(slice []string, element string) bool {
	for _, code := range slice {
		e := strings.ToUpper(code)
		if strings.Contains(e, "@") {
			codes := strings.Split(e, "@")
			if codes[0] == element {
				return true
			}
		} else {
			if e == element {
				return true
			}
		}
	}
	return false
}
