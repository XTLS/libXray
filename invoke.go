package libXray

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/xtls/libxray/geo"
	"github.com/xtls/libxray/nodep"
	"github.com/xtls/libxray/share"
	"github.com/xtls/libxray/xray"
)

type invokeResponse struct {
	Success bool   `json:"success"`
	Data    any    `json:"data"`
	Err     string `json:"error"`
}

const (
	maxInvokeJSONSizeMiB = 16
	maxInvokeJSONBytes   = maxInvokeJSONSizeMiB * 1024 * 1024
)

func Invoke(requestJSON string) string {
	if len(requestJSON) > maxInvokeJSONBytes {
		return encodeInvokeResponse(nil, errors.New(invokeJSONSizeLimitMessage("request")))
	}
	var request LibXrayInvokeRequest
	if err := json.Unmarshal([]byte(requestJSON), &request); err != nil {
		return encodeInvokeResponse(nil, err)
	}
	if err := validateAPIVersion(request.APIVersion); err != nil {
		return encodeInvokeResponse(nil, err)
	}

	switch request.Method {
	case LibXrayMethodGetFreePorts:
		return invokeGetFreePorts(request.Payload)
	case LibXrayMethodConvertShareLinksToXrayJson:
		return invokeConvertShareLinksToXrayJson(request.Payload)
	case LibXrayMethodConvertXrayJsonToShareLinks:
		return invokeConvertXrayJsonToShareLinks(request.Payload)
	case LibXrayMethodGenerateAgeKeyPair:
		return invokeGenerateAgeKeyPair(request.Payload)
	case LibXrayMethodCountGeoData:
		return invokeCountGeoData(request.Payload)
	case LibXrayMethodPingBatch:
		return invokePingBatch(request.Payload)
	case LibXrayMethodTestXray:
		return invokeTestXray(request.Payload)
	case LibXrayMethodRunXray:
		return invokeRunXray(request.Payload)
	case LibXrayMethodStopXray:
		return encodeInvokeNoDataResponse(xray.StopXray())
	case LibXrayMethodXrayVersion:
		return encodeInvokeResponse(&XrayVersionResponse{Version: xray.XrayVersion()}, nil)
	case LibXrayMethodGetXrayState:
		return encodeInvokeResponse(&GetXrayStateResponse{Running: xray.GetXrayState()}, nil)
	default:
		return encodeInvokeResponse(nil, errors.New("unknown method"))
	}
}
func validateAPIVersion(version int) error {
	if version == LibXrayAPIVersion {
		return nil
	}
	return errors.New("unsupported apiVersion")
}

func decodePayload[T any](payload json.RawMessage) (T, error) {
	var request T
	if len(payload) == 0 {
		return request, nil
	}
	err := json.Unmarshal(payload, &request)
	return request, err
}

func encodeInvokeResponse(data any, err error) string {
	response := invokeResponse{Data: data}
	if err != nil {
		response.Success = false
		response.Err = err.Error()
	} else {
		response.Success = true
	}
	raw, err := json.Marshal(&response)
	if err != nil {
		return `{"success":false,"data":null,"error":"failed to encode response"}`
	}
	if len(raw) > maxInvokeJSONBytes {
		return encodeInvokeFailure(invokeJSONSizeLimitMessage("response"))
	}
	return string(raw)
}

func invokeJSONSizeLimitMessage(kind string) string {
	return fmt.Sprintf(
		"invoke %s exceeds the %d MiB size limit",
		kind,
		maxInvokeJSONSizeMiB,
	)
}

func encodeInvokeFailure(message string) string {
	raw, err := json.Marshal(&invokeResponse{Err: message})
	if err != nil {
		return `{"success":false,"data":null,"error":"failed to encode response"}`
	}
	return string(raw)
}

func encodeInvokeNoDataResponse(err error) string {
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	return encodeInvokeResponse(struct{}{}, nil)
}

func invokeGetFreePorts(payload json.RawMessage) string {
	request, err := decodePayload[GetFreePortsRequest](payload)
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	ports, err := nodep.GetFreePorts(request.Count)
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	return encodeInvokeResponse(&GetFreePortsResponse{Ports: ports}, nil)
}

func invokeConvertShareLinksToXrayJson(payload json.RawMessage) string {
	request, err := decodePayload[ConvertShareLinksToXrayJsonRequest](payload)
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	secretKey := ""
	if request.Age != nil {
		secretKey = request.Age.SecretKey
	}
	xrayJson, err := share.ConvertShareLinksToXrayJsonWithAge(request.Text, secretKey)
	return encodeInvokeResponse(xrayJson, err)
}

func invokeGenerateAgeKeyPair(payload json.RawMessage) string {
	request, err := decodePayload[GenerateAgeKeyPairRequest](payload)
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	pair, err := share.GenerateAgeKeyPair(share.AgeKeyType(request.KeyType))
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	return encodeInvokeResponse(&GenerateAgeKeyPairResponse{
		SecretKey: pair.SecretKey,
		PublicKey: pair.PublicKey,
	}, nil)
}

func invokeConvertXrayJsonToShareLinks(payload json.RawMessage) string {
	request, err := decodePayload[ConvertXrayJsonToShareLinksRequest](payload)
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	links, err := share.ConvertXrayJsonToShareLinks([]byte(request.XrayJson))
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	return encodeInvokeResponse(&ConvertXrayJsonToShareLinksResponse{Links: links}, nil)
}

func invokeCountGeoData(payload json.RawMessage) string {
	request, err := decodePayload[CountGeoDataRequest](payload)
	if err != nil {
		return encodeInvokeNoDataResponse(err)
	}
	if request.DatDir == "" {
		return encodeInvokeNoDataResponse(errors.New("missing datDir"))
	}
	err = geo.CountGeoData(request.DatDir, request.Name, request.GeoType)
	return encodeInvokeNoDataResponse(err)
}

func invokePingBatch(payload json.RawMessage) string {
	request, err := decodePayload[PingBatchRequest](payload)
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}

	configs := make([]xray.PingBatchItem, len(request.Configs))
	for i, config := range request.Configs {
		configs[i] = xray.PingBatchItem{
			XrayJSON:    config.XrayJson,
			OutboundTag: config.OutboundTag,
		}
	}

	results, err := xray.PingBatch(
		configs,
		request.Timeout,
		request.URL,
	)
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}

	responseResults := make([]PingBatchItemResponse, len(results))
	for i, result := range results {
		responseResults[i] = PingBatchItemResponse{
			Success: result.Success,
			Delay:   result.Delay,
			Error:   result.Error,
		}
	}
	return encodeInvokeResponse(&PingBatchResponse{Results: responseResults}, nil)
}

func invokeTestXray(payload json.RawMessage) string {
	request, err := decodePayload[TestXrayRequest](payload)
	if err != nil {
		return encodeInvokeNoDataResponse(err)
	}
	err = xray.TestXray(request.XrayJson)
	return encodeInvokeNoDataResponse(err)
}

func invokeRunXray(payload json.RawMessage) string {
	request, err := decodePayload[RunXrayRequest](payload)
	if err != nil {
		return encodeInvokeNoDataResponse(err)
	}
	err = xray.RunXray(request.XrayJson)
	return encodeInvokeNoDataResponse(err)
}
