package main

import (
	"C"

	. "github.com/xtls/libxray"
)

func main() {}

//export CGoInitDns
func CGoInitDns(base64Text *C.char) *C.char {
	text := C.GoString(base64Text)
	return C.CString(InitDns(text))
}

//export CGoResetDns
func CGoResetDns() *C.char {
	return C.CString(ResetDns())
}

//export CGoRunXrayFromJSON
func CGoRunXrayFromJSON(base64Text *C.char) *C.char {
	text := C.GoString(base64Text)
	return C.CString(RunXrayFromJSON(text))
}

//export CGoGetFreePorts
func CGoGetFreePorts(count int) *C.char {
	return C.CString(GetFreePorts(count))
}

//export CGoConvertShareLinksToXrayJson
func CGoConvertShareLinksToXrayJson(base64Text *C.char) *C.char {
	text := C.GoString(base64Text)
	return C.CString(ConvertShareLinksToXrayJson(text))
}

//export CGOConvertXrayJsonToShareLinks
func CGOConvertXrayJsonToShareLinks(base64Text *C.char) *C.char {
	text := C.GoString(base64Text)
	return C.CString(ConvertXrayJsonToShareLinks(text))
}

//export CGoCountGeoData
func CGoCountGeoData(base64Text *C.char) *C.char {
	text := C.GoString(base64Text)
	return C.CString(CountGeoData(text))
}

//export CGoThinGeoData
func CGoThinGeoData(base64Text *C.char) *C.char {
	text := C.GoString(base64Text)
	return C.CString(ThinGeoData(text))
}

//export CGoReadGeoFiles
func CGoReadGeoFiles(base64Text *C.char) *C.char {
	text := C.GoString(base64Text)
	return C.CString(ReadGeoFiles(text))
}

//export CGoPing
func CGoPing(base64Text *C.char) *C.char {
	text := C.GoString(base64Text)
	return C.CString(Ping(text))
}

//export CGoPingTCP
func CGoPingTCP(base64Text *C.char) *C.char {
	text := C.GoString(base64Text)
	return C.CString(PingTCP(text))
}

//export CGoConnect
func CGoConnect(base64Text *C.char) *C.char {
	text := C.GoString(base64Text)
	return C.CString(Connect(text))
}

//export CGoPingFromJSON
func CGoPingFromJSON(base64Text *C.char) *C.char {
	text := C.GoString(base64Text)
	return C.CString(PingFromJSON(text))
}

//export CGoPingTCPFromJSON
func CGoPingTCPFromJSON(base64Text *C.char) *C.char {
	text := C.GoString(base64Text)
	return C.CString(PingTCPFromJSON(text))
}

//export CGoConnectFromJSON
func CGoConnectFromJSON(base64Text *C.char) *C.char {
	text := C.GoString(base64Text)
	return C.CString(ConnectFromJSON(text))
}

//export CGoQueryStats
func CGoQueryStats(base64Text *C.char) *C.char {
	text := C.GoString(base64Text)
	return C.CString(QueryStats(text))
}

//export CGoTestXray
func CGoTestXray(base64Text *C.char) *C.char {
	text := C.GoString(base64Text)
	return C.CString(TestXray(text))
}

//export CGoRunXray
func CGoRunXray(base64Text *C.char) *C.char {
	text := C.GoString(base64Text)
	return C.CString(RunXray(text))
}

//export CGoStopXray
func CGoStopXray() *C.char {
	return C.CString(StopXray())
}

//export CGoXrayVersion
func CGoXrayVersion() *C.char {
	return C.CString(XrayVersion())
}
