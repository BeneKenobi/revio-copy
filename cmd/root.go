package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/schnurbe/revio-copy/pkg/flags"
	"github.com/schnurbe/revio-copy/pkg/logging"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	outputDir string
	runName   string
	debugMode bool
	dryRun    bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "revio-copy",
	Short: "Process PacBio Revio sequencing data",
	Long: `revio-copy lists runs and (optionally) copies HiFi read BAM/PBI files for PacBio Revio sequencing data.
It works in two phases: 1) discover + display metadata; 2) when an output directory is supplied, identify/copy files.`,
	// Ensure flags are synchronized and debug logging toggled.
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		updateFlags()
		if flags.GetDebugMode() {
			logging.EnableDebug()
		} else {
			logging.DisableDebug()
		}
	},
}

// checkRcloneAvailability checks if rclone is available in the system path
func checkRcloneAvailability() error {
	cmd := exec.Command("rclone", "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("rclone not found or not executable: %w", err)
	}
	versionLine := strings.Split(string(output), "\n")[0]
	logging.Debugf("rclone available: %s", versionLine)
	return nil
}

// Execute adds all child commands to the root command and sets flags appropriately.
// Execute runs the CLI. Only minimal setup is done here; heavy lifting in subcommands.
func Execute() {
	viper.AutomaticEnv()
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings
	rootCmd.PersistentFlags().StringVar(&outputDir, "output", "", "output directory for processed files")
	rootCmd.PersistentFlags().StringVar(&runName, "run", "", "specific run name to process")
	rootCmd.PersistentFlags().BoolVar(&debugMode, "debug", false, "enable debug output")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "identify files without copying")

	// Set prefix for environment variables (REVIO_RUN instead of just RUN)
	viper.SetEnvPrefix("REVIO")

	// Enable replacement of - with _ in environment variables (REVIO_DRY_RUN vs REVIO_DRY-RUN)
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Bind flags to viper
	viper.BindPFlag("output", rootCmd.PersistentFlags().Lookup("output"))
	viper.BindPFlag("run", rootCmd.PersistentFlags().Lookup("run"))
	viper.BindPFlag("debug", rootCmd.PersistentFlags().Lookup("debug"))
	viper.BindPFlag("dry-run", rootCmd.PersistentFlags().Lookup("dry-run"))
}

// updateFlags updates the flags package with the current flag values
func updateFlags() {
	outputDir = viper.GetString("output")
	runName = viper.GetString("run")
	debugMode = viper.GetBool("debug")
	dryRun = viper.GetBool("dry-run")
	flags.SetFlags(outputDir, runName, debugMode, dryRun)
}
