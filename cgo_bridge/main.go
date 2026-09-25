package main

/*
#include <stdlib.h>
*/
import "C"

import (
	"os"
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

// CGoSetEnv sets an environment variable inside the Go runtime.
//
// The Go runtime snapshots environ when the c-archive image initializes, so a
// libc setenv() performed later by the host application is never visible to
// os.LookupEnv. That makes xray-core's env-based knobs unreachable from a
// non-Go host — most importantly platform.TunFdKey ("xray.tun.fd"), which
// proxy/tun/tun_darwin.go reads to adopt an existing tun descriptor.
//
// On iOS the descriptor only exists after NEPacketTunnelProvider.startTunnel,
// long after the snapshot, so the "iOS: use provided fd from NetworkExtension"
// branch can never be taken: NewTun falls through to creating its own utun via
// an AF_SYSTEM socket, which the app sandbox denies, and the inbound fails with
// "app/proxyman/inbound: failed to start proxy > operation not permitted".
//
//export CGoSetEnv
func CGoSetEnv(name *C.char, value *C.char) {
	_ = os.Setenv(C.GoString(name), C.GoString(value))
}
