package share

import (
	"encoding/json"
	"errors"
	"net"
	"net/url"
	"reflect"
	"strconv"
	"strings"

	"github.com/xtls/xray-core/infra/conf"
)

var errHysteria2Link = errors.New("unsupported or invalid Hysteria2 share link")

// Hysteria permits port lists in the authority, which net/url rejects. Replace
// only that port component before parsing; let net/url handle escaping/userinfo.
func hysteria2URL(text string) (*url.URL, conf.PortList, error) {
	scheme, rest, _ := strings.Cut(text, "://")
	end := strings.IndexAny(rest, "/?#")
	if end < 0 {
		end = len(rest)
	}
	authority, suffix := rest[:end], rest[end:]
	user, hostPort := "", authority
	if at := strings.LastIndexByte(authority, '@'); at >= 0 {
		user, hostPort = authority[:at+1], authority[at+1:]
	}
	host, portText := hostPort, "443"
	if strings.HasPrefix(hostPort, "[") {
		close := strings.IndexByte(hostPort, ']')
		if close < 0 {
			return nil, conf.PortList{}, errHysteria2Link
		}
		host = hostPort[:close+1]
		if net.ParseIP(host[1:len(host)-1]) == nil {
			return nil, conf.PortList{}, errHysteria2Link
		}
		if tail := hostPort[close+1:]; tail != "" {
			if !strings.HasPrefix(tail, ":") {
				return nil, conf.PortList{}, errHysteria2Link
			}
			portText = tail[1:]
		}
	} else if strings.Contains(hostPort, ":") {
		host, portText, _ = strings.Cut(hostPort, ":")
	}
	ports, err := hysteria2Ports(portText)
	if err != nil {
		return nil, conf.PortList{}, err
	}
	link, err := url.Parse(scheme + "://" + user + host + ":" + strconv.Itoa(int(ports.Range[0].From)) + suffix)
	if err != nil || link.Hostname() == "" || (link.Path != "" && link.Path != "/") {
		return nil, conf.PortList{}, errHysteria2Link
	}
	return link, ports, nil
}

func hysteria2Ports(text string) (conf.PortList, error) {
	ports := conf.PortList{}
	for _, item := range strings.Split(text, ",") {
		from, to, ranged := strings.Cut(item, "-")
		if !ranged {
			to = from
		}
		left, leftErr := strconv.ParseUint(from, 10, 16)
		right, rightErr := strconv.ParseUint(to, 10, 16)
		if leftErr != nil || rightErr != nil || left == 0 || left > right {
			return conf.PortList{}, errHysteria2Link
		}
		ports.Range = append(ports.Range, conf.PortRange{From: uint32(left), To: uint32(right)})
	}
	return ports, nil
}

func parseHysteria2Link(text string) (*conf.OutboundDetourConfig, error) {
	link, ports, err := hysteria2URL(text)
	if err != nil {
		return nil, err
	}
	query, err := url.ParseQuery(link.RawQuery)
	if err != nil {
		return nil, errHysteria2Link
	}
	for _, values := range query {
		if len(values) != 1 {
			return nil, errHysteria2Link
		}
	}
	// Core no longer supports skipping verification. Its certificate pin is not
	// equivalent to Hysteria's leaf pin + ordinary PKI. Retain explicit Xray pcs
	// and vcn extensions, but never reinterpret the official TLS flags as those.
	for _, key := range []string{"insecure", "allowInsecure"} {
		if query.Has(key) {
			insecure, err := strconv.ParseBool(query.Get(key))
			if err != nil || insecure {
				return nil, errHysteria2Link
			}
		}
	}
	if query.Get("pinSHA256") != "" || (query.Get("security") != "" && query.Get("security") != "tls") {
		return nil, errHysteria2Link
	}
	// Preserve old port-query links, while exporting standard multi-port authorities.
	for _, key := range []string{"ports", "mport"} {
		if query.Has(key) {
			if len(ports.Range) != 1 || ports.Range[0].From != ports.Range[0].To || (key == "mport" && query.Has("ports")) {
				return nil, errHysteria2Link
			}
			ports, err = hysteria2Ports(query.Get(key))
			if err != nil {
				return nil, err
			}
		}
	}
	settings, err := convertJsonToRawMessage(&conf.HysteriaClientConfig{
		Version: 2, Address: parseAddress(link.Hostname()), Port: uint16(ports.Range[0].From),
	})
	if err != nil {
		return nil, err
	}
	auth := ""
	if link.User != nil {
		auth = link.User.Username()
		if password, present := link.User.Password(); present {
			auth += ":" + password
		}
	}
	stream := &conf.StreamConfig{
		Network:          new(conf.TransportProtocol("hysteria")),
		HysteriaSettings: &conf.HysteriaConfig{Version: 2, Auth: auth},
	}
	if err := (xrayShareLink{link: link}).parseSecurityFromURL(link, stream); err != nil {
		return nil, err
	}
	mask := &conf.FinalMask{}
	if obfs := query.Get("obfs"); obfs != "" && obfs != "none" {
		password := query.Get("obfs-password")
		if obfs != "salamander" || len(password) < 4 {
			return nil, errHysteria2Link
		}
		raw, err := convertJsonToRawMessage(&conf.Salamander{Password: password})
		if err != nil {
			return nil, err
		}
		mask.Udp = append(mask.Udp, conf.Mask{Type: "salamander", Settings: &raw})
	} else if query.Get("obfs-password") != "" {
		return nil, errHysteria2Link
	}
	if len(ports.Range) > 1 || ports.Range[0].From != ports.Range[0].To {
		interval := int64(30)
		if query.Has("hop-interval") {
			interval, err = strconv.ParseInt(query.Get("hop-interval"), 10, 32)
			if err != nil || interval < 5 {
				return nil, errHysteria2Link
			}
		}
		raw, err := convertJsonToRawMessage(&conf.UDPHop{
			Mode: "intervalLocal,intervalRemote", RemotePorts: ports,
			Interval: conf.Int32Range{Left: int32(interval), Right: int32(interval), From: int32(interval), To: int32(interval)},
		})
		if err != nil {
			return nil, err
		}
		// The core wraps UDP masks in reverse order; hopping must wrap the raw socket.
		mask.Udp = append(mask.Udp, conf.Mask{Type: "udphop", Settings: &raw})
	} else if query.Has("hop-interval") {
		return nil, errHysteria2Link
	}
	if query.Get("up") != "" || query.Get("down") != "" {
		mask.QuicParams = &conf.QuicParamsConfig{
			Congestion: "brutal", BrutalUp: conf.Bandwidth(query.Get("up")), BrutalDown: conf.Bandwidth(query.Get("down")),
		}
	}
	if len(mask.Udp) > 0 || mask.QuicParams != nil {
		stream.FinalMask = mask
	}
	return &conf.OutboundDetourConfig{
		Protocol: "hysteria", Tag: link.Fragment, Settings: &settings, StreamSetting: stream,
	}, nil
}

