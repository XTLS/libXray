package main

import "C"
import (
	"encoding/json"
)

func main() {}

//export CGoGetFreePorts
func CGoGetFreePorts(count int) *C.char {
	return C.CString(GetFreePorts(count))
}

//export CGoConvertShareLinksToXrayJson
func CGoConvertShareLinksToXrayJson(text *C.char) *C.char {
	p0 := C.GoString(text)
	return C.CString(ConvertShareLinksToXrayJson(p0))
}

//export CGoConvertXrayJsonToShareText
func CGoConvertXrayJsonToShareText(xray *C.char) *C.char {
	p0 := C.GoString(xray)
	return C.CString(ConvertXrayJsonToShareLinks(p0))
}

type loadGeoDataRequest struct {
	DatDir  string `json:"datDir,omitempty"`
	Name    string `json:"name,omitempty"`
	GeoType string `json:"geoType,omitempty"`
}

//export CGoLoadGeoData
func CGoLoadGeoData(request *C.char) *C.char {
	p0 := C.GoString(request)
	var req loadGeoDataRequest
	err := json.Unmarshal([]byte(p0), &req)
	if err != nil {
		return C.CString("")
	}
	return C.CString(LoadGeoData(req.DatDir, req.Name, req.GeoType))
}

type pingRequest struct {
	DatDir     string `json:"datDir,omitempty"`
	ConfigPath string `json:"configPath,omitempty"`
	Timeout    int    `json:"timeout,omitempty"`
	Url        string `json:"url,omitempty"`
	Proxy      string `json:"proxy,omitempty"`
}

//export CGoPing
func CGoPing(request *C.char) *C.char {
	p0 := C.GoString(request)
	var req pingRequest
	err := json.Unmarshal([]byte(p0), &req)
	if err != nil {
		return C.CString("")
	}
	return C.CString(Ping(req.DatDir, req.ConfigPath, req.Timeout, req.Url, req.Proxy))
}

//export CGoQueryStats
func CGoQueryStats(server *C.char) *C.char {
	p0 := C.GoString(server)
	return C.CString(QueryStats(p0))
}

//export CGoCustomUUID
func CGoCustomUUID(text *C.char) *C.char {
	p0 := C.GoString(text)
	return C.CString(CustomUUID(p0))
}

type testXrayRequest struct {
	DatDir     string `json:"datDir,omitempty"`
	ConfigPath string `json:"configPath,omitempty"`
}

//export CGoTestXray
func CGoTestXray(request *C.char) *C.char {
	p0 := C.GoString(request)
	var req testXrayRequest
	err := json.Unmarshal([]byte(p0), &req)
	if err != nil {
		return C.CString("")
	}
	return C.CString(TestXray(req.DatDir, req.ConfigPath))
}

type runXrayRequest struct {
	DatDir     string `json:"datDir,omitempty"`
	ConfigPath string `json:"configPath,omitempty"`
	MaxMemory  int64  `json:"maxMemory,omitempty"`
}

//export CGoRunXray
func CGoRunXray(request *C.char) *C.char {
	p0 := C.GoString(request)
	var req runXrayRequest
	err := json.Unmarshal([]byte(p0), &req)
	if err != nil {
		return C.CString("")
	}
	return C.CString(RunXray(req.DatDir, req.ConfigPath, req.MaxMemory))
}

//export CGoStopXray
func CGoStopXray() *C.char {
	return C.CString(StopXray())
}

//export CGoXrayVersion
func CGoXrayVersion() *C.char {
	return C.CString(XrayVersion())
}
