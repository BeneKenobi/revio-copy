package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

// Version information
var (
	Version = "0.1.0"
	Commit  = "none"
	Date    = "unknown"
)

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print the version number",
	Long:  `Print the version, commit, and build date of revio-copy.`,
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("revio-copy v%s\n", Version)
		fmt.Printf("Commit: %s\n", Commit)
		fmt.Printf("Build Date: %s\n", Date)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
