package libxray

import (
	"runtime/debug"
	"time"

	"github.com/xtls/xray-core/common/platform"
)

// will be removed when 1.9.0 released

func forceFree(interval time.Duration) {
	go func() {
		for {
			time.Sleep(interval)
			debug.FreeOSMemory()
		}
	}()
}

func readForceFreeInterval() int {
	const key = "XRAY_MEMORY_FORCEFREE"
	const defaultValue = 0
	interval := platform.EnvFlag{
		Name:    key,
		AltName: platform.NormalizeEnvName(key),
	}.GetValueAsInt(defaultValue)
	return interval
}

func initForceFree(maxMemory int64) {
	debug.SetGCPercent(10)
	debug.SetMemoryLimit(maxMemory)
	interval := readForceFreeInterval()
	if interval > 0 {
		duration := time.Duration(interval) * time.Second
		forceFree(duration)
	}
}
