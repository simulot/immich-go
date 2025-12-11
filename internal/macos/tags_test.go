//go:build darwin
// +build darwin

package macos

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSetFinderTag(t *testing.T) {
	if !IsSupported() {
		t.Skip("macOS Finder tagging not supported")
	}

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("failed to create test file: %v", err)
	}

	if err := SetFinderTag(testFile, TagRed); err != nil {
		t.Errorf("SetFinderTag failed: %v", err)
	}
}

func TestIsSupported(t *testing.T) {
	if !IsSupported() {
		t.Error("IsSupported should return true on macOS")
	}
}
