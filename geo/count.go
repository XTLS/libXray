package geo

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"sort"

	"github.com/xtls/libxray/nodep"
	"github.com/xtls/xray-core/app/router"
	"google.golang.org/protobuf/proto"
)

type geoList struct {
	Codes         []*geoCountryCode `json:"codes,omitempty"`
	CategoryCount int               `json:"categoryCount,omitempty"`
	RuleCount     int               `json:"ruleCount,omitempty"`
}
type geoCountryCode struct {
	Code      string `json:"code,omitempty"`
	RuleCount int    `json:"ruleCount,omitempty"`
}

const (
	geoTypeDomain string = "domain"
	geoTypeIP     string = "ip"
)

// Read geo data and write all codes to text file.
// datDir means the dir which geo dat are in.
// name means the geo dat file name, like "geosite", "geoip"
// geoType must be the value of geoType
func CountGeoData(datDir string, name string, geoType string) error {
	switch geoType {
	case geoTypeDomain:
		if err := countGeoSite(datDir, name); err != nil {
			return err
		}
	case geoTypeIP:
		if err := countGeoIP(datDir, name); err != nil {
			return err
		}
	default:
		return fmt.Errorf("wrong geoType: %s", geoType)
	}
	return nil
}

func countGeoSite(datDir string, name string) error {
	datName := name + ".dat"
	jsonName := name + ".json"
	datPath := path.Join(datDir, datName)
	jsonPath := path.Join(datDir, jsonName)
	geositeBytes, err := os.ReadFile(datPath)
	if err != nil {
		return err
	}
	var geositeList router.GeoSiteList
	if err := proto.Unmarshal(geositeBytes, &geositeList); err != nil {
		return err
	}

	var list geoList
	list.CategoryCount = len(geositeList.Entry)
	var codes []*geoCountryCode
	for _, site := range geositeList.Entry {
		var siteCode geoCountryCode
		siteCode.Code = site.CountryCode
		siteCode.RuleCount = len(site.Domain)
		codes = append(codes, &siteCode)
		list.RuleCount += siteCode.RuleCount
		for _, domain := range site.Domain {
			for _, attribute := range domain.Attribute {
				attr := fmt.Sprintf("%s@%s", site.CountryCode, attribute.Key)
				attrCode := findAttrCode(codes, attr)
				if attrCode == nil {
					var newCode geoCountryCode
					newCode.Code = attr
					newCode.RuleCount = 1
					codes = append(codes, &newCode)
				} else {
					attrCode.RuleCount += 1
				}
			}
		}
	}

	sortCodes(codes)
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

func countGeoIP(datDir string, name string) error {
	datName := name + ".dat"
	jsonName := name + ".json"
	datPath := path.Join(datDir, datName)
	jsonPath := path.Join(datDir, jsonName)
	geoipBytes, err := os.ReadFile(datPath)
	if err != nil {
		return err
	}
	var geoipList router.GeoIPList
	if err := proto.Unmarshal(geoipBytes, &geoipList); err != nil {
		return err
	}

	var list geoList
	list.CategoryCount = len(geoipList.Entry)
	var codes []*geoCountryCode
	for _, geoip := range geoipList.Entry {
		var code geoCountryCode
		code.Code = geoip.CountryCode
		code.RuleCount = len(geoip.Cidr)
		codes = append(codes, &code)
		list.RuleCount += code.RuleCount
	}

	sortCodes(codes)

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
