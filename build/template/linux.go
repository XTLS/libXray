//go:build linux && !android

package main

import "C"

//export CGoInitDns
func CGoInitDns(base64Text *C.char) *C.char {
	text := C.GoString(base64Text)
	return C.CString(InitDns(text))
}

//export CGoResetDns
func CGoResetDns() *C.char {
	return C.CString(ResetDns())
}

//export CGoStartTun
func CGoStartTun(base64Text *C.char) *C.char {
	text := C.GoString(base64Text)
	return C.CString(StartTun(text))
}

//export CGoStopTun
func CGoStopTun() *C.char {
	return C.CString(StopTun())
}
