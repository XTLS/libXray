//go:build android

package log

/*
#cgo LDFLAGS: -landroid -llog

#include <android/log.h>
#include <string.h>
#include <stdlib.h>
*/
import "C"
import "unsafe"

import (
	"github.com/xtls/xray-core/common/log"
)

var ctag = C.CString("xray")

type AndroidLogger struct {
}

func (al *AndroidLogger) Handle(msg log.Message) {
	message := msg.String()
	var priority = C.ANDROID_LOG_FATAL // this value should never be used in client mode
	generalMsg, ok := msg.(*log.GeneralMessage)
	if ok {
		switch generalMsg.Severity {
		case log.Severity_Error:
			priority = C.ANDROID_LOG_ERROR
		case log.Severity_Warning:
			priority = C.ANDROID_LOG_WARN
		case log.Severity_Info:
			priority = C.ANDROID_LOG_INFO
		case log.Severity_Debug:
			priority = C.ANDROID_LOG_DEBUG
		}
	}
	cstr := C.CString(message)
	defer C.free(unsafe.Pointer(cstr))
	C.__android_log_write(C.int(priority), ctag, cstr)
}

func init() {
	common.Must(xlog.RegisterHandlerCreator(xlog.LogType_Console, func(_ xlog.LogType, _ xlog.HandlerCreatorOptions) (log.Handler, error) {
		return &AndroidLogger{}, nil
	}))
}
