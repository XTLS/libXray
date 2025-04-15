package libXray

import (
	"github.com/xtls/libxray/nodep"
)

type getFreePortsResponse struct {
	Ports []int `json:"ports,omitempty"`
}

// GetFreePorts Wrapper of nodep.GetFreePorts
// count means how many ports you need.
func GetFreePorts(count int) string {
	var response nodep.CallResponse[*getFreePortsResponse]
	ports, err := nodep.GetFreePorts(count)
	if err != nil {
		return response.EncodeToBase64(nil, err)
	}
	var res getFreePortsResponse
	res.Ports = ports
	return response.EncodeToBase64(&res, nil)
}
