package fileops

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/schnurbe/revio-copy/pkg/metadata"
)

// FileMapping represents mapping between source BAM/PBI files and their destinations.
type FileMapping struct {
	SourceBAM string
	SourcePBI string
	DestBAM   string
	DestPBI   string
	BioSample string
}

// IdentifyHiFiFiles identifies HiFi read BAM and PBI files for a given metadata file
// Returns the source and destination file paths without copying
// IdentifyHiFiFiles returns file mappings for a single metadata XML file + its biosamples without copying.
func IdentifyHiFiFiles(metadataPath string, biosamples []metadata.BioSampleInfo, outputDir string) ([]*FileMapping, error) {
	debugf("Processing metadata file: %s for biosamples: %v", metadataPath, biosamples)

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
	}

	var mappings []*FileMapping

	multiplexed := false
	for _, b := range biosamples { // robust detection instead of only first element
		if b.Barcode != "" {
			multiplexed = true
			break
		}
	}

	if multiplexed { // Multiplexed sample
		for _, biosampleInfo := range biosamples {
			// Find .bam files for each barcode
			barcode := strings.Split(biosampleInfo.Barcode, "--")[0]
			bamPattern := filepath.Join(hifiDir, fmt.Sprintf("*%s*.bam", barcode))
			debugf("Looking for BAM files with pattern: %s", bamPattern)

			bamFiles, err := filepath.Glob(bamPattern)
			if err != nil {
				debugf("Error globbing BAM files: %v", err)
				continue
			}

			for _, bamFile := range bamFiles {
				pbiFile := bamFile + ".pbi"
				if _, err := os.Stat(pbiFile); os.IsNotExist(err) {
					debugf("PBI file not found for BAM: %s", bamFile)
					continue
				}

				destDir := filepath.Join(outputDir, fmt.Sprintf("Sample_%s", biosampleInfo.Name))
				destBAM := filepath.Join(destDir, fmt.Sprintf("%s.mod.unmapped.bam", biosampleInfo.Name))
				destPBI := filepath.Join(destDir, fmt.Sprintf("%s.mod.unmapped.bam.pbi", biosampleInfo.Name))

				mappings = append(mappings, &FileMapping{
					SourceBAM: bamFile,
					SourcePBI: pbiFile,
					DestBAM:   destBAM,
					DestPBI:   destPBI,
					BioSample: biosampleInfo.Name,
				})
			}
		}
	} else { // Single sample
		// Find .bam files (assuming there's only one per biosample)
		bamPattern := filepath.Join(hifiDir, "*.hifi_reads.bam")
		debugf("Looking for BAM files with pattern: %s", bamPattern)

		bamFiles, err := filepath.Glob(bamPattern)
		if err != nil {
			debugf("Error globbing BAM files: %v", err)
			return nil, err
		}

		debugf("Found %d BAM files", len(bamFiles))
		if len(bamFiles) == 0 {
			debugf("No HiFi BAM files found")
			return nil, fmt.Errorf("no HiFi BAM files found in %s", hifiDir)
		}

		bamFile := bamFiles[0] // Choose first deterministically for single sample case.
		pbiFile := bamFile + ".pbi"
		if _, err := os.Stat(pbiFile); os.IsNotExist(err) {
			return nil, fmt.Errorf("PBI file not found for BAM: %s", bamFile)
		}

		biosample := biosamples[0].Name
		destDir := filepath.Join(outputDir, fmt.Sprintf("Sample_%s", biosample))
		destBAM := filepath.Join(destDir, fmt.Sprintf("%s.mod.unmapped.bam", biosample))
		destPBI := filepath.Join(destDir, fmt.Sprintf("%s.mod.unmapped.bam.pbi", biosample))

		mappings = append(mappings, &FileMapping{
			SourceBAM: bamFile,
			SourcePBI: pbiFile,
			DestBAM:   destBAM,
			DestPBI:   destPBI,
			BioSample: biosample,
		})
	}

	return mappings, nil
}

// IdentifyAllHiFiFiles identifies all HiFi files for all cells in a run
// IdentifyAllHiFiFiles iterates across metadata files to aggregate all HiFi file mappings.
func IdentifyAllHiFiFiles(cells []string, biosamples map[string][]metadata.BioSampleInfo, outputDir string) ([]*FileMapping, error) {
	var fileMappings []*FileMapping

	for _, metadataPath := range cells {
		biosampleList, ok := biosamples[metadataPath]
		if !ok {
			return nil, fmt.Errorf("biosample not found for metadata file: %s", metadataPath)
		}

		mappings, err := IdentifyHiFiFiles(metadataPath, biosampleList, outputDir)
		if err != nil {
			// Continue processing other files; caller will evaluate final result.
			debugf("warning while identifying files for %s: %v", metadataPath, err)
			continue
		}

		fileMappings = append(fileMappings, mappings...)
	}

	if len(fileMappings) == 0 {
		return nil, fmt.Errorf("no valid HiFi files identified")
	}

	return fileMappings, nil
}
