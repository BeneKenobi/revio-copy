package fileops

import (
	"fmt"

	"github.com/schnurbe/revio-copy/pkg/flags"
)

// debugf prints a formatted debug message if debug mode is enabled
func debugf(format string, args ...interface{}) {
	if flags.GetDebugMode() {
		fmt.Printf("Debug [fileops]: "+format+"\n", args...)
	}
}
