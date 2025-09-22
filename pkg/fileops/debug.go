package fileops

import "github.com/schnurbe/revio-copy/pkg/logging"

func debugf(format string, args ...interface{}) { logging.Debugf("fileops: "+format, args...) }
