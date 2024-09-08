package cliflags

import "testing"

func TestStringList_Include(t *testing.T) {
	tests := []struct {
		name string
		sl   ExtensionList
		ext  string
		want bool
	}{
		{
			name: "empty",
			sl:   ExtensionList{},
			ext:  ".jpg",
			want: true,
		},
		{
			name: ".jpg",
			sl:   ExtensionList{".jpg"},
			ext:  ".JPG",
			want: true,
		},
		{
			name: ".jpg but .heic",
			sl:   ExtensionList{".jpg"},
			ext:  ".heic",
			want: false,
		},
		{
			name: ".jpg,.mp4,.mov with .mov",
			sl:   ExtensionList{".jpg", ".mp4", ".mov"},
			ext:  ".MOV",
			want: true,
		},
		{
			name: ".jpg,.mp4,.mov with .heic",
			sl:   ExtensionList{".jpg", ".mp4", ".mov"},
			ext:  ".HEIC",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sl.Include(tt.ext); got != tt.want {
				t.Errorf("StringList.Include() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringList_Exclude(t *testing.T) {
	tests := []struct {
		name string
		sl   ExtensionList
		ext  string
		want bool
	}{
		{
			name: "empty",
			sl:   ExtensionList{},
			ext:  ".jpg",
			want: false,
		},
		{
			name: ".jpg",
			sl:   ExtensionList{".jpg"},
			ext:  ".JPG",
			want: true,
		},
		{
			name: ".jpg but .heic",
			sl:   ExtensionList{".jpg"},
			ext:  ".heic",
			want: false,
		},
		{
			name: ".jpg,.mp4,.mov with .mov",
			sl:   ExtensionList{".jpg", ".mp4", ".mov"},
			ext:  ".MOV",
			want: true,
		},
		{
			name: ".jpg,.mp4,.mov with .heic",
			sl:   ExtensionList{".jpg", ".mp4", ".mov"},
			ext:  ".HEIC",
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.sl.Exclude(tt.ext); got != tt.want {
				t.Errorf("StringList.Exclude() = %v, want %v", got, tt.want)
			}
		})
	}
}
