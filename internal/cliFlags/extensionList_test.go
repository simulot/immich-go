package cliflags

import (
	"slices"
	"testing"

	"github.com/simulot/immich-go/internal/filetypes"
)

func TestSetIncludeTypeExtensionsDefaultNoop(t *testing.T) {
	var flags InclusionFlags

	flags.SetIncludeTypeExtensions()

	if len(flags.IncludedExtensions) != 0 {
		t.Fatalf("expected default IncludedExtensions to remain empty, got %v", flags.IncludedExtensions)
	}
}

func TestSetIncludeTypeExtensionsAddsVideoAndSidecar(t *testing.T) {
	var flags InclusionFlags
	flags.IncludedType = IncludeVideo

	flags.SetIncludeTypeExtensions()

	mediaExts := filetypes.MediaToExtensions()
	expected := append(
		slices.Clone(mediaExts[filetypes.TypeVideo]),
		mediaExts[filetypes.TypeSidecar]...,
	)

	for _, ext := range expected {
		if !flags.IncludedExtensions.Include(ext) {
			t.Fatalf("expected extension %q to be included, got %v", ext, flags.IncludedExtensions)
		}
	}

	if len(flags.IncludedExtensions) != len(expected) {
		t.Fatalf("expected exactly %d extensions, got %d (%v)", len(expected), len(flags.IncludedExtensions), flags.IncludedExtensions)
	}
}
