package libXray

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/xtls/libxray/nodep"
	"github.com/xtls/xray-core/app/router"
	"google.golang.org/protobuf/proto"
)

type geoCutCode struct {
	Dat []geoCutCodeDat `json:"dat,omitempty"`
}

// Type must be the value of geoType
type geoCutCodeDat struct {
	Name   string   `json:"name,omitempty"`
	Type   string   `json:"type,omitempty"`
	UrlMd5 string   `json:"urlMd5,omitempty"`
	Codes  []string `json:"codes,omitempty"`
}

// Read geo data and cut the codes we need.
// datDir means the dir which geo dat are in.
// dstDir means the dir which new geo dat are in.
// cutCodePath means geoCutCode json file path
//
// This function is used to reduce memory when init instance.
// You can cut the country codes which rules and nameservers contain.
func CutGeoData(datDir string, dstDir string, cutCodePath string) string {
	codeBytes, err := os.ReadFile(cutCodePath)
	if err != nil {
		return err.Error()
	}

	var code geoCutCode

	if err := json.Unmarshal(codeBytes, &code); err != nil {
		return err.Error()
	}

	for _, dat := range code.Dat {
		switch dat.Type {
		case geoTypeDomain:
			if err := cutGeoSite(datDir, dstDir, dat); err != nil {
				return err.Error()
			}

		case geoTypeIP:
			if err := cutGeoIP(datDir, dstDir, dat); err != nil {
				return err.Error()
			}
		default:
			return fmt.Errorf("wrong geoType: %s", dat.Type).Error()
		}
	}

	return ""
}

func cutGeoSite(datDir string, dstDir string, dat geoCutCodeDat) error {
	srcName := dat.UrlMd5 + ".dat"
	dstName := dat.Name + ".dat"
	srcPath := path.Join(datDir, srcName)
	dstPath := path.Join(dstDir, dstName)
	geositeBytes, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	var geositeList router.GeoSiteList
	if err := proto.Unmarshal(geositeBytes, &geositeList); err != nil {
		return err
	}

	var newEntry []*router.GeoSite
	for _, site := range geositeList.Entry {
		if containsCountryCode(dat.Codes, site.CountryCode) {
			newEntry = append(newEntry, site)
		}
	}
	var newGeositeList router.GeoSiteList
	newGeositeList.Entry = newEntry
	newDatBytes, err := proto.Marshal(&newGeositeList)
	if err != nil {
		return err
	}
	if err := nodep.WriteBytes(newDatBytes, dstPath); err != nil {
		return err
	}

	return nil
}

func cutGeoIP(datDir string, dstDir string, dat geoCutCodeDat) error {
	srcName := dat.UrlMd5 + ".dat"
	dstName := dat.Name + ".dat"
	srcPath := path.Join(datDir, srcName)
	dstPath := path.Join(dstDir, dstName)
	geoipBytes, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}
	var geoipList router.GeoIPList
	if err := proto.Unmarshal(geoipBytes, &geoipList); err != nil {
		return err
	}

	var newEntry []*router.GeoIP
	for _, ip := range geoipList.Entry {
		if containsCountryCode(dat.Codes, ip.CountryCode) {
			newEntry = append(newEntry, ip)
		}
	}
	var newGeoipList router.GeoIPList
	newGeoipList.Entry = newEntry
	newDatBytes, err := proto.Marshal(&newGeoipList)
	if err != nil {
		return err
	}
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
