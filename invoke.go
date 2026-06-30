package libXray

import (
	"encoding/json"
	"errors"
	"os"

	"github.com/xtls/libxray/geo"
	"github.com/xtls/libxray/nodep"
	"github.com/xtls/libxray/share"
	"github.com/xtls/libxray/xray"
	"github.com/xtls/xray-core/common/platform"
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
	applyEnv(request.Env)

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
		return encodeInvokeResponse("", xray.StopXray())
	case LibXrayMethodXrayVersion:
		return encodeInvokeResponse(xray.XrayVersion(), nil)
	case LibXrayMethodGetXrayState:
		return encodeInvokeResponse(xray.GetXrayState(), nil)
	default:
		return encodeInvokeResponse(nil, errors.New("unknown method"))
	}
}

func applyEnv(env *LibXrayEnvJson) {
	if env == nil {
		return
	}
	setEnvIfNotEmpty(platform.ConfigLocation, env.ConfigLocation)
	setEnvIfNotEmpty(platform.ConfdirLocation, env.ConfdirLocation)
	setEnvIfNotEmpty(platform.AssetLocation, env.AssetLocation)
	setEnvIfNotEmpty(platform.CertLocation, env.CertLocation)
	setEnvIfNotEmpty(platform.UseReadV, env.UseReadV)
	setEnvIfNotEmpty(platform.UseFreedomSplice, env.UseFreedomSplice)
	setEnvIfNotEmpty(platform.UseVmessPadding, env.UseVmessPadding)
	setEnvIfNotEmpty(platform.UseCone, env.UseCone)
	setEnvIfNotEmpty(platform.UseStrictJSON, env.UseStrictJSON)
	setEnvIfNotEmpty(platform.BufferSize, env.BufferSize)
	setEnvIfNotEmpty(platform.BrowserDialerAddress, env.BrowserDialerAddress)
	setEnvIfNotEmpty(platform.XUDPLog, env.XUDPLog)
	setEnvIfNotEmpty(platform.XUDPBaseKey, env.XUDPBaseKey)
	setEnvIfNotEmpty(platform.TunFdKey, env.TunFd)
}

func setEnvIfNotEmpty(key string, value string) {
	if value != "" {
		_ = os.Setenv(key, value)
	}
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

func invokeGetFreePorts(payload json.RawMessage) string {
	request, err := decodePayload[GetFreePortsRequest](payload)
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	ports, err := nodep.GetFreePorts(request.Count)
	if err != nil {
		return encodeInvokeResponse(nil, err)
	}
	return encodeInvokeResponse(&getFreePortsResponse{Ports: ports}, nil)
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
		return encodeInvokeResponse("", err)
	}
	links, err := share.ConvertXrayJsonToShareLinks([]byte(request.XrayJson))
	return encodeInvokeResponse(links, err)
}

func invokeCountGeoData(payload json.RawMessage) string {
	request, err := decodePayload[CountGeoDataRequest](payload)
	if err != nil {
		return encodeInvokeResponse("", err)
	}
	datDir := platform.NewEnvFlag(platform.AssetLocation).GetValue(func() string { return "" })
	if datDir == "" {
		return encodeInvokeResponse("", errors.New("missing xray.location.asset"))
	}
	err = geo.CountGeoData(datDir, request.Name, request.GeoType)
	return encodeInvokeResponse("", err)
}

func invokePing(payload json.RawMessage) string {
	request, err := decodePayload[PingRequest](payload)
	if err != nil {
		return encodeInvokeResponse(nodep.PingDelayError, err)
	}
	delay, err := xray.Ping(request.ConfigPath, request.Timeout, request.URL, request.Proxy)
	return encodeInvokeResponse(delay, err)
}

func invokeTestXray(payload json.RawMessage) string {
	request, err := decodePayload[RunXrayRequest](payload)
	if err != nil {
		return encodeInvokeResponse("", err)
	}
	err = xray.TestXray(request.ConfigPath)
	return encodeInvokeResponse("", err)
}

func invokeRunXray(payload json.RawMessage) string {
	request, err := decodePayload[RunXrayRequest](payload)
	if err != nil {
		return encodeInvokeResponse("", err)
	}
	err = xray.RunXray(request.ConfigPath)
	return encodeInvokeResponse("", err)
}

func invokeRunXrayFromJSON(payload json.RawMessage) string {
	request, err := decodePayload[RunXrayFromJSONRequest](payload)
	if err != nil {
		return encodeInvokeResponse("", err)
	}
	err = xray.RunXrayFromJSON(request.ConfigJSON)
	return encodeInvokeResponse("", err)
}
