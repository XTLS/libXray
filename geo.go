package libXray

import (
	"encoding/json"
	"fmt"
	"path"
	"sort"
	"strconv"
	"time"

	"github.com/xtls/libxray/nodep"
	"github.com/xtls/xray-core/app/router"
	"github.com/xtls/xray-core/common/platform/filesystem"
	"google.golang.org/protobuf/proto"
)

type geoList struct {
	Codes []*geoCountryCode `json:"codes,omitempty"`
}
type geoCountryCode struct {
	Code  string `json:"code,omitempty"`
	Count int    `json:"count,omitempty"`
}

// Read geo data and write all codes to text file.
// datDir means the dir which geosite.dat and geoip.dat are in.
func LoadGeoData(datDir string) string {
	if err := loadGeoSite(datDir); err != nil {
		return err.Error()
	}
	if err := loadGeoIP(datDir); err != nil {
		return err.Error()
	}
	ts := time.Now().Unix()
	tsText := strconv.FormatInt(ts, 10)
	tsPath := path.Join(datDir, "timestamp.txt")
	if err := nodep.WriteText(tsText, tsPath); err != nil {
		return err.Error()
	}
	return ""
}

func loadGeoSite(datDir string) error {
	datPath := path.Join(datDir, "geosite.dat")
	jsonPath := path.Join(datDir, "geosite.json")
	geositeBytes, err := filesystem.ReadFile(datPath)
	if err != nil {
		return err
	}
	var geositeList router.GeoSiteList
	if err := proto.Unmarshal(geositeBytes, &geositeList); err != nil {
		return err
	}

	var codes []*geoCountryCode
	for _, site := range geositeList.Entry {
		var siteCode geoCountryCode
		siteCode.Code = site.CountryCode
		siteCode.Count = len(site.Domain)
		codes = append(codes, &siteCode)
		for _, domain := range site.Domain {
			for _, attribute := range domain.Attribute {
				attr := fmt.Sprintf("%s@%s", site.CountryCode, attribute.Key)
				attrCode := findAttrCode(codes, attr)
				if attrCode == nil {
					var newCode geoCountryCode
					newCode.Code = attr
					newCode.Count = 1
					codes = append(codes, &newCode)
				} else {
					attrCode.Count += 1
				}
			}
		}
	}

	sortCodes(codes)
	var list geoList
	list.Codes = codes
	jsonBytes, err := json.Marshal(&list)
	if err != nil {
		return err
	}
	if err := nodep.WriteBytes(jsonBytes, jsonPath); err != nil {
		return err
	}

	return nil
}

func loadGeoIP(datDir string) error {
	datPath := path.Join(datDir, "geoip.dat")
	jsonPath := path.Join(datDir, "geoip.json")
	geoipBytes, err := filesystem.ReadFile(datPath)
	if err != nil {
		return err
	}
	var geoipList router.GeoIPList
	if err := proto.Unmarshal(geoipBytes, &geoipList); err != nil {
		return err
	}

	var codes []*geoCountryCode
	for _, geoip := range geoipList.Entry {
		var code geoCountryCode
		code.Code = geoip.CountryCode
		code.Count = len(geoip.Cidr)
		codes = append(codes, &code)
	}

	sortCodes(codes)
	var list geoList
	list.Codes = codes
	jsonBytes, err := json.Marshal(&list)
	if err != nil {
		return err
	}
	if err := nodep.WriteBytes(jsonBytes, jsonPath); err != nil {
		return err
	}

	return nil
}

func findAttrCode(codes []*geoCountryCode, attrCode string) *geoCountryCode {
	for _, code := range codes {
		if code.Code == attrCode {
			return code
		}
	}
	return nil
}

func sortCodes(codes []*geoCountryCode) {
	sort.Slice(codes, func(i, j int) bool {
		return codes[i].Code < codes[j].Code
	})
}
