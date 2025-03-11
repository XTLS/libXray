//go:build ios

package memory

import (
	"runtime/debug"

	"time"
)

const (
	interval = 1
	// 30M
	maxMemory = 30 * 1024 * 1024
)

func forceFree(interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			debug.FreeOSMemory()
		}
	}()
}

func InitForceFree() {
	debug.SetGCPercent(10)
	debug.SetMemoryLimit(maxMemory)
	duration := time.Duration(interval) * time.Second
	forceFree(duration)
}
