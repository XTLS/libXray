package main

import "C"

func main() {}

//export CGoGetFreePorts
func CGoGetFreePorts(count int) *C.char {
	return C.CString(GetFreePorts(count))
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
