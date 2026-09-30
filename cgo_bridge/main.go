package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"unsafe"

	libXray "github.com/xtls/libxray"
)

func main() {}

//export CGoInvoke
func CGoInvoke(requestJSON *C.char) *C.char {
	text := C.GoString(requestJSON)
	return C.CString(libXray.Invoke(text))
}

//export CGoFree
func CGoFree(value *C.char) {
	C.free(unsafe.Pointer(value))
}
