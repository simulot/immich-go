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

// ParsePath accumulates errors across different branches of the switch: an error
// recorded for one argument must survive a rejection recorded for a later one.
// This guards the general accumulator, not just repeated tgz rejections.
func TestParsePathAccumulatesAcrossSources(t *testing.T) {
	dir := t.TempDir()
	badZip := filepath.Join(dir, "broken.zip")
	tgz := filepath.Join(dir, "bad-archive.tgz")
	// broken.zip is not a valid zip, so zipname.OpenReader records an error and
	// continues; bad-archive.tgz is then rejected by the tgz branch.
	for _, f := range []string{badZip, tgz} {
		if err := os.WriteFile(f, []byte("not a real archive"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	_, err := ParsePath([]string{badZip, tgz})
	if err == nil {
		t.Fatal("expected errors for a bad zip and a tgz, got nil")
	}
	for _, name := range []string{"broken.zip", "bad-archive.tgz"} {
		if !strings.Contains(err.Error(), name) {
			t.Errorf("error should mention %s, got: %v", name, err)
		}
	}
}
