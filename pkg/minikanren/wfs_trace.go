package minikanren

import (
	"log"
	"os"
	"sync/atomic"
)

// Lightweight, opt-in tracing for WFS/negation synchronization paths.
// Enable by setting env var GOKANDO_WFS_TRACE=1 or by setting
// engine.config.DebugWFS=true (the latter flips the global flag at engine
// construction time).

var wfsTraceEnabled atomic.Bool

func init() {
	if os.Getenv("GOKANDO_WFS_TRACE") == "1" {
		wfsTraceEnabled.Store(true)
	}
}

func enableWFSTrace()  { wfsTraceEnabled.Store(true) }
func disableWFSTrace() { wfsTraceEnabled.Store(false) }

func wfsTracef(format string, args ...any) {
	if !wfsTraceEnabled.Load() {
		return
	}
	log.Printf("[WFS] "+format, args...)
}
