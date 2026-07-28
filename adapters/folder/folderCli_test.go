package folder

import "testing"

func TestAlbumFolderModeSet(t *testing.T) {
	tests := []struct {
		value   string
		want    AlbumFolderMode
		wantErr bool
	}{
		{value: "FOLDER", want: FolderModeFolder},
		{value: "PATH", want: FolderModePath},
		{value: "TOP", want: FolderModeTop},
		{value: "NONE", want: FolderModeNone},
		{value: "top", want: FolderModeTop},
		{value: "  Top  ", want: FolderModeTop},
		{value: "TOPMOST", wantErr: true},
		{value: "", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			var m AlbumFolderMode
			err := m.Set(tt.value)
			if (err != nil) != tt.wantErr {
				t.Fatalf("Set(%q) error = %v, wantErr %v", tt.value, err, tt.wantErr)
			}
			if err == nil && m != tt.want {
				t.Errorf("Set(%q) = %q, want %q", tt.value, m, tt.want)
			}
		})
	}
}

func TestAlbumFolderModeTextRoundTrip(t *testing.T) {
	for _, want := range []AlbumFolderMode{FolderModeNone, FolderModeFolder, FolderModePath, FolderModeTop} {
		b, err := want.MarshalText()
		if err != nil {
			t.Fatalf("MarshalText(%q): %v", want, err)
		}
		var got AlbumFolderMode
		if err := got.UnmarshalText(b); err != nil {
			t.Fatalf("UnmarshalText(%q): %v", b, err)
		}
		if got != want {
			t.Errorf("round trip = %q, want %q", got, want)
		}
	}
}
