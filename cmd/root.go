package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/schnurbe/revio-copy/pkg/flags"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var outputDir string
var runName string
var debugMode bool
var dryRun bool

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "revio-copy",
	Short: "A tool to process PacBio Revio sequencing data",
	Long: `revio-copy is a CLI tool designed to process PacBio Revio sequencing data,
extract metadata information, and copy HiFi reads to an output directory with proper naming.`,
	// This hook runs before any command executes and ensures flags are updated
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		// Update flags package with the current flag values
		updateFlags()
	},
}

// checkRcloneAvailability checks if rclone is available in the system path
func checkRcloneAvailability() error {
	cmd := exec.Command("rclone", "version")
	output, err := cmd.CombinedOutput()

	if err != nil {
		return fmt.Errorf("rclone not found or not executable. Please install rclone and make sure it's in your PATH. Error: %w", err)
	}

	// Extract the first line of rclone version for the debug message
	versionLine := strings.Split(string(output), "\n")[0]
	debugf("rclone is available. Version: %s", versionLine)
	return nil
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	// Check if rclone is available before proceeding
	if err := checkRcloneAvailability(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}

	// Initialize viper
	viper.AutomaticEnv() // Read environment variables that match

	// Execute the root command
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
	// Use values from viper, which will include both command-line flags and environment variables
	outputDir = viper.GetString("output")
	runName = viper.GetString("run")
	debugMode = viper.GetBool("debug")
	dryRun = viper.GetBool("dry-run")

	flags.SetFlags(outputDir, runName, debugMode, dryRun)
}
