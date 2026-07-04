package libXray

import (
	"encoding/json"
	"errors"

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

func Invoke(requestJSON string) string {
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
	case LibXrayMethodCountGeoData:
		return invokeCountGeoData(request.Payload)
	case LibXrayMethodPing:
		return invokePing(request.Payload)
	case LibXrayMethodTestXray:
		return invokeTestXray(request.Payload)
	case LibXrayMethodRunXray:
		return invokeRunXray(request.Payload)
	case LibXrayMethodRunXrayFromJson:
		return invokeRunXrayFromJSON(request.Payload)
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
	if version == 0 || version == 1 {
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
		return `{"success":false,"error":"failed to encode response"}`
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
	xrayJson, err := share.ConvertShareLinksToXrayJson(request.Text)
	return encodeInvokeResponse(xrayJson, err)
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

func invokePing(payload json.RawMessage) string {
	request, err := decodePayload[PingRequest](payload)
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	delay, err := xray.Ping(request.ConfigPath, request.Timeout, request.URL, request.Proxy)
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	return encodeInvokeResponse(&PingResponse{Delay: delay}, nil)
}

func invokeTestXray(payload json.RawMessage) string {
	request, err := decodePayload[RunXrayRequest](payload)
	if err != nil {
		return encodeInvokeNoDataResponse(err)
	}
	err = xray.TestXray(request.ConfigPath)
	return encodeInvokeNoDataResponse(err)
}

func invokeRunXray(payload json.RawMessage) string {
	request, err := decodePayload[RunXrayRequest](payload)
	if err != nil {
		return encodeInvokeNoDataResponse(err)
	}
	err = xray.RunXray(request.ConfigPath)
	return encodeInvokeNoDataResponse(err)
}

func invokeRunXrayFromJSON(payload json.RawMessage) string {
	request, err := decodePayload[RunXrayFromJSONRequest](payload)
	if err != nil {
		return encodeInvokeNoDataResponse(err)
	}
	err = xray.RunXrayFromJSON(request.ConfigJSON)
	return encodeInvokeNoDataResponse(err)
}
