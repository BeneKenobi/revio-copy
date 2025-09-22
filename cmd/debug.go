package cmd

import "github.com/schnurbe/revio-copy/pkg/logging"

// debugf kept for local convenience; delegates to central logging.
func debugf(format string, args ...interface{}) { logging.Debugf(format, args...) }
