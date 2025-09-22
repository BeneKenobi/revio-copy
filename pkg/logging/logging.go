package logging

import (
	"log"
	"os"
	"sync/atomic"
)

var debugEnabled atomic.Bool

var (
	// logger writes to stderr by default so normal CLI output can stay on stdout
	logger = log.New(os.Stderr, "", log.LstdFlags)
)

// EnableDebug turns on debug logging.
func EnableDebug() { debugEnabled.Store(true) }

// DisableDebug turns off debug logging.
func DisableDebug() { debugEnabled.Store(false) }

// Debugf logs a formatted debug line when debug is enabled.
func Debugf(format string, args ...any) {
	if debugEnabled.Load() {
		logger.Printf("DEBUG: "+format, args...)
	}
}

// Infof logs an informational message (always shown).
func Infof(format string, args ...any) {
	logger.Printf(format, args...)
}
