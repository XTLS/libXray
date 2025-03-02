//go:build ios

package nodep

import (
	"runtime"
	"runtime/debug"
	"time"
)

const (
	// Оставляем небольшой интервал для быстрого реагирования
	interval = 1
	// Возвращаем консервативный лимит для iOS
	maxMemory = 30 * 1024 * 1024
	// Снижаем порог для более агрессивной очистки
	cleanupThreshold = 0.4
)

var (
	lastGC       time.Time
	forceCleanup bool
)

func getMemStats() (uint64, uint64) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return m.Alloc, m.Sys
}

func shouldForceCleanup() bool {
	allocated, _ := getMemStats()
	return allocated > uint64(float64(maxMemory)*cleanupThreshold)
}

func forceFree(interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)

			if shouldForceCleanup() {
				runtime.GC()
				debug.FreeOSMemory()
				lastGC = time.Now()
			}
		}
	}()
}

func InitForceFree() {
	// Более агрессивный GC для iOS
	debug.SetGCPercent(10)
	debug.SetMemoryLimit(maxMemory)
	lastGC = time.Now()
	duration := time.Duration(interval) * time.Second
	forceFree(duration)
}
