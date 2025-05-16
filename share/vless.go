package share

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"strings"
)

type StreamSettings struct {
	Network  string           `json:"network"`
	Security string           `json:"security"`
	TLS      *TLSSettings     `json:"tlsSettings,omitempty"`
	Reality  *RealitySettings `json:"realitySettings,omitempty"`
	WS       *WSSettings      `json:"wsSettings,omitempty"`
	TCP      *TCPSettings     `json:"tcpSettings,omitempty"`
	QUIC     *QUICSettings    `json:"quicSettings,omitempty"`
	GRPC     *GRPCSettings    `json:"grpcSettings,omitempty"`
}

type TLSSettings struct {
	AllowInsecure bool     `json:"allowInsecure"`
	ALPN          []string `json:"alpn,omitempty"`
	Fingerprint   string   `json:"fingerprint"`
	ServerName    string   `json:"serverName"`
	Show          bool     `json:"show"`
}

type RealitySettings struct {
	AllowInsecure bool   `json:"allowInsecure"`
	Fingerprint   string `json:"fingerprint"`
	PublicKey     string `json:"publicKey"`
	ServerName    string `json:"serverName"`
	ShortID       string `json:"shortId"`
	Show          bool   `json:"show"`
	SpiderX       string `json:"spiderX"`
}

type WSSettings struct {
	Path    string            `json:"path"`
	Headers map[string]string `json:"headers"`
}

type TCPSettings struct {
	Header struct {
		Type string `json:"type"`
	} `json:"header"`
}

type QUICSettings struct {
	Security string `json:"security"`
	Key      string `json:"key"`
	Header   struct {
		Type string `json:"type"`
	} `json:"header"`
}

type GRPCSettings struct {
	Authority          string `json:"authority"`
	HealthCheckTimeout int    `json:"health_check_timeout"`
	IdleTimeout        int    `json:"idle_timeout"`
	MultiMode          bool   `json:"multiMode"`
	ServiceName        string `json:"serviceName"`
}

type Config struct {
	Inbounds []struct {
		Listen   string `json:"listen"`
		Port     int    `json:"port"`
		Protocol string `json:"protocol"`
		Settings struct {
			Auth      string `json:"auth,omitempty"`
			UDP       bool   `json:"udp,omitempty"`
			UserLevel int    `json:"userLevel,omitempty"`
		} `json:"settings"`
		Sniffing *struct {
			DestOverride []string `json:"destOverride"`
			Enabled      bool     `json:"enabled"`
			RouteOnly    bool     `json:"routeOnly"`
		} `json:"sniffing,omitempty"`
		Tag string `json:"tag"`
	} `json:"inbounds"`
	Log struct {
		LogLevel string `json:"loglevel"`
	} `json:"log"`
	Outbounds []struct {
		Mux *struct {
			Concurrency int  `json:"concurrency"`
			Enabled     bool `json:"enabled"`
		} `json:"mux,omitempty"`
		Protocol       string          `json:"protocol"`
		Settings       interface{}     `json:"settings"`
		StreamSettings *StreamSettings `json:"streamSettings,omitempty"`
		Tag            string          `json:"tag"`
	} `json:"outbounds"`
	Routing struct {
		DomainStrategy string `json:"domainStrategy"`
		Rules          []struct {
			IP          []string `json:"ip,omitempty"`
			Domain      []string `json:"domain,omitempty"`
			Network     string   `json:"network,omitempty"`
			OutboundTag string   `json:"outboundTag"`
			Port        string   `json:"port,omitempty"`
			Type        string   `json:"type"`
		} `json:"rules"`
	} `json:"routing"`
}

