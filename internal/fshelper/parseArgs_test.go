package fshelper

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ParsePath must accumulate one error per rejected tgz archive, not just keep the last one.
func TestParsePathAccumulatesTgzErrors(t *testing.T) {
	dir := t.TempDir()
	first := filepath.Join(dir, "takeout-001.tgz")
	second := filepath.Join(dir, "takeout-002.tgz")
	for _, f := range []string{first, second} {
		if err := os.WriteFile(f, []byte("dummy"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	_, err := ParsePath([]string{first, second})
	if err == nil {
		t.Fatal("expected an error for tgz archives, got nil")
	}
	for _, name := range []string{"takeout-001.tgz", "takeout-002.tgz"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error should mention %s, got: %v", name, err)
		}
	}
}