func hysteria2ShareLink(outbound conf.OutboundDetourConfig) (*url.URL, error) {
	settings, err := decodeOutboundSettings[conf.HysteriaClientConfig](outbound)
	if err != nil {
		return nil, err
	}
	stream := outbound.StreamSetting
	if settings.Version != 2 || settings.Address == nil || settings.Port == 0 || stream == nil ||
		stream.Security != "tls" || stream.HysteriaSettings == nil || stream.HysteriaSettings.Version != 2 {
		return nil, errHysteria2Link
	}
	network := stream.Network
	if stream.Method != nil {
		network = stream.Method
	}
	if network == nil || *network != "hysteria" {
		return nil, errHysteria2Link
	}
	host := strings.Trim(settings.Address.String(), "[]")
	link := &url.URL{
		Scheme: "hysteria2", User: url.User(stream.HysteriaSettings.Auth),
		Host: net.JoinHostPort(host, strconv.Itoa(int(settings.Port))), Fragment: getOutboundName(outbound),
	}
	// Reuse the existing TLS field projection; Hysteria does not use type/security.
	streamSettingsQuery(outbound, link)
	query := link.Query()
	query.Del("type")
	query.Del("security")
	query.Del("fm")
	if tls := stream.TLSSettings; tls != nil {
		remaining := *tls
		remaining.ServerName, remaining.Fingerprint, remaining.ECHConfigList = "", "", ""
		remaining.PinnedPeerCertSha256, remaining.VerifyPeerCertByName = "", ""
		remaining.ALPN = nil
		if !reflect.DeepEqual(remaining, conf.TLSConfig{}) {
			return nil, errHysteria2Link
		}
		if _, err := tls.Build(); err != nil {
			return nil, errHysteria2Link
		}
	}
	if mask := stream.FinalMask; mask != nil {
		if len(mask.Tcp) > 0 {
			return nil, errHysteria2Link
		}
		for index, entry := range mask.Udp {
			if entry.Settings == nil {
				return nil, errHysteria2Link
			}
			switch entry.Type {
			case "salamander":
				var obfs conf.Salamander
				if err := json.Unmarshal(*entry.Settings, &obfs); err != nil || index != 0 ||
					len(obfs.Password) < 4 || obfs.PacketSize != (conf.Int32Range{}) {
					return nil, errHysteria2Link
				}
				query.Set("obfs", "salamander")
				query.Set("obfs-password", obfs.Password)
			case "udphop":
				var hop conf.UDPHop
				if err := json.Unmarshal(*entry.Settings, &hop); err != nil || index != len(mask.Udp)-1 ||
					hop.Mode != "intervalLocal,intervalRemote" || len(hop.RemoteIPs) > 0 || hop.Sockopt != nil ||
					hop.Interval.From < 5 || hop.Interval.From != hop.Interval.To {
					return nil, errHysteria2Link
				}
				ports, err := hysteria2Ports(hop.RemotePorts.String())
				if err != nil || (len(ports.Range) == 1 && ports.Range[0].From == ports.Range[0].To) {
					return nil, errHysteria2Link
				}
				link.Host = net.JoinHostPort(host, ports.String())
				if hop.Interval.From != 30 {
					query.Set("hop-interval", strconv.Itoa(int(hop.Interval.From)))
				}
			default:
				return nil, errHysteria2Link
			}
		}
		// QUIC bandwidth/tuning is client-local, not part of a standard share URI.
	}
	link.RawQuery = strings.ReplaceAll(query.Encode(), "+", "%20")
	return link, nil
}
