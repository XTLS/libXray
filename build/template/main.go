package main

import "C"

func main() {}

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

//export CGoLoadGeoData
func CGoLoadGeoData(base64Text *C.char) *C.char {
	text := C.GoString(base64Text)
	return C.CString(LoadGeoData(text))
}

//export CGoPing
func CGoPing(base64Text *C.char) *C.char {
	text := C.GoString(base64Text)
	return C.CString(Ping(text))
}

//export CGoQueryStats
func CGoQueryStats(base64Text *C.char) *C.char {
	text := C.GoString(base64Text)
	return C.CString(QueryStats(text))
}

//export CGoCustomUUID
func CGoCustomUUID(base64Text *C.char) *C.char {
	text := C.GoString(base64Text)
	return C.CString(CustomUUID(text))
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
