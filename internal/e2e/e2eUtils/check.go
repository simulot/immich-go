package e2eutils

import (
	"testing"

	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fileprocessor"
)

func CheckResults(t *testing.T, expectedResults map[fileevent.Code]int64, forcedJSON bool, processor *fileprocessor.FileProcessor) bool {
	r := true

	if processor != nil {
		gotResults := processor.GetEventCounts()
		for code, value := range expectedResults {
			if gotResults[code] != value {
				t.Errorf("Expected %d results for code '%s', got %d", value, code.String(), gotResults[code])
				r = false
			}
		}

		// Check asset tracking completeness
		counters := processor.GetAssetCounters()
		if counters.Pending > 0 {
			t.Errorf("Found %d pending assets that never reached final state", counters.Pending)
			r = false
		}
	}
	return r
}

// VerifyTagList checks that all assets for the given user have the expected tags
func VerifyTagList(t *testing.T, email, password string, expectedTags map[string][]string) {
	assets, err := GetAllAssets(email, password)
	if err != nil {
		t.Fatalf("failed to get assets: %v", err)
	}

	for filename, expectedTagList := range expectedTags {
		asset, exists := assets[filename]
		if !exists {
			t.Errorf("expected asset not found: %s", filename)
			continue
		}
		asset, err := GetAssetDetails(email, password, asset.ID)
		if err != nil {
			t.Errorf("failed to get details for asset %s: %v", filename, err)
			continue
		}

		if len(asset.Tags) != len(expectedTagList) {
			t.Errorf("asset %s: expected %d tags, got %d", filename, len(expectedTagList), len(asset.Tags))
		}

	TAGLIST:
		for _, expectedTag := range expectedTagList {
			for _, actualTag := range asset.Tags {
				if actualTag.Value == expectedTag {
					continue TAGLIST
				}
			}
			t.Errorf("asset %s: expected tag not found: %s (got: %v)\n\n", filename, expectedTag, asset.Tags)
		}
	}
}
