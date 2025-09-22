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
		// Only ensure rclone exists if we intend to copy (output provided and not dry-run) to keep 'list only' usage lightweight.
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
		fmt.Printf("Scanning for metadata files in %s...\n", rootDir)
		metadataFiles, err := metadata.FindMetadataFiles(rootDir)
		if err != nil {
			return err
		}

		if len(metadataFiles) == 0 {
			return fmt.Errorf("no metadata files found in %s", rootDir)
		}

		fmt.Printf("Found %d metadata files.\n", len(metadataFiles))

		// Debug: Print all metadata files found
		for i, file := range metadataFiles {
			debugf("metadata file %d: %s", i+1, file)
		}

		// Check if a specific run was requested
		var selectedRun *metadata.RunInfo
		runName := flags.GetRunName()

		if runName != "" {
			// Process specific run
			fmt.Printf("Looking for run: %s\n", runName)
			selectedRun, err = metadata.FindRunsByName(metadataFiles, runName)
			if err != nil {
				return err
			}

			fmt.Printf("Found run '%s' with %d biosamples\n",
				runName, selectedRun.BioSampleCount())
		} else {
			// No specific run, list available runs for selection
			allRuns, err := metadata.GetAllRuns(metadataFiles)
			if err != nil {
				return err
			}

			// Display available runs sorted by date (newest first)
			fmt.Println("Available runs (sorted by started date, newest first):")
			for i, run := range allRuns {
				dateStr := "Date unknown"
				if run.StartedDate != "" {
					dateStr = fmt.Sprintf("Started: %s", run.StartedDate)
				}

				fmt.Printf("%d. %s - %s (%d biosamples)\n",
					i+1, run.Name, dateStr, run.BioSampleCount())
			}

			// Prompt for run selection
			selected := promptForSelection("Select a run by number", len(allRuns))
			if selected < 0 {
				return fmt.Errorf("invalid selection")
			}

			selectedRun = allRuns[selected]
			fmt.Printf("Selected run: %s\n", selectedRun.Name)
		}

		// Print information about the selected run
		fmt.Printf("\nRun Details:\n")
		fmt.Printf("Run Name: %s\n", selectedRun.Name)

		// Print started date information if available
		if selectedRun.StartedDate != "" {
			fmt.Printf("Run Started: %s\n", selectedRun.StartedDate)
		}

		fmt.Printf("Number of Unique Biosamples: %d\n\n", selectedRun.BioSampleCount())

		// Print unique biosamples
		fmt.Printf("\nUnique biosamples in this run:\n")
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
			fmt.Printf("\nIdentifying files to copy...\n")

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
				fmt.Printf("Error identifying files: %v\n", err)
			} else {
				fmt.Printf("\nIdentified %d files to copy:\n", len(fileMappings))
				fmt.Println("\n=============== FILE IDENTIFICATION REPORT ===============")

				// Track totals for summary
				var totalBAMSize, totalPBISize int64
				var validFileCount, invalidFileCount int

				for i, mapping := range fileMappings {
					fmt.Printf("\n[%d] Biosample: %s\n", i+1, mapping.BioSample)

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
						fmt.Printf("      - Size: %.2f MB, Status: EXISTS\n", float64(bamSize)/(1024*1024))
					} else {
						fmt.Printf("      - Status: MISSING, Error: %v\n", bamErr)
					}

					fmt.Printf("    Source PBI: %s\n", mapping.SourcePBI)
					if pbiExists {
						fmt.Printf("      - Size: %.2f MB, Status: EXISTS\n", float64(pbiSize)/(1024*1024))
					} else {
						fmt.Printf("      - Status: MISSING, Error: %v\n", pbiErr)
					}

					// Print destination file information
					fmt.Printf("    Destination BAM: %s\n", mapping.DestBAM)
					fmt.Printf("    Destination PBI: %s\n", mapping.DestPBI)

					// Check if destination directory exists
					destDir := filepath.Dir(mapping.DestBAM)
					if _, err := os.Stat(destDir); os.IsNotExist(err) {
						fmt.Printf("    Destination directory does not exist: %s\n", destDir)
					}
				}

				// Print summary statistics
				fmt.Println("\n=============== SUMMARY ===============")
				fmt.Printf("Total files identified: %d (%d BAM + %d PBI files)\n",
					len(fileMappings)*2, len(fileMappings), len(fileMappings))
				fmt.Printf("Valid files found: %d\n", validFileCount)
				fmt.Printf("Missing files: %d\n", invalidFileCount)
				fmt.Printf("Total data size: %.2f GB (BAM: %.2f GB, PBI: %.2f GB)\n",
					float64(totalBAMSize+totalPBISize)/(1024*1024*1024),
					float64(totalBAMSize)/(1024*1024*1024),
					float64(totalPBISize)/(1024*1024*1024))
				fmt.Println("========================================")

				// If files are identified and there are no missing files, proceed with copying
				if len(fileMappings) > 0 && invalidFileCount == 0 {
					// Check if we're in dry-run mode
					dryRunMode := flags.GetDryRunMode()
					verboseMode := flags.GetDebugMode()

					if dryRunMode {
						fmt.Println("\n[DRY RUN] Copy operations will be simulated but not executed")
					} else {
						fmt.Println("\nProceeding with file copying...")
					}

					// Create file copier and perform copy
					copier := copyfiles.NewFileCopier(dryRunMode, verboseMode)
					err := copier.CopyAllFileMappings(fileMappings)

					if err != nil {
						fmt.Printf("\nError during file copying: %v\n", err)
					} else if dryRunMode {
						fmt.Println("\n[DRY RUN] Copy simulation completed successfully.")
						fmt.Println("Run without --dry-run flag to perform actual copying.")
					} else {
						fmt.Println("\nAll files copied successfully!")
					}
				} else if invalidFileCount > 0 {
					fmt.Println("\nCannot proceed with copying due to missing source files.")
					fmt.Println("Please check the file identification report above.")
				}
			}
		} else {
			fmt.Printf("\nUse --output flag to identify files for copying\n")
		}

		if flags.GetDryRunMode() {
			fmt.Printf("\nFile identification and dry-run complete.\n")
		} else {
			fmt.Printf("\nFile processing complete.\n")
		}

		return nil
	},
}

// promptForSelection prompts the user to select an option by number
func promptForSelection(prompt string, max int) int {
	reader := bufio.NewReader(os.Stdin)
	for {
		fmt.Printf("%s (1-%d): ", prompt, max)
		input, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Error reading input:", err)
			return -1
		}

		// Trim whitespace and convert to int
		input = strings.TrimSpace(input)
		selected, err := strconv.Atoi(input)
		if err != nil || selected < 1 || selected > max {
			fmt.Printf("Please enter a number between 1 and %d\n", max)
			continue
		}

		return selected - 1 // Convert to zero-based index
	}
}

func init() { rootCmd.AddCommand(processCmd) }
