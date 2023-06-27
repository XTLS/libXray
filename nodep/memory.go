package nodep

import (
	"runtime/debug"
	"time"
)

func forceFree(interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			debug.FreeOSMemory()
		}
	}()
}

func InitForceFree(maxMemory int64, interval int) {
	debug.SetGCPercent(10)
	debug.SetMemoryLimit(maxMemory)
	if interval > 0 {
		duration := time.Duration(interval) * time.Second
		forceFree(duration)
	}
}
