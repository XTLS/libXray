package share

import (
	"encoding/json"

	"github.com/xtls/xray-core/infra/conf"
)

// buildHy2FinalMask builds Hysteria2 bandwidth / salamander / port-hopping masks (shared by URI and Clash).
func buildHy2FinalMask(up, down, ports string, hopInterval *int32, obfsType, obfsPassword string) (*conf.FinalMask, error) {
	var quicParams *conf.QuicParamsConfig
	if up != "" || down != "" {
		quicParams = &conf.QuicParamsConfig{Congestion: "brutal"}
		if up != "" {
			quicParams.BrutalUp = conf.Bandwidth(up)
		}
		if down != "" {
			quicParams.BrutalDown = conf.Bandwidth(down)
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
	if ports != "" {
		var portList conf.PortList
		portListJSON, err := json.Marshal(ports)
		if err != nil {
			return nil, err
		}
		if err := json.Unmarshal(portListJSON, &portList); err != nil {
			return nil, err
		}
		interval := int32(30)
		if hopInterval != nil && *hopInterval != 0 {
			interval = *hopInterval
		}
		udpHop := &conf.UDPHop{
			Mode:        "intervalRemote",
			RemotePorts: portList,
			Interval:    conf.Int32Range{Left: interval, Right: interval, From: interval, To: interval},
		}
		hopSettings, err := convertJsonToRawMessage(udpHop)
		if err != nil {
			return nil, err
		}
		// Xray wraps masks in reverse order; port hopping must be outermost.
		udpMasks = append(udpMasks, conf.Mask{Type: "udphop", Settings: &hopSettings})
	}

	if quicParams == nil && len(udpMasks) == 0 {
		return nil, nil
	}
	return &conf.FinalMask{QuicParams: quicParams, Udp: udpMasks}, nil
}
