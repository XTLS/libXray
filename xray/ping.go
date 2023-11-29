package xray

import (
	"fmt"
	"net"
	"os"
	"path"

	"github.com/xtls/libxray/nodep"
	"github.com/xtls/xray-core/app/router"
	"google.golang.org/protobuf/proto"
)

// Ping Xray config and find the delay and country code of its outbound.
// datDir means the dir which geosite.dat and geoip.dat are in.
// configPath means the config.json file path.
// timeout means how long the http request will be cancelled if no response, in units of seconds.
// url means the website we use to test speed. "https://www.google.com" is a good choice for most cases.
// proxy means the local http/socks5 proxy, like "socks5://[::1]:1080".
func Ping(datDir string, configPath string, timeout int, url string, proxy string) string {
	initEnv(datDir)
	server, err := startXray(configPath)
	if err != nil {
		return fmt.Sprintf("%d::%s", nodep.PingDelayError, err)
	}

	if err := server.Start(); err != nil {
		return fmt.Sprintf("%d::%s", nodep.PingDelayError, err)
	}
	defer server.Close()

	delay, ip, err := nodep.MeasureDelay(timeout, url, proxy)
	if err != nil {
		return fmt.Sprintf("%d::%s", delay, err)
	}
	country := ""
	if len(ip) != 0 {
		code, err := FindCountryCodeOfIp(datDir, ip)
		if err == nil {
			country = code
		}
	}

	return fmt.Sprintf("%d:%s:", delay, country)
}

func FindCountryCodeOfIp(datDir string, ipAddress string) (string, error) {
	datPath := path.Join(datDir, "geoip.dat")
	geoipBytes, err := os.ReadFile(datPath)
	if err != nil {
		return "", err
	}
	var geoipList router.GeoIPList
	if err := proto.Unmarshal(geoipBytes, &geoipList); err != nil {
		return "", err
	}

	for _, geoip := range geoipList.Entry {
		m := &router.GeoIPMatcher{}
		m.SetReverseMatch(geoip.ReverseMatch)
		if err := m.Init(geoip.Cidr); err != nil {
			return "", err
		}
		ip := net.ParseIP(ipAddress)
		if ip != nil {
			if m.Match(ip) {
				return geoip.CountryCode, nil
			}
		}
	}
	return "", fmt.Errorf("can not find ip: %s location", ipAddress)
}
