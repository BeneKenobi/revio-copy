package copyfiles

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/schnurbe/revio-copy/pkg/fileops"
)

// FileCopier handles copying files with rclone
type FileCopier struct {
	DryRun  bool
	Verbose bool
}

// NewFileCopier creates a new FileCopier
func NewFileCopier(dryRun bool, verbose bool) *FileCopier {
	return &FileCopier{
		DryRun:  dryRun,
		Verbose: verbose,
	}
}

// CopyFileMapping copies files based on a FileMapping
func (fc *FileCopier) CopyFileMapping(mapping *fileops.FileMapping) error {
	// Create destination directory
	destDir := filepath.Dir(mapping.DestBAM)
	if !fc.DryRun {
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return fmt.Errorf("failed to create destination directory: %w", err)
		}
	}

	// Copy BAM file
	if err := fc.copyFileRclone(mapping.SourceBAM, mapping.DestBAM); err != nil {
		return fmt.Errorf("failed to copy BAM file: %w", err)
	}

	// Copy PBI file
	if err := fc.copyFileRclone(mapping.SourcePBI, mapping.DestPBI); err != nil {
		return fmt.Errorf("failed to copy PBI file: %w", err)
	}

	return nil
}

// CopyAllFileMappings copies all files in the provided file mappings
func (fc *FileCopier) CopyAllFileMappings(mappings []*fileops.FileMapping) error {
	totalFiles := len(mappings) * 2 // BAM + PBI
	completedFiles := 0

	fmt.Printf("Starting copy of %d files (%d BAM + %d PBI)...\n",
		totalFiles, len(mappings), len(mappings))

	for i, mapping := range mappings {
		fmt.Printf("\n[%d/%d] Processing biosample: %s\n",
			i+1, len(mappings), mapping.BioSample)

		err := fc.CopyFileMapping(mapping)
		if err != nil {
			fmt.Printf("Error copying files for biosample %s: %v\n",
				mapping.BioSample, err)
			continue
		}

		completedFiles += 2 // BAM + PBI
		fmt.Printf("Progress: %d/%d files completed (%.1f%%)\n",
			completedFiles, totalFiles, float64(completedFiles)/float64(totalFiles)*100)
	}

	fmt.Printf("\nCopy operation completed. %d/%d files copied successfully.\n",
		completedFiles, totalFiles)

	return nil
}

// CopyHiFiReads copies HiFi reads BAM and PBI files to the output directory
func (fc *FileCopier) CopyHiFiReads(metadataPath, biosample, outputDir string) error {
	// Determine source directory - metadata file is in the metadata subdir
	metadataDir := filepath.Dir(metadataPath)
	runDir := filepath.Dir(metadataDir) // Go up one level from metadata

	// Source directory for HiFi reads
	hifiDir := filepath.Join(runDir, "hifi_reads")

	// Check if hifi_reads directory exists
	if _, err := os.Stat(hifiDir); os.IsNotExist(err) {
		return fmt.Errorf("hifi_reads directory not found at %s", hifiDir)
	}

	// Find .bam and .bam.pbi files
	bamFiles, err := filepath.Glob(filepath.Join(hifiDir, "*.hifi_reads.bam"))
	if err != nil {
		return err
	}

	if len(bamFiles) == 0 {
		return fmt.Errorf("no HiFi BAM files found in %s", hifiDir)
	}

	// Destination directory for this sample
	destDir := filepath.Join(outputDir, fmt.Sprintf("Sample_%s", biosample))

	// Create destination directory
	if !fc.DryRun {
		if err := os.MkdirAll(destDir, 0755); err != nil {
			return err
		}
	}

	// For each BAM file, copy BAM and its PBI
	for _, bamFile := range bamFiles {
		// Get corresponding PBI file
		pbiFile := bamFile + ".pbi"
		if _, err := os.Stat(pbiFile); os.IsNotExist(err) {
			return fmt.Errorf("PBI file not found for BAM: %s", bamFile)
		}

		// Target filenames
		destBam := filepath.Join(destDir, fmt.Sprintf("%s.mod.unmapped.bam", biosample))
		destPbi := filepath.Join(destDir, fmt.Sprintf("%s.mod.unmapped.bam.pbi", biosample))

		// Copy BAM file
		if err := fc.copyFileRclone(bamFile, destBam); err != nil {
			return err
		}

		// Copy PBI file
		if err := fc.copyFileRclone(pbiFile, destPbi); err != nil {
			return err
		}
	}

	return nil
}

// copyFileRclone uses rclone to copy a file with checksum verification
func (fc *FileCopier) copyFileRclone(src, dest string) error {
	// Check if source file exists
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("source file error: %w", err)
	}

	// Get file size for display
	srcSize := srcInfo.Size()
	srcSizeMB := float64(srcSize) / (1024 * 1024)

	// Prepare rclone command
	args := []string{
		"copyto",
		"--checksum", // Verify checksums for data integrity
		"--progress", // Show progress
	}

	// Add source and destination
	args = append(args, src, dest)

	// Add --dry-run flag if necessary
	if fc.DryRun {
		args = append([]string{"--dry-run"}, args...)
	}

	// Log the operation
	if fc.DryRun {
		fmt.Printf("  [DRY RUN] Would copy: %s (%.2f MB) -> %s\n",
			filepath.Base(src), srcSizeMB, filepath.Base(dest))
		if fc.Verbose {
			fmt.Printf("  [DRY RUN] Command: rclone %s\n", strings.Join(args, " "))
		}
		return nil
	}

	// In actual copy mode
	fmt.Printf("  Copying: %s (%.2f MB) -> %s\n",
		filepath.Base(src), srcSizeMB, filepath.Base(dest))

	// Execute rclone command
	cmd := exec.Command("rclone", args...)

	// Always show output for progress monitoring
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("rclone error: %w", err)
	}

	// Verify destination file exists and has correct size
	if !fc.DryRun {
		destInfo, err := os.Stat(dest)
		if err != nil {
			return fmt.Errorf("destination verification failed: %w", err)
		}

		if destInfo.Size() != srcInfo.Size() {
			return fmt.Errorf("size mismatch: source=%d bytes, destination=%d bytes",
				srcInfo.Size(), destInfo.Size())
		}

		fmt.Printf("  ✓ Copy successful and verified (%.2f MB)\n", srcSizeMB)
	}

	return nil
}
