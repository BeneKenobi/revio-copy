package flags

// Package flags centralizes state derived from command-line flags / environment.
// Keeping the variables unexported avoids accidental mutation from other packages.

var (
	outputDir  string
	runName    string
	debugMode  bool
	dryRunMode bool
)

// GetDebugMode reports whether debug output is enabled.
func GetDebugMode() bool { return debugMode }

// GetDryRunMode reports whether copy operations should be simulated only.
func GetDryRunMode() bool { return dryRunMode }

// GetOutputDir returns the configured output directory (may be empty for list-only mode).
func GetOutputDir() string { return outputDir }

// GetRunName returns the explicitly requested run name (empty means interactive selection).
func GetRunName() string { return runName }

// SetFlags updates all internally stored flag values.
func SetFlags(output string, run string, debug bool, dryRun bool) {
	outputDir = output
	runName = run
	debugMode = debug
	dryRunMode = dryRun
}
