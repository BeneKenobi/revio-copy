package metadata

import (
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// PacBioDataModel represents the root element of the metadata XML file.
type PacBioDataModel struct {
	XMLName             xml.Name            `xml:"PacBioDataModel"`
	ExperimentContainer ExperimentContainer `xml:"ExperimentContainer"`
}

// ExperimentContainer represents the ExperimentContainer element.
type ExperimentContainer struct {
	Runs Runs `xml:"Runs"`
}

// Runs represents the Runs element.
type Runs struct {
	Run Run `xml:"Run"`
}

// Run represents the Run element.
type Run struct {
	Name    string  `xml:"Name,attr"`
	Outputs Outputs `xml:"Outputs"`
}

// Outputs represents the Outputs element.
type Outputs struct {
	SubreadSets SubreadSets `xml:"SubreadSets"`
}

// SubreadSets represents the SubreadSets element.
type SubreadSets struct {
	SubreadSet SubreadSet `xml:"SubreadSet"`
}

// SubreadSet represents the SubreadSet element.
type SubreadSet struct {
	DataSetMetadata DataSetMetadata `xml:"DataSetMetadata"`
}

// DataSetMetadata represents the DataSetMetadata element.
type DataSetMetadata struct {
	Collections Collections `xml:"Collections"`
}

// Collections represents the Collections element.
type Collections struct {
	CollectionMetadata CollectionMetadata `xml:"CollectionMetadata"`
}

// CollectionMetadata represents the CollectionMetadata element.
type CollectionMetadata struct {
	RunDetails RunDetails `xml:"RunDetails"`
	WellSample WellSample `xml:"WellSample"`
}

// RunDetails represents the RunDetails element.
type RunDetails struct {
	Name        string `xml:"Name"`
	CreatedBy   string `xml:"CreatedBy"`
	WhenCreated string `xml:"WhenCreated"`
	StartedBy   string `xml:"StartedBy"`
	WhenStarted string `xml:"WhenStarted"`
}

// WellSample represents the WellSample element.
type WellSample struct {
	Name       string      `xml:"Name,attr"`
	BioSamples []BioSample `xml:"BioSamples>BioSample"`
}

// BioSample represents the BioSample element.
type BioSample struct {
	Name        string       `xml:"Name,attr"`
	DNABarcodes []DNABarcode `xml:"DNABarcodes>DNABarcode"`
}

// DNABarcode represents the DNABarcode element.
type DNABarcode struct {
	Name string `xml:"Name,attr"`
}

// BioSampleInfo holds a biosample and its associated barcode.
type BioSampleInfo struct {
	Name    string
	Barcode string
}

// RunStatus represents the completion status of a run.
type RunStatus string

const (
	// RunComplete indicates that the run has finished processing.
	RunComplete RunStatus = "complete"
	// RunPending indicates that the run is still awaiting data.
	RunPending RunStatus = "pending"
)

// MetadataInfo holds extracted metadata information.
type MetadataInfo struct {
	RunName        string
	BioSamples     []BioSampleInfo // Changed to a slice of BioSampleInfo
	FilePath       string
	CreatedDate    string
	StartedDate    string
	IsMultiplex    bool
	WellSampleName string
	Status         RunStatus
}

// ParseMetadataFile parses a metadata XML file and extracts run + biosample information.
func ParseMetadataFile(filePath string) (*MetadataInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return parseMetadata(file, filePath)
}

// ParseMetadataFromReader parses metadata from an io.Reader (exported for testing).
func ParseMetadataFromReader(r io.Reader, filePath string) (*MetadataInfo, error) {
	return parseMetadata(r, filePath)
}

// parseMetadata parses metadata from an io.Reader
func parseMetadata(r io.Reader, filePath string) (*MetadataInfo, error) {
	var model PacBioDataModel
	decoder := xml.NewDecoder(r)
	if err := decoder.Decode(&model); err != nil {
		return nil, err
	}

	// Extract run details
	collectionMetadata := model.ExperimentContainer.Runs.Run.Outputs.SubreadSets.SubreadSet.DataSetMetadata.Collections.CollectionMetadata
	runDetails := collectionMetadata.RunDetails

	runName := runDetails.Name
	if runName == "" {
		return nil, errors.New("run name not found in metadata")
	}

	// Extract biosamples
	bioSamples := collectionMetadata.WellSample.BioSamples
	if len(bioSamples) == 0 {
		return nil, errors.New("no biosamples found in metadata")
	}

	var bioSampleInfos []BioSampleInfo
	for _, bs := range bioSamples {
		if len(bs.DNABarcodes) > 0 {
			for _, bc := range bs.DNABarcodes {
				bioSampleInfos = append(bioSampleInfos, BioSampleInfo{Name: bs.Name, Barcode: bc.Name})
			}
		} else {
			bioSampleInfos = append(bioSampleInfos, BioSampleInfo{Name: bs.Name, Barcode: ""})
		}
	}

	isMultiplex := len(bioSampleInfos) > 1 && bioSampleInfos[0].Barcode != ""

	// Extract dates
	createdDate := runDetails.WhenCreated
	startedDate := runDetails.WhenStarted

	return &MetadataInfo{
		RunName:        runName,
		BioSamples:     bioSampleInfos, // Store as a slice of BioSampleInfo
		FilePath:       filePath,
		CreatedDate:    createdDate,
		StartedDate:    startedDate,
		IsMultiplex:    isMultiplex,
		WellSampleName: collectionMetadata.WellSample.Name,
		Status:         RunComplete,
	}, nil
}

// FindMetadataFiles finds all metadata XML files under root (excluding previews).
func FindMetadataFiles(rootDir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil { // Propagate filesystem errors.
			return err
		}
		if !d.IsDir() {
			return nil
		}
		if filepath.Base(path) != "metadata" {
			return nil
		}
		matches, globErr := filepath.Glob(filepath.Join(path, "*.metadata.xml"))
		if globErr != nil {
			return globErr
		}
		for _, f := range matches {
			base := filepath.Base(f)
			if strings.Contains(strings.ToLower(base), "preview") {
				continue
			}
			files = append(files, f)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// FindPendingRuns finds runs that have started transferring but are not yet complete.
func FindPendingRuns(rootDir string) (map[string]*RunInfo, error) {
	pendingRuns := make(map[string]*RunInfo)

	err := filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !strings.HasPrefix(d.Name(), "Transfer_Test_") || !strings.HasSuffix(d.Name(), ".txt") {
			return nil
		}

		// Get the parent directory, which should be "metadata"
		metadataDir := filepath.Dir(path)
		if filepath.Base(metadataDir) != "metadata" {
			return nil
		}

		// Check if a metadata file already exists in the same directory
		metadataFiles, err := filepath.Glob(filepath.Join(metadataDir, "*.metadata.xml"))
		if err != nil {
			return err
		}
		if len(metadataFiles) > 0 {
			return nil // This cell is already complete
		}

		// This is a pending cell, so let's get its run name
		relPath, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		parts := strings.Split(relPath, string(os.PathSeparator))
		if len(parts) < 3 {
			return nil // Invalid path structure
		}
		runName := parts[0]

		// Create a new RunInfo if it's the first time we see this run
		if _, exists := pendingRuns[runName]; !exists {
			// Try to infer date from run name (e.g., r84297_20250922_085610)
			var startedDate string
			nameParts := strings.Split(runName, "_")
			if len(nameParts) >= 2 {
				dateStr := nameParts[1]
				if len(dateStr) == 8 {
					// Basic validation for YYYYMMDD
					year, errYear := strconv.Atoi(dateStr[0:4])
					month, errMonth := strconv.Atoi(dateStr[4:6])
					day, errDay := strconv.Atoi(dateStr[6:8])
					if errYear == nil && errMonth == nil && errDay == nil && year > 2000 && month > 0 && month <= 12 && day > 0 && day <= 31 {
						startedDate = dateStr
					}
				}
			}

			pendingRuns[runName] = &RunInfo{
				Name:        runName,
				Status:      RunPending,
				Cells:       []*MetadataInfo{},
				StartedDate: startedDate,
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return pendingRuns, nil
}

// FindRunsByName aggregates all metadata cells for a specific run name.
func FindRunsByName(rootDir string, runName string) (*RunInfo, error) {
	allRuns, err := GetAllRuns(rootDir)
	if err != nil {
		return nil, err
	}

	for _, run := range allRuns {
		if run.Name == runName {
			if run.Status == RunPending {
				return nil, errors.New("selected run is pending and cannot be processed")
			}
			return run, nil
		}
	}

	return nil, errors.New("no metadata files found for run: " + runName)
}

// RunInfo contains aggregated information about a run.
type RunInfo struct {
	Name           string
	CreatedDate    string
	StartedDate    string
	Cells          []*MetadataInfo
	BioSampleNames map[string]bool // Used as a set to track unique biosamples
	Status         RunStatus
}

// BioSampleCount returns the number of unique biosamples in the run.
func (r *RunInfo) BioSampleCount() int {
	return len(r.BioSampleNames)
}

// GetAllRuns parses and aggregates metadata for all available runs.
func GetAllRuns(rootDir string) ([]*RunInfo, error) {
	// Find all completed runs first
	metadataFiles, err := FindMetadataFiles(rootDir)
	if err != nil {
		// We can proceed without completed runs, as there might be pending ones.
	}

	runsMap := make(map[string]*RunInfo)

	for _, file := range metadataFiles {
		info, err := ParseMetadataFile(file)
		if err != nil {
			continue // Skip files that can't be parsed
		}

		// Get or create run info
		runInfo, exists := runsMap[info.RunName]
		if !exists {
			runInfo = &RunInfo{
				Name:           info.RunName,
				CreatedDate:    info.CreatedDate,
				StartedDate:    info.StartedDate,
				Cells:          []*MetadataInfo{},
				BioSampleNames: make(map[string]bool),
				Status:         RunComplete,
			}
			runsMap[info.RunName] = runInfo
		}

		// Add cell info and track unique biosamples
		runInfo.Cells = append(runInfo.Cells, info)
		for _, bs := range info.BioSamples {
			runInfo.BioSampleNames[bs.Name] = true
		}
	}

	// Find and merge pending runs
	pendingRuns, err := FindPendingRuns(rootDir)
	if err != nil {
		// Log or handle error, but don't abort if we have complete runs
	}

	for name, pendingRun := range pendingRuns {
		if _, exists := runsMap[name]; !exists {
			runsMap[name] = pendingRun
		}
	}

	if len(runsMap) == 0 {
		return nil, errors.New("no valid runs found")
	}

	// Convert map to slice for sorting
	runs := make([]*RunInfo, 0, len(runsMap))
	for _, run := range runsMap {
		runs = append(runs, run)
	}

	// Sort runs by date, newest first
	sortRunsByDate(runs)

	return runs, nil
}

// sortRunsByDate sorts runs by their started date, newest first
func sortRunsByDate(runs []*RunInfo) {
	// Sort runs by date (newest first)
	sort.Slice(runs, func(i, j int) bool {
		// If we have started dates, use them
		if runs[i].StartedDate != "" && runs[j].StartedDate != "" {
			return runs[i].StartedDate > runs[j].StartedDate
		}
		// If no started dates available, sort by name
		return runs[i].Name > runs[j].Name
	})
}
