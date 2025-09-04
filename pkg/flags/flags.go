package flags

// Package flags provides access to the application's command-line flags.

// These variables will be set from the cmd package
var (
	OutputDir  string
	RunName    string
	DebugMode  bool
	DryRunMode bool
)

// GetDebugMode returns whether debug mode is enabled
func GetDebugMode() bool {
	return DebugMode
}

// GetDryRunMode returns whether dry-run mode is enabled
func GetDryRunMode() bool {
	return DryRunMode
}

// GetOutputDir returns the configured output directory
func GetOutputDir() string {
	return OutputDir
}

// GetRunName returns the specified run name
func GetRunName() string {
	return RunName
}

// SetFlags sets all the flag values
func SetFlags(output string, run string, debug bool, dryRun bool) {
	OutputDir = output
	RunName = run
	DebugMode = debug
	DryRunMode = dryRun
}
