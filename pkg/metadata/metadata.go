package metadata

import (
	"encoding/xml"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// PacBioDataModel represents the root element of the metadata XML file
type PacBioDataModel struct {
	XMLName             xml.Name            `xml:"PacBioDataModel"`
	ExperimentContainer ExperimentContainer `xml:"ExperimentContainer"`
}

// ExperimentContainer represents the ExperimentContainer element
type ExperimentContainer struct {
	Runs Runs `xml:"Runs"`
}

// Runs represents the Runs element
type Runs struct {
	Run Run `xml:"Run"`
}

// Run represents the Run element
type Run struct {
	Name    string  `xml:"Name,attr"`
	Outputs Outputs `xml:"Outputs"`
}

// Outputs represents the Outputs element
type Outputs struct {
	SubreadSets SubreadSets `xml:"SubreadSets"`
}

// SubreadSets represents the SubreadSets element
type SubreadSets struct {
	SubreadSet SubreadSet `xml:"SubreadSet"`
}

// SubreadSet represents the SubreadSet element
type SubreadSet struct {
	DataSetMetadata DataSetMetadata `xml:"DataSetMetadata"`
}

// DataSetMetadata represents the DataSetMetadata element
type DataSetMetadata struct {
	Collections Collections `xml:"Collections"`
}

// Collections represents the Collections element
type Collections struct {
	CollectionMetadata CollectionMetadata `xml:"CollectionMetadata"`
}

// CollectionMetadata represents the CollectionMetadata element
type CollectionMetadata struct {
	RunDetails RunDetails `xml:"RunDetails"`
	WellSample WellSample `xml:"WellSample"`
}

// RunDetails represents the RunDetails element
type RunDetails struct {
	Name        string `xml:"Name"`
	CreatedBy   string `xml:"CreatedBy"`
	WhenCreated string `xml:"WhenCreated"`
	StartedBy   string `xml:"StartedBy"`
	WhenStarted string `xml:"WhenStarted"`
}

// WellSample represents the WellSample element
type WellSample struct {
	Name       string     `xml:"Name,attr"`
	BioSamples BioSamples `xml:"BioSamples"`
}

// BioSamples represents the BioSamples element
type BioSamples struct {
	XMLName   xml.Name  `xml:"BioSamples"`
	Namespace string    `xml:"xmlns,attr"`
	BioSample BioSample `xml:"BioSample"`
}

// BioSample represents the BioSample element
type BioSample struct {
	Name string `xml:"Name,attr"`
}

// MetadataInfo holds extracted metadata information
type MetadataInfo struct {
	RunName     string
	BioSample   string
	FilePath    string
	CreatedDate string
	StartedDate string
}

// ParseMetadataFile parses a metadata XML file and extracts run name and biosample
func ParseMetadataFile(filePath string) (*MetadataInfo, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	return parseMetadata(file, filePath)
}

// ParseMetadataFromReader parses metadata from an io.Reader (exported for testing)
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

	// Extract biosample
	bioSample := collectionMetadata.WellSample.BioSamples.BioSample.Name
	if bioSample == "" {
		return nil, errors.New("biosample name not found in metadata")
	}

	// Extract dates
	createdDate := runDetails.WhenCreated
	startedDate := runDetails.WhenStarted

	return &MetadataInfo{
		RunName:     runName,
		BioSample:   bioSample,
		FilePath:    filePath,
		CreatedDate: createdDate,
		StartedDate: startedDate,
	}, nil
}

// FindMetadataFiles finds all metadata XML files in the given directory and its subdirectories
// Excludes files with "preview" in their name
func FindMetadataFiles(rootDir string) ([]string, error) {
	var files []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && filepath.Base(path) == "metadata" {
			// This is a metadata directory, look for XML files
			metadataFiles, err := filepath.Glob(filepath.Join(path, "*.metadata.xml"))
			if err != nil {
				return err
			}

			// Filter out files with "preview" in their name
			for _, file := range metadataFiles {
				base := filepath.Base(file)
				// Skip files with "preview" in their name
				if !strings.Contains(strings.ToLower(base), "preview") {
					files = append(files, file)
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return files, nil
}

// FindRunsByName finds all metadata files for a specific run name
func FindRunsByName(metadataFiles []string, runName string) (*RunInfo, error) {
	runInfo := &RunInfo{
		Name:           runName,
		Cells:          []*MetadataInfo{},
		BioSampleNames: make(map[string]bool),
	}

	for _, file := range metadataFiles {
		info, err := ParseMetadataFile(file)
		if err != nil {
			continue // Skip files that can't be parsed
		}

		if info.RunName == runName {
			runInfo.Cells = append(runInfo.Cells, info)
			runInfo.BioSampleNames[info.BioSample] = true

			// Set run dates if not already set
			if runInfo.CreatedDate == "" && info.CreatedDate != "" {
				runInfo.CreatedDate = info.CreatedDate
			}
			if runInfo.StartedDate == "" && info.StartedDate != "" {
				runInfo.StartedDate = info.StartedDate
			}
		}
	}

	if len(runInfo.Cells) == 0 {
		return nil, errors.New("no metadata files found for run: " + runName)
	}

	return runInfo, nil
}

// RunInfo contains aggregated information about a run
type RunInfo struct {
	Name           string
	CreatedDate    string
	StartedDate    string
	Cells          []*MetadataInfo
	BioSampleNames map[string]bool // Used as a set to track unique biosamples
}

// BioSampleCount returns the number of unique biosamples in the run
func (r *RunInfo) BioSampleCount() int {
	return len(r.BioSampleNames)
}

// GetAllRuns gets metadata for all available runs
func GetAllRuns(metadataFiles []string) ([]*RunInfo, error) {
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
			}
			runsMap[info.RunName] = runInfo
		}

		// Add cell info and track unique biosamples
		runInfo.Cells = append(runInfo.Cells, info)
		runInfo.BioSampleNames[info.BioSample] = true
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
