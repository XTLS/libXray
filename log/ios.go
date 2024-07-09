//go:build ios

package log

/*
#cgo LDFLAGS: -framework CoreFoundation

#include <os/log.h>
#include <stdlib.h>
#include <string.h>
#include <CoreFoundation/CoreFoundation.h>

static os_log_t logger = NULL;

void OSLog(uint8_t type, const char *message) {
    if (logger == NULL) {
        logger = os_log_create("io.github.xtls", "xray");
    }
    os_log_with_type(logger, type, "%{public}s", message);
}
*/
import "C"

import (
	xlog "github.com/xtls/xray-core/app/log"
	"github.com/xtls/xray-core/common"
	"github.com/xtls/xray-core/common/log"
	"unsafe"
)

type iOSLogger struct {
}

func (l *iOSLogger) Handle(msg log.Message) {
	message := msg.String()
	var logType uint8 = 0x00
	generalMsg, ok := msg.(*log.GeneralMessage)
	if ok {
		switch generalMsg.Severity {
		case log.Severity_Error:
			logType = 0x10
		case log.Severity_Warning:
			logType = 0x01
		case log.Severity_Info:
			logType = 0x01
		case log.Severity_Debug:
			logType = 0x02
		}
	}
	cstr := C.CString(message)
	defer C.free(unsafe.Pointer(cstr))
	C.OSLog(C.uint8_t(logType), cstr)
}

func init() {
	common.Must(xlog.RegisterHandlerCreator(xlog.LogType_Console, func(_ xlog.LogType, _ xlog.HandlerCreatorOptions) (log.Handler, error) {
		return &iOSLogger{}, nil
	}))
}
