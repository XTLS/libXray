// libXray is an Xray wrapper focusing on improving the experience of Xray-core mobile development.
package libXray

import (
	"encoding/json"

	"github.com/xtls/libxray/share"
	"github.com/xtls/libxray/xray"
)

type LibXrayMethod string

const LibXrayAPIVersion = 3

const (
	LibXrayMethodGetFreePorts                LibXrayMethod = "getFreePorts"
	LibXrayMethodConvertShareLinksToXrayJson LibXrayMethod = "convertShareLinksToXrayJson"
	LibXrayMethodConvertXrayJsonToShareLinks LibXrayMethod = "convertXrayJsonToShareLinks"
	LibXrayMethodGenerateAgeKeyPair          LibXrayMethod = "generateAgeKeyPair"
	LibXrayMethodCountGeoData                LibXrayMethod = "countGeoData"
	LibXrayMethodPingBatch                   LibXrayMethod = "pingBatch"
	LibXrayMethodTestXray                    LibXrayMethod = "testXray"
	LibXrayMethodCheckRoute                  LibXrayMethod = "checkRoute"
	LibXrayMethodRunXray                     LibXrayMethod = "runXray"
	LibXrayMethodStopXray                    LibXrayMethod = "stopXray"
	LibXrayMethodXrayVersion                 LibXrayMethod = "xrayVersion"
	LibXrayMethodGetXrayState                LibXrayMethod = "getXrayState"
)

type LibXrayInvokeRequest struct {
	APIVersion int             `json:"apiVersion,omitempty"`
	Method     LibXrayMethod   `json:"method,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
}

type GetFreePortsRequest struct {
	Count int `json:"count,omitempty"`
}

type GetFreePortsResponse struct {
	Ports []int `json:"ports,omitempty"`
}

type AgeDecryptConfig struct {
	SecretKey string `json:"secretKey,omitempty"`
}

type ConvertShareLinksToXrayJsonRequest struct {
	Text         string            `json:"text,omitempty"`
	Age          *AgeDecryptConfig `json:"age,omitempty"`
	IncludeStats bool              `json:"includeStats,omitempty"`
}

type ConvertShareLinksToXrayJsonResponse = share.ParseStats

type AgeKeyType string

const (
	AgeKeyTypeX25519 AgeKeyType = "x25519"
	AgeKeyTypeHybrid AgeKeyType = "hybrid"
)

type GenerateAgeKeyPairRequest struct {
	KeyType AgeKeyType `json:"keyType,omitempty"`
}

type GenerateAgeKeyPairResponse struct {
	SecretKey string `json:"secretKey,omitempty"`
	PublicKey string `json:"publicKey,omitempty"`
}

type ConvertXrayJsonToShareLinksRequest struct {
	XrayJson string `json:"xrayJson,omitempty"`
}

type ConvertXrayJsonToShareLinksResponse struct {
	Links string `json:"links,omitempty"`
}

type CountGeoDataRequest struct {
	Name    string `json:"name,omitempty"`
	GeoType string `json:"geoType,omitempty"`
	DatDir  string `json:"datDir,omitempty"`
}

type PingBatchRequest struct {
	Configs     []PingBatchItemRequest `json:"configs,omitempty"`
	Timeout     int                    `json:"timeout,omitempty"`
	URL         string                 `json:"url,omitempty"`
	LocationURL string                 `json:"locationUrl,omitempty"`
}

type PingBatchItemRequest struct {
	XrayJson    string `json:"xrayJson,omitempty"`
	OutboundTag string `json:"outboundTag,omitempty"`
}

type PingBatchResponse struct {
	Results []PingBatchItemResponse `json:"results,omitempty"`
}

type PingBatchItemResponse struct {
	Success       bool    `json:"success"`
	Delay         int64   `json:"delay"`
	Error         string  `json:"error,omitempty"`
	LocationJSON  *string `json:"locationJson,omitempty"`
	LocationError string  `json:"locationError,omitempty"`
}

type RunXrayRequest struct {
	XrayJson string         `json:"xrayJson,omitempty"`
	Runtime  *RuntimeConfig `json:"runtime,omitempty"`
}

type RuntimeConfig = xray.RuntimeConfig
type RuntimeSnapshot = xray.RuntimeSnapshot

type TestXrayRequest struct {
	XrayJson   string `json:"xrayJson,omitempty"`
	BuildOnly  bool   `json:"buildOnly,omitempty"`
	URL        string `json:"url,omitempty"`
	Timeout    int    `json:"timeout,omitempty"`
	InboundTag string `json:"inboundTag,omitempty"`
}

type TestXrayResponse struct {
	Delay int64 `json:"delay"`
}

type CheckRouteRequest struct {
	XrayJson   string `json:"xrayJson"`
	Domain     string `json:"domain,omitempty"`
	IP         string `json:"ip,omitempty"`
	Port       int    `json:"port"`
	Network    string `json:"network"`
	InboundTag string `json:"inboundTag,omitempty"`
	Timeout    int    `json:"timeout"`
}

type CheckRouteResponse struct {
	Matched     bool   `json:"matched"`
	RuleTag     string `json:"ruleTag"`
	OutboundTag string `json:"outboundTag"`
	BalancerTag string `json:"balancerTag"`
	Defaulted   bool   `json:"defaulted"`
}

type XrayVersionResponse struct {
	Version string `json:"version,omitempty"`
}

type GetXrayStateResponse struct {
	Running bool `json:"running"`
}