func ParseVLESS(uri string, httpPort int, socksPort int) string {
	if !strings.HasPrefix(uri, "vless://") {
		return ""
	}

	// Remove the scheme
	uri = strings.TrimPrefix(uri, "vless://")

	// Remove fragment (remarks) if present
	if idx := strings.Index(uri, "#"); idx != -1 {
		uri = uri[:idx]
	}

	// Split the URI into parts
	parts := strings.SplitN(uri, "@", 2)
	if len(parts) != 2 {
		return ""
	}

	userID := parts[0]
	serverPart := parts[1]

	// Split server part into address:port and query
	serverParts := strings.SplitN(serverPart, "?", 2)
	if len(serverParts) != 2 {
		return ""
	}

	// Parse address and port
	addrParts := strings.Split(serverParts[0], ":")
	if len(addrParts) != 2 {
		return ""
	}

	serverAddress := addrParts[0]
	portStr := strings.TrimSpace(strings.Split(addrParts[1], "/")[0])
	serverPort, err := strconv.Atoi(portStr)
	if err != nil {
		return ""
	}

	// Parse query parameters
	query, err := url.ParseQuery(serverParts[1])
	if err != nil {
		return ""
	}

	// Create base config
	config := &Config{
		Log: struct {
			LogLevel string `json:"loglevel"`
		}{
			LogLevel: "warning",
		},
	}

	// Set up inbounds
	config.Inbounds = []struct {
		Listen   string `json:"listen"`
		Port     int    `json:"port"`
		Protocol string `json:"protocol"`
		Settings struct {
			Auth      string `json:"auth,omitempty"`
			UDP       bool   `json:"udp,omitempty"`
			UserLevel int    `json:"userLevel,omitempty"`
		} `json:"settings"`
		Sniffing *struct {
			DestOverride []string `json:"destOverride"`
			Enabled      bool     `json:"enabled"`
			RouteOnly    bool     `json:"routeOnly"`
		} `json:"sniffing,omitempty"`
		Tag string `json:"tag"`
	}{
		{
			Listen:   "127.0.0.1",
			Port:     socksPort,
			Protocol: "socks",
			Settings: struct {
				Auth      string `json:"auth,omitempty"`
				UDP       bool   `json:"udp,omitempty"`
				UserLevel int    `json:"userLevel,omitempty"`
			}{
				Auth:      "noauth",
				UDP:       true,
				UserLevel: 8,
			},
			Sniffing: &struct {
				DestOverride []string `json:"destOverride"`
				Enabled      bool     `json:"enabled"`
				RouteOnly    bool     `json:"routeOnly"`
			}{
				DestOverride: []string{"http", "tls"},
				Enabled:      true,
				RouteOnly:    false,
			},
			Tag: "socks",
		},
		{
			Listen:   "127.0.0.1",
			Port:     httpPort,
			Protocol: "http",
			Settings: struct {
				Auth      string `json:"auth,omitempty"`
				UDP       bool   `json:"udp,omitempty"`
				UserLevel int    `json:"userLevel,omitempty"`
			}{
				UserLevel: 8,
			},
			Sniffing: nil,
			Tag:      "http",
		},
	}

	// Set up outbounds
	security := query.Get("security")
	if security == "none" {
		security = ""
	}
	streamSettings := StreamSettings{
		Network:  query.Get("type"),
		Security: security,
	}

	// Configure stream settings based on network type
	switch query.Get("type") {
	case "tcp":
		streamSettings.TCP = &TCPSettings{}
		streamSettings.TCP.Header.Type = "none"
	case "ws":
		streamSettings.WS = &WSSettings{
			Path: query.Get("path"),
			Headers: map[string]string{
				"Host": query.Get("host"),
			},
		}
	case "quic":
		streamSettings.QUIC = &QUICSettings{
			Security: "none",
			Key:      "",
		}
		streamSettings.QUIC.Header.Type = query.Get("headerType")
	case "grpc":
		streamSettings.GRPC = &GRPCSettings{
			Authority:          "",
			HealthCheckTimeout: 20,
			IdleTimeout:        60,
			MultiMode:          false,
			ServiceName:        query.Get("serviceName"),
		}
	}

	// Configure TLS settings if needed
	if query.Get("security") == "tls" {
		streamSettings.TLS = &TLSSettings{
			AllowInsecure: false,
			Fingerprint:   query.Get("fp"),
			ServerName:    query.Get("sni"),
			Show:          false,
		}
		if alpn := query.Get("alpn"); alpn != "" {
			streamSettings.TLS.ALPN = []string{alpn}
		}
	} else if query.Get("security") == "reality" {
		streamSettings.Reality = &RealitySettings{
			AllowInsecure: false,
			Fingerprint:   query.Get("fp"),
			PublicKey:     query.Get("pbk"),
			ServerName:    query.Get("sni"),
			ShortID:       query.Get("sid"),
			Show:          false,
			SpiderX:       query.Get("spx"),
		}
	}

	config.Outbounds = []struct {
		Mux *struct {
			Concurrency int  `json:"concurrency"`
			Enabled     bool `json:"enabled"`
		} `json:"mux,omitempty"`
		Protocol       string          `json:"protocol"`
		Settings       interface{}     `json:"settings"`
		StreamSettings *StreamSettings `json:"streamSettings,omitempty"`
		Tag            string          `json:"tag"`
	}{
		{
			Mux: &struct {
				Concurrency int  `json:"concurrency"`
				Enabled     bool `json:"enabled"`
			}{
				Concurrency: -1,
				Enabled:     false,
			},
			Protocol: "vless",
			Settings: map[string]interface{}{
				"vnext": []map[string]interface{}{
					{
						"address": serverAddress,
						"port":    serverPort,
						"users": []map[string]interface{}{
							{
								"encryption": "none",
								"flow":       query.Get("flow"),
								"id":         userID,
								"level":      8,
								"security":   "auto",
							},
						},
					},
				},
			},
			StreamSettings: &streamSettings,
			Tag:            "proxy",
		},
		{
			Protocol: "freedom",
			Settings: map[string]interface{}{
				"domainStrategy": "UseIP",
			},
			Tag: "direct",
		},
		{
			Protocol: "blackhole",
			Settings: map[string]interface{}{
				"response": map[string]interface{}{
					"type": "http",
				},
			},
			Tag: "block",
		},
	}

	// Set up routing
	config.Routing.DomainStrategy = "IPIfNonMatch"
	config.Routing.Rules = []struct {
		IP          []string `json:"ip,omitempty"`
		Domain      []string `json:"domain,omitempty"`
		Network     string   `json:"network,omitempty"`
		OutboundTag string   `json:"outboundTag"`
		Port        string   `json:"port,omitempty"`
		Type        string   `json:"type"`
	}{
		{
			Network:     "udp",
			OutboundTag: "block",
			Port:        "443",
			Type:        "field",
		},
		{
			Domain:      []string{"geosite:category-ads-all"},
			OutboundTag: "block",
			Type:        "field",
		},
		{
			Domain:      []string{"geosite:private"},
			OutboundTag: "direct",
			Type:        "field",
		},
		{
			IP:          []string{"geoip:private"},
			OutboundTag: "direct",
			Type:        "field",
		},
		{
			OutboundTag: "proxy",
			Port:        "0-65535",
			Type:        "field",
		},
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing VLESS URI: %v\n", err)
		os.Exit(1)
	}

	jsonData, err := json.MarshalIndent(config, "", "\t")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error marshaling config: %v\n", err)
		os.Exit(1)
	}

	return string(jsonData)
}
