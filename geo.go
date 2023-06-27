package libXray

import (
	"fmt"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/xtls/libxray/nodep"
	"github.com/xtls/xray-core/app/router"
	"github.com/xtls/xray-core/common/platform/filesystem"
	"google.golang.org/protobuf/proto"
)

// Read geo data and write all codes to text file.
// dir means the dir which geosite.dat and geoip.dat are in.
func LoadGeoData(dir string) string {
	if err := loadGeoSite(dir); err != nil {
		return err.Error()
	}
	if err := loadGeoIP(dir); err != nil {
		return err.Error()
	}
	ts := time.Now().Unix()
	tsText := strconv.FormatInt(ts, 10)
	tsPath := path.Join(dir, "timestamp.txt")
	if err := nodep.WriteText(tsText, tsPath); err != nil {
		return err.Error()
	}
	return ""
}

func loadGeoSite(dir string) error {
	datPath := path.Join(dir, "geosite.dat")
	txtPath := path.Join(dir, "geosite.txt")
	geositeBytes, err := filesystem.ReadFile(datPath)
	if err != nil {
		return err
	}
	var geositeList router.GeoSiteList
	if err := proto.Unmarshal(geositeBytes, &geositeList); err != nil {
		return err
	}

	var countries []string
	for _, site := range geositeList.Entry {
		countries = append(countries, site.CountryCode)
		for _, domain := range site.Domain {
			for _, attribute := range domain.Attribute {
				attr := fmt.Sprintf("%s@%s", site.CountryCode, attribute.Key)
				if !containsString(countries, attr) {
					countries = append(countries, attr)
				}
			}
		}
	}
	text := strings.Join(countries, "\n")
	if err := nodep.WriteText(text, txtPath); err != nil {
		return err
	}

	return nil
}

func loadGeoIP(dir string) error {
	datPath := path.Join(dir, "geoip.dat")
	txtPath := path.Join(dir, "geoip.txt")
	geoipBytes, err := filesystem.ReadFile(datPath)
	if err != nil {
		return err
	}
	var geoipList router.GeoIPList
	if err := proto.Unmarshal(geoipBytes, &geoipList); err != nil {
		return err
	}

	var countries []string
	for _, geoip := range geoipList.Entry {
		countries = append(countries, geoip.CountryCode)
	}

	text := strings.Join(countries, "\n")
	if err := nodep.WriteText(text, txtPath); err != nil {
		return err
	}

	return nil
}

func containsString(slice []string, element string) bool {
	for _, e := range slice {
		if e == element {
			return true
		}
	}
	return false
}
