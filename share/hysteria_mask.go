package share

import (
	"encoding/json"

	"github.com/xtls/xray-core/infra/conf"
)

// buildHy2FinalMask builds Hysteria2 QUIC hop / bandwidth / salamander mask (shared by URI and Clash).
func buildHy2FinalMask(up, down, ports string, hopInterval *int32, obfsType, obfsPassword string) (*conf.FinalMask, error) {
	var quicParams *conf.QuicParamsConfig
	if up != "" || down != "" || ports != "" {
		quicParams = &conf.QuicParamsConfig{}
		if up != "" || down != "" {
			quicParams.Congestion = "brutal"
		}
		if up != "" {
			quicParams.BrutalUp = conf.Bandwidth(up)
		}
		if down != "" {
			quicParams.BrutalDown = conf.Bandwidth(down)
		}
		if ports != "" {
			udpHop := conf.UdpHop{}
			portListJSON, err := json.Marshal(ports)
			if err != nil {
				return nil, err
			}
			udpHop.PortList = portListJSON
			if hopInterval != nil {
				i := *hopInterval
				udpHop.Interval = &conf.Int32Range{Left: i, Right: i, From: i, To: i}
			}
			quicParams.UdpHop = udpHop
		}
	}

	var udpMasks []conf.Mask
	if obfsType == "salamander" && obfsPassword != "" {
		obfs := conf.Mask{Type: "salamander"}
		salamander := &conf.Salamander{Password: obfsPassword}
		salamanderRawMessage, err := convertJsonToRawMessage(salamander)
		if err != nil {
			return nil, err
		}
		obfs.Settings = &salamanderRawMessage
		udpMasks = []conf.Mask{obfs}
	}

	if quicParams == nil && len(udpMasks) == 0 {
		return nil, nil
	}
	return &conf.FinalMask{QuicParams: quicParams, Udp: udpMasks}, nil
}
