//go:build ios

package memory

import (
	"runtime/debug"
	"sync"
	"time"
)

const (
	interval = 1
	// 30M
	maxMemory = 30 * 1024 * 1024
)

var initForceFreeOnce sync.Once

func forceFree(interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			debug.FreeOSMemory()
		}
	}()
}

func InitForceFree() {
	initForceFreeOnce.Do(func() {
		debug.SetGCPercent(10)
		debug.SetMemoryLimit(maxMemory)
		duration := time.Duration(interval) * time.Second
		forceFree(duration)
	})
}
