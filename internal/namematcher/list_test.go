package namematcher

import (
	"testing"
)

func TestList_Match(t *testing.T) {
	type args struct {
		name string
		want bool
	}
	tests := []struct {
		name string
		l    List
		want []args
	}{
		{
			name: "*.*",
			want: []args{
				{"hello.world", true},
				{"/path/to/file.exe", true},
				{"/path/to/file!exe", false},
				{"/path/to/file", false},
			},
		},
		{
			name: "f?le.*",
			want: []args{
				{"hello.world", false},
				{"/path/to/file.raw", true},
				{"/path/to/fale.jpg", true},
				{"/path/to/file", false},
			},
		},
		{
			name: "f[aeiou]le.*",
			want: []args{
				{"hello.world", false},
				{"/path/to/file.raw", true},
				{"/path/to/fIle.exe", true},
				{"/path/to/fule.jpg", true},
				{"/path/to/file", false},
			},
		},
		{
			name: "file[s$].*",
			want: []args{
				{"hello.world", false},
				{"/path/to/files.raw", true},
				{"/path/to/fIleS.exe", true},
				{"/path/to/fIle$.exe", true},
				{"/path/to/fIleX.exe", false},
				{"/path/to/fule.jpg", false},
				{"/path/to/file", false},
			},
		},
		{
			name: "file$.jpg",
			want: []args{
				{"hello.world", false},
				{"/path/to/file.jpg", false},
				{"/path/to/file.jpg$", false},
				{"/path/to/file.$jpg", false},
				{"/path/to/file$.jpg", true},
				{"/path/to/file", false},
			},
		},
		{
			name: "fi(le).jpg",
			want: []args{
				{"hello.world", false},
				{"/path/to/file.jpg", false},
				{"/path/to/fi(le).jpg", true},
				{"/path/to/file.jpg", false},
			},
		},
		{
			name: `fi\*e.jpg`,
			want: []args{
				{"hello.world", false},
				{"/path/to/fi*e.jpg", true},
				{"/path/to/file.jpg", false},
			},
		},
		{
			name: "SYNOFILE_THUMB_*.*",
			want: []args{
				{"hello.world", false},
				{"/path/to/file.exe", false},
				{"/path/to/SYNOFILE_THUMB_M_000213.jpg", true},
				{"/path/to/synofile_thumb_m_000213.jpg", true},
				{"/path/to/SYNOFILE_THUMB_M/file.jpg", false},
				{"/path/to/.@__thumb/000213.jpg", false},
			},
		},
		{
			name: "@__thumb/",
			want: []args{
				{"hello.world", false},
				{"/path/to/file.exe", false},
				{"/path/to/.@__thumb/000213.jpg", true},
				{"/path/to/SYNOFILE_THUMB_M_000213.jpg", false},
			},
		},
		{
			name: "@eaDir/",
			want: []args{
				{"hello.world", false},
				{"/path/to/file.exe", false},
				{"/path/to/.@__thumb/000213.jpg", false},
				{"/path/to/SYNOFILE_THUMB_M_000213.jpg", false},
				{"/path/to/@eaDir/000213.jpg", true},
				{"/path/to/@eaDir/sub/000213.jpg", true},
				{"@eaDir/SYNOFILE_THUMB_M_000213.jpg", true},
			},
		},

		{
			name: "/._*",
			want: []args{
				{"hello.world", false},
				{"._hello.world", true},
				{"/path/to/file.exe", false},
				{"/path/to/._file.exe", true},
				{"/path/to/file", false},
				{"/path/to/PXL_20210825_041449609._exported_699_1629864935.jpg", false},
				{"PXL_20210825_041449609._exported_699_1629864935.jpg", false},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			l, err := New([]string{tt.name}...)
			if err != nil {
				t.Errorf("Error creating the list: %s", err.Error())
				return
			}
			for _, arg := range tt.want {
				if got := l.Match(arg.name); got != arg.want {
					t.Errorf("List.Match(%v) = %v, want %v", arg.name, got, arg.want)
				}
			}
		})
	}
}

func BenchmarkPatternToRe(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = patternToRe("SYNOFILE_THUMB_*.*")
	}
}
