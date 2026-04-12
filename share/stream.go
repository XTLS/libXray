package share

import (
	"net/url"
	"strings"

	"github.com/xtls/xray-core/infra/conf"
)

func (proxy xrayShareLink) streamSettings(link *url.URL) (*conf.StreamConfig, error) {
	query := link.Query()
	if len(query) == 0 {
		return nil, nil
	}

	fields := transportFieldsFromURLQuery(query)
	streamSettings, err := buildStreamFromTransportFields(fields)
	if err != nil {
		return nil, err
	}

	if err := proxy.parseSecurityFromURL(link, streamSettings); err != nil {
		return nil, err
	}
	return streamSettings, nil
}

func (proxy xrayShareLink) parseSecurityFromURL(link *url.URL, streamSettings *conf.StreamConfig) error {
	query := link.Query()

	tlsSettings := &conf.TLSConfig{}
	realitySettings := &conf.REALITYConfig{}

	fp := query.Get("fp")
	tlsSettings.Fingerprint = fp
	realitySettings.Fingerprint = fp

	sni := query.Get("sni")
	tlsSettings.ServerName = sni
	realitySettings.ServerName = sni

	tlsSettings.ECHConfigList = query.Get("ech")
	tlsSettings.PinnedPeerCertSha256 = query.Get("pcs")
	tlsSettings.VerifyPeerCertByName = query.Get("vcn")

	if alpn := query.Get("alpn"); alpn != "" {
		tlsSettings.ALPN = new(conf.StringList(strings.Split(alpn, ",")))
	}

	if query.Get("insecure") == "1" {
		tlsSettings.AllowInsecure = true
	}

	pbk := query.Get("pbk")
	realitySettings.Password = pbk
	realitySettings.PublicKey = pbk
	realitySettings.ShortId = query.Get("sid")
	realitySettings.Mldsa65Verify = query.Get("pqv")
	realitySettings.SpiderX = query.Get("spx")

	if security := query.Get("security"); security == "" {
		streamSettings.Security = "none"
	} else {
		streamSettings.Security = security
	}

	switch proxy.link.Scheme {
	case "trojan", "hysteria2", "hy2":
		if streamSettings.Security == "none" {
			streamSettings.Security = "tls"
		}
	}

	network, err := streamSettings.Network.Build()
	if err != nil {
		return err
	}
	if network == "websocket" && tlsSettings.ServerName == "" &&
		streamSettings.WSSettings != nil && streamSettings.WSSettings.Host != "" {
		tlsSettings.ServerName = streamSettings.WSSettings.Host
	}

	switch streamSettings.Security {
	case "tls":
		streamSettings.TLSSettings = tlsSettings
	case "reality":
		streamSettings.REALITYSettings = realitySettings
	}
	return nil
}
