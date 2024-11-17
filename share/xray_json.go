package share

import (
	"encoding/json"

	"github.com/xtls/xray-core/common/net"
	"github.com/xtls/xray-core/infra/conf"
)

type XrayRawSettingsHeader struct {
	Type    string                        `json:"type,omitempty"`
	Request *XrayRawSettingsHeaderRequest `json:"request,omitempty"`
}

type XrayRawSettingsHeaderRequest struct {
	Path    []string                             `json:"path,omitempty"`
	Headers *XrayRawSettingsHeaderRequestHeaders `json:"headers,omitempty"`
}

type XrayRawSettingsHeaderRequestHeaders struct {
	Host []string `json:"Host,omitempty"`
}

type XrayFakeHeader struct {
	Type string `json:"type,omitempty"`
}

func setOutboundName(outbound *conf.OutboundDetourConfig, name string) {
	outbound.SendThrough = &name
}

func getOutboundName(outbound conf.OutboundDetourConfig) string {
	if outbound.SendThrough != nil {
		return *outbound.SendThrough
	}
	return ""
}

func parseAddress(addr string) *conf.Address {
	address := &conf.Address{}
	address.Address = net.ParseAddress(addr)
	return address
}

func convertJsonToRawMessage(v any) (json.RawMessage, error) {
	vBytes, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return json.RawMessage(vBytes), nil
}
