package gp

import "testing"

func TestEditedBaseName(t *testing.T) {
	for _, tc := range []struct {
		name string
		want string
		ok   bool
	}{
		{"X-edited.jpg", "X.jpg", true},
		{"X-edited.MP4", "X.MP4", true},
		{"X-EDITED.JPG", "X.JPG", true},
		{"IMG.1234-edited.jpg", "IMG.1234.jpg", true},
		{"X.jpg", "", false},
		{"X-edited", "X", true},
		{"-edited.jpg", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := editedBaseName(tc.name)
			if got != tc.want || ok != tc.ok {
				t.Errorf("editedBaseName(%q) = %q, %v; want %q, %v", tc.name, got, ok, tc.want, tc.ok)
			}
		})
	}
}
