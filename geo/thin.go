package geo

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/xtls/libxray/nodep"
	"github.com/xtls/xray-core/app/router"
	"github.com/xtls/xray-core/infra/conf"
	"google.golang.org/protobuf/proto"
)

// Read geo data and cut the codes we need.
// datDir means the dir which geo dat are in.
// configPath means where xray config file is.
// dstDir means the dir which new geo dat are in.
//
// This function is used to reduce memory when init instance.
// You can cut the country codes which rules and nameservers contain.
func ThinGeoData(datDir string, configPath string, dstDir string) error {
	xrayBytes, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	domain, ip := loadXrayConfig(xrayBytes)
	domainCodes := filterAndStrip(domain, "geosite")
	for key, value := range domainCodes {
		err := cutGeoSite(datDir, dstDir, key, value)
		if err != nil {
			return err
		}
	}

	ipCodes := filterAndStrip(ip, "geoip")
	for key, value := range ipCodes {
		err := cutGeoIP(datDir, dstDir, key, value)
		if err != nil {
			return err
		}
	}

	return nil
}

func loadXrayConfig(configBytes []byte) ([]string, []string) {
	domain := []string{}
	ip := []string{}

	var xray conf.Config
	err := json.Unmarshal(configBytes, &xray)
	if err != nil {
		return domain, ip
	}

	routingDomain, routingIP := filterRouting(xray)
	domain = append(domain, routingDomain...)
	ip = append(ip, routingIP...)

	dnsDomain, dnsIP := filterDns(xray)
	domain = append(domain, dnsDomain...)
	ip = append(ip, dnsIP...)

	return domain, ip
}

func filterRouting(xray conf.Config) ([]string, []string) {
	domain := []string{}
	ip := []string{}

	routing := xray.RouterConfig
	if routing == nil {
		return domain, ip
	}
	rules := routing.RuleList
	if len(rules) == 0 {
		return domain, ip
	}
	// parse rules
	// we only care about domain and ip
	type RawRule struct {
		Domain *conf.StringList `json:"domain"`
		IP     *conf.StringList `json:"ip"`
	}

	for _, rule := range rules {
		var rawRule RawRule
		err := json.Unmarshal(rule, &rawRule)
		if err != nil {
			continue
		}
		if rawRule.Domain != nil {
			domain = append(domain, *rawRule.Domain...)
		}
		if rawRule.IP != nil {
			ip = append(ip, *rawRule.IP...)
		}
	}
	return domain, ip
}

func filterDns(xray conf.Config) ([]string, []string) {
	domain := []string{}
	ip := []string{}

	dns := xray.DNSConfig
	if dns == nil {
		return domain, ip
	}
	servers := dns.Servers
	if len(servers) == 0 {
		return domain, ip
	}

	for _, server := range servers {
		if len(server.Domains) > 0 {
			domain = append(domain, server.Domains...)
		}
		if len(server.ExpectIPs) > 0 {
			ip = append(ip, server.ExpectIPs...)
		}
	}
	return domain, ip
}

func filterAndStrip(rules []string, retain string) map[string][]string {
	m := make(map[string][]string)
	retainPrefix := fmt.Sprintf("%s:", retain)
	retainFile := fmt.Sprintf("%s.dat", retain)
	for _, rule := range rules {
		if strings.HasPrefix(rule, retainPrefix) {
			values := strings.SplitN(rule, ":", 2)
			appendMap(m, retainFile, values[1])
		} else if strings.HasPrefix(rule, "ext:") {
			values := strings.SplitN(rule, ":", 3)
			appendMap(m, values[1], values[2])
		}
	}
	return m
}

func appendMap(m map[string][]string, key string, value string) {
	v, ok := m[key]
	if ok {
		v = append(v, value)
	} else {
		v = []string{value}
	}
	m[key] = v
}

func cutGeoSite(datDir string, dstDir string, fileName string, codes []string) error {
	srcPath := path.Join(datDir, fileName)
	dstPath := path.Join(dstDir, fileName)
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
	if err := nodep.WriteBytes(newDatBytes, dstPath); err != nil {
		return err
	}

	return nil
}

func cutGeoIP(datDir string, dstDir string, fileName string, codes []string) error {
	srcPath := path.Join(datDir, fileName)
	dstPath := path.Join(dstDir, fileName)
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
