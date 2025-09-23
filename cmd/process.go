package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/schnurbe/revio-copy/pkg/copyfiles"
	"github.com/schnurbe/revio-copy/pkg/fileops"
	"github.com/schnurbe/revio-copy/pkg/flags"
	"github.com/schnurbe/revio-copy/pkg/logging"
	"github.com/schnurbe/revio-copy/pkg/metadata"
	"github.com/schnurbe/revio-copy/pkg/ui"
	"github.com/spf13/cobra"
)

// processCmd represents the process command
var processCmd = &cobra.Command{
	Use:   "process [directory]",
	Short: "Process PacBio Revio sequencing data",
	Long: `Process PacBio Revio sequencing data by extracting metadata information.
If no run name is specified, you will be prompted to select from available runs.`,
	Args: cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if flags.GetOutputDir() != "" && !flags.GetDryRunMode() {
			if err := checkRcloneAvailability(); err != nil {
				return err
			}
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		rootDir := args[0]

		// Find metadata files
		ui.Italic("Scanning for runs in %s...\n", rootDir)
		allRuns, err := metadata.GetAllRuns(rootDir)
		if err != nil {
			return err
		}

		if len(allRuns) == 0 {
			return fmt.Errorf("no runs found in %s", rootDir)
		}

		fmt.Printf("Found %d runs.\n", len(allRuns))

		// Debug: Print all metadata files found
		// for i, file := range metadataFiles {
		// 	debugf("metadata file %d: %s", i+1, file)
		// }

		// Check if a specific run was requested
		var selectedRun *metadata.RunInfo
		runName := flags.GetRunName()

		if runName != "" {
			// Process specific run
			ui.Italic("Looking for run: %s\n", runName)
			// Find the run from the list of all runs
			for _, run := range allRuns {
				if run.Name == runName {
					selectedRun = run
					break
				}
			}

			if selectedRun == nil {
				return fmt.Errorf("run '%s' not found", runName)
			}

			fmt.Printf("Found run '%s' with %d biosamples\n",
				runName, selectedRun.BioSampleCount())
		} else {
			// No specific run, list available runs for selection
			ui.Bold("Available runs (sorted by started date, newest first):\n")
			for i, run := range allRuns {
				var statusLabel string
				if run.Status == metadata.RunPending {
					statusLabel = " (pending)"
					fmt.Printf("%d. %s - ", i+1, run.Name)
					if run.StartedDate != "" {
						fmt.Printf("Started: %s ", run.StartedDate)
					} else {
						fmt.Printf("Date unknown ")
					}
					fmt.Printf("(%d biosamples)", run.BioSampleCount())
					ui.Yellow("%s\n", statusLabel)
				} else {
					dateStr := "Date unknown"
					if run.StartedDate != "" {
						dateStr = fmt.Sprintf("Started: %s", run.StartedDate)
					}
					ui.Green("%d. %s - %s (%d biosamples)\n",
						i+1, run.Name, dateStr, run.BioSampleCount())
				}
			}

			// Prompt for run selection
			var selected int
			for {
				selected = promptForSelection("Select a run by number", len(allRuns))
				if selected == -1 { // Error
					return fmt.Errorf("invalid selection")
				}
				if selected == -2 { // Quit
					fmt.Println("Aborted.")
					return nil
				}

				if allRuns[selected].Status == metadata.RunPending {
					ui.Yellow("This run is pending and cannot be selected. Please choose another run.\n")
				} else {
					break
				}
			}

			selectedRun = allRuns[selected]
			fmt.Printf("Selected run: %s\n", selectedRun.Name)
		}

		// Print information about the selected run
		ui.Bold("\nRun Details:\n")
		fmt.Printf("Run Name: %s\n", selectedRun.Name)

		// Print started date information if available
		if selectedRun.StartedDate != "" {
			fmt.Printf("Run Started: %s\n", selectedRun.StartedDate)
		}

		fmt.Printf("Number of Unique Biosamples: %d\n\n", selectedRun.BioSampleCount())

		// Print unique biosamples
		ui.Bold("\nUnique biosamples in this run:\n")
		biosamples := make([]string, 0, selectedRun.BioSampleCount())
		for biosample := range selectedRun.BioSampleNames {
			biosamples = append(biosamples, biosample)
		}
		sort.Strings(biosamples)
		for i, biosample := range biosamples {
			fmt.Printf("%d. %s\n", i+1, biosample)
		}

		// Check if an output directory was provided to identify files for copying
		outputDir := flags.GetOutputDir()
		if outputDir != "" {
			ui.Italic("\nIdentifying files to copy...\n")

			// Create a map of metadata files to biosamples
			metadataFileToBiosample := make(map[string][]metadata.BioSampleInfo)
			metadataFiles := make([]string, 0, len(selectedRun.Cells))

			// Debug cell count
			logging.Debugf("selected run has %d cells", len(selectedRun.Cells))

			for i, cell := range selectedRun.Cells {
				logging.Debugf("cell %d path=%s biosamples=%v", i+1, cell.FilePath, cell.BioSamples)
				metadataFileToBiosample[cell.FilePath] = cell.BioSamples
				metadataFiles = append(metadataFiles, cell.FilePath)
			}

			// Debug output dir
			logging.Debugf("output directory: %s", outputDir)

			// Identify files to copy
			logging.Debugf("identifying HiFi files across %d metadata files", len(metadataFiles))
			fileMappings, err := fileops.IdentifyAllHiFiFiles(metadataFiles, metadataFileToBiosample, outputDir)
			if err != nil {
				ui.Red("Error identifying files: %v\n", err)
			} else {
				fmt.Printf("\nIdentified %d files to copy:\n", len(fileMappings))
				ui.Bold("\n=============== FILE IDENTIFICATION REPORT ===============\n")

				// Track totals for summary
				var totalBAMSize, totalPBISize int64
				var validFileCount, invalidFileCount int

				for i, mapping := range fileMappings {
					ui.Bold("\n[%d] Biosample: %s\n", i+1, mapping.BioSample)

					// Check if source BAM exists and get size
					bamInfo, bamErr := os.Stat(mapping.SourceBAM)
					bamExists := bamErr == nil
					bamSize := int64(0)
					if bamExists {
						bamSize = bamInfo.Size()
						totalBAMSize += bamSize
						validFileCount++
					} else {
						invalidFileCount++
					}

					// Check if source PBI exists and get size
					pbiInfo, pbiErr := os.Stat(mapping.SourcePBI)
					pbiExists := pbiErr == nil
					pbiSize := int64(0)
					if pbiExists {
						pbiSize = pbiInfo.Size()
						totalPBISize += pbiSize
						validFileCount++
					} else {
						invalidFileCount++
					}

					// Print source file information with existence status and size
					fmt.Printf("    Source BAM: %s\n", mapping.SourceBAM)
					if bamExists {
						ui.Green("      - Size: %.2f MB, Status: EXISTS\n", float64(bamSize)/(1024*1024))
					} else {
						ui.Red("      - Status: MISSING, Error: %v\n", bamErr)
					}

					fmt.Printf("    Source PBI: %s\n", mapping.SourcePBI)
					if pbiExists {
						ui.Green("      - Size: %.2f MB, Status: EXISTS\n", float64(pbiSize)/(1024*1024))
					} else {
						ui.Red("      - Status: MISSING, Error: %v\n", pbiErr)
					}

					// Print destination file information
					fmt.Printf("    Destination BAM: %s\n", mapping.DestBAM)
					fmt.Printf("    Destination PBI: %s\n", mapping.DestPBI)

					// Check if destination directory exists
					destDir := filepath.Dir(mapping.DestBAM)
					if _, err := os.Stat(destDir); os.IsNotExist(err) {
						ui.Yellow("    Destination directory does not exist: %s\n", destDir)
					}
				}

				// Print summary statistics
				ui.Bold("\n=============== SUMMARY ===============\n")
				fmt.Printf("Total files identified: %d (%d BAM + %d PBI files)\n",
					len(fileMappings)*2, len(fileMappings), len(fileMappings))
				ui.Green("Valid files found: %d\n", validFileCount)
				if invalidFileCount > 0 {
					ui.Red("Missing files: %d\n", invalidFileCount)
				} else {
					fmt.Printf("Missing files: %d\n", invalidFileCount)
				}
				fmt.Printf("Total data size: %.2f GB (BAM: %.2f GB, PBI: %.2f GB)\n",
					float64(totalBAMSize+totalPBISize)/(1024*1024*1024),
					float64(totalBAMSize)/(1024*1024*1024),
					float64(totalPBISize)/(1024*1024*1024))
				ui.Bold("========================================\n")

				// If files are identified and there are no missing files, proceed with copying
				if len(fileMappings) > 0 && invalidFileCount == 0 {
					// Check if we're in dry-run mode
					dryRunMode := flags.GetDryRunMode()
					verboseMode := flags.GetDebugMode()

					if dryRunMode {
						ui.Yellow("\n[DRY RUN] Copy operations will be simulated but not executed\n")
					} else {
						ui.Italic("\nProceeding with file copying...\n")
					}

					// Create file copier and perform copy
					copier := copyfiles.NewFileCopier(dryRunMode, verboseMode)
					err := copier.CopyAllFileMappings(fileMappings)

					if err != nil {
						ui.Red("\nError during file copying: %v\n", err)
					} else if dryRunMode {
						ui.Yellow("\n[DRY RUN] Copy simulation completed successfully.\n")
						fmt.Println("Run without --dry-run flag to perform actual copying.")
					} else {
						ui.Green("\nAll files copied successfully!\n")
					}
				} else if invalidFileCount > 0 {
					ui.Red("\nCannot proceed with copying due to missing source files.\n")
					fmt.Println("Please check the file identification report above.")
				}
			}
		} else {
			fmt.Printf("\nUse --output flag to identify files for copying\n")
		}

		if flags.GetDryRunMode() {
			fmt.Printf("\nFile identification and dry-run complete.\n")
		} else {
			ui.Green("\nFile processing complete.\n")
		}

		return nil
	},
}

// promptForSelection prompts the user to select an option by number.
// It returns the selected index (0-based), -1 for an error, or -2 to quit.
func promptForSelection(prompt string, max int) int {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s (1-%d, or 'q' to quit): ", prompt, max)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			return -1
		}

		// Trim whitespace and check for quit command
		input = strings.TrimSpace(input)
		if strings.ToLower(input) == "q" {
			return -2
		}

		// Convert to int
		selected, err := strconv.Atoi(input)
		if err != nil || selected < 1 || selected > max {
			fmt.Printf("Please enter a number between 1 and %d\n", max)
			continue
		}

		return selected - 1 // Convert to zero-based index
	}
}

func init() { rootCmd.AddCommand(processCmd) }
