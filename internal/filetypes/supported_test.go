package filetypes

import "testing"

func TestRawExtensionsAreSupportedImages(t *testing.T) {
	for ext := range rawExtensions {
		if got := DefaultSupportedMedia.TypeFromExt(ext); got != TypeImage {
			t.Errorf("RAW extension %s: TypeFromExt = %q, want %q", ext, got, TypeImage)
		}
	}
}

func TestNRWIsSupported(t *testing.T) {
	for _, name := range []string{"DSCN0001.NRW", "dscn0001.nrw"} {
		if got := DefaultSupportedMedia.TypeFromName(name); got != TypeImage {
			t.Errorf("TypeFromName(%q) = %q, want %q", name, got, TypeImage)
		}
	}
	if !DefaultSupportedMedia.IsMedia(".nrw") {
		t.Error("IsMedia(.nrw) = false, want true")
	}
}
