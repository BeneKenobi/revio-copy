package fileops

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// FileMapping represents mapping between source BAM/PBI files and their destinations
type FileMapping struct {
	SourceBAM string
	SourcePBI string
	DestBAM   string
	DestPBI   string
	BioSample string
}

// IdentifyHiFiFiles identifies HiFi read BAM and PBI files for a given metadata file
// Returns the source and destination file paths without copying
func IdentifyHiFiFiles(metadataPath, biosample, outputDir string) (*FileMapping, error) {
	debugf("Processing metadata file: %s for biosample: %s", metadataPath, biosample)

	// Determine source directory - metadata file is in the metadata subdir
	metadataDir := filepath.Dir(metadataPath)
	debugf("Metadata directory: %s", metadataDir)

	runDir := filepath.Dir(metadataDir) // Go up one level from metadata
	debugf("Run directory: %s", runDir)

	// Source directory for HiFi reads
	hifiDir := filepath.Join(runDir, "hifi_reads")
	debugf("HiFi reads directory: %s", hifiDir)

	// Check if hifi_reads directory exists
	if _, err := os.Stat(hifiDir); os.IsNotExist(err) {
		debugf("Error - hifi_reads directory not found")
		return nil, fmt.Errorf("hifi_reads directory not found at %s", hifiDir)
	} // Find .bam files (assuming there's only one per biosample)
	bamPattern := filepath.Join(hifiDir, "*.hifi_reads.bam")
	debugf("Looking for BAM files with pattern: %s", bamPattern)

	bamFiles, err := filepath.Glob(bamPattern)
	if err != nil {
		debugf("Error globbing BAM files: %v", err)
		return nil, err
	}

	debugf("Found %d BAM files", len(bamFiles))
	for i, file := range bamFiles {
		debugf("BAM file %d: %s", i+1, file)
	}

	if len(bamFiles) == 0 {
		debugf("No HiFi BAM files found")
		return nil, fmt.Errorf("no HiFi BAM files found in %s", hifiDir)
	}

	// In most cases there should be only one BAM file per directory
	// but let's handle the case where there might be more and find the one that matches
	// the metadata file naming pattern

	metadataBaseName := filepath.Base(metadataPath)
	metadataPrefix := strings.TrimSuffix(metadataBaseName, ".metadata.xml")
	debugf("Metadata base name: %s, prefix: %s", metadataBaseName, metadataPrefix)

	var bamFile string
	for _, file := range bamFiles {
		baseFile := filepath.Base(file)
		if strings.HasPrefix(baseFile, metadataPrefix) || len(bamFiles) == 1 {
			bamFile = file
			break
		}
	}

	if bamFile == "" {
		return nil, fmt.Errorf("could not find matching BAM file for %s", metadataPath)
	}

	// Get corresponding PBI file
	pbiFile := bamFile + ".pbi"
	if _, err := os.Stat(pbiFile); os.IsNotExist(err) {
		return nil, fmt.Errorf("PBI file not found for BAM: %s", bamFile)
	}

	// Determine destination paths
	destDir := filepath.Join(outputDir, fmt.Sprintf("Sample_%s", biosample))
	destBAM := filepath.Join(destDir, fmt.Sprintf("%s.mod.unmapped.bam", biosample))
	destPBI := filepath.Join(destDir, fmt.Sprintf("%s.mod.unmapped.bam.pbi", biosample))

	return &FileMapping{
		SourceBAM: bamFile,
		SourcePBI: pbiFile,
		DestBAM:   destBAM,
		DestPBI:   destPBI,
		BioSample: biosample,
	}, nil
}

// IdentifyAllHiFiFiles identifies all HiFi files for all cells in a run
func IdentifyAllHiFiFiles(cells []string, biosamples map[string]string, outputDir string) ([]*FileMapping, error) {
	var fileMappings []*FileMapping

	for _, metadataPath := range cells {
		biosample, ok := biosamples[metadataPath]
		if !ok {
			return nil, fmt.Errorf("biosample not found for metadata file: %s", metadataPath)
		}

		mapping, err := IdentifyHiFiFiles(metadataPath, biosample, outputDir)
		if err != nil {
			// Log the error but continue with other files
			fmt.Printf("Warning: %v\n", err)
			continue
		}

		fileMappings = append(fileMappings, mapping)
	}

	if len(fileMappings) == 0 {
		return nil, fmt.Errorf("no valid HiFi files identified")
	}

	return fileMappings, nil
}
