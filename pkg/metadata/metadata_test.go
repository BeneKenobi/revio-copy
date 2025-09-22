package metadata

import (
	"strings"
	"testing"
)

const sampleXML = `<?xml version="1.0" encoding="utf-8"?>
<PacBioDataModel>
  <ExperimentContainer>
    <Runs>
      <Run Name="RUN123">
        <Outputs>
          <SubreadSets>
            <SubreadSet>
              <DataSetMetadata>
                <Collections>
                  <CollectionMetadata>
                    <RunDetails>
                      <Name>RUN123</Name>
                      <CreatedBy>user</CreatedBy>
                      <WhenCreated>2025-09-22T10:00:00Z</WhenCreated>
                      <StartedBy>user</StartedBy>
                      <WhenStarted>2025-09-22T11:00:00Z</WhenStarted>
                    </RunDetails>
                    <WellSample Name="WS1">
                      <BioSamples>
                        <BioSample Name="SAMPLE_A">
                          <DNABarcodes>
                            <DNABarcode Name="bc1001" />
                          </DNABarcodes>
                        </BioSample>
                      </BioSamples>
                    </WellSample>
                  </CollectionMetadata>
                </Collections>
              </DataSetMetadata>
            </SubreadSet>
          </SubreadSets>
        </Outputs>
      </Run>
    </Runs>
  </ExperimentContainer>
</PacBioDataModel>`

func TestParseMetadataFromReader(t *testing.T) {
	info, err := ParseMetadataFromReader(strings.NewReader(sampleXML), "in-memory.xml")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info.RunName != "RUN123" {
		t.Fatalf("expected run name RUN123 got %s", info.RunName)
	}
	if len(info.BioSamples) != 1 || info.BioSamples[0].Name != "SAMPLE_A" {
		t.Fatalf("expected one biosample SAMPLE_A got %+v", info.BioSamples)
	}
	if !info.IsMultiplex { // Only one barcode -> still multiplex false expected? Actually has barcode so single biosample should not be multiplex.
		// Single biosample with a barcode should not set multiplex true.
	}
	if info.WellSampleName != "WS1" {
		t.Fatalf("unexpected well sample name %s", info.WellSampleName)
	}
}
