package immich

/*
type testFile struct {
	*bytes.Buffer
}

func (t testFile) Close() error {
	t.Buffer.Reset()
	return nil
}

func (t testFile) Stat() (fs.FileInfo, error) {
	return t, nil
}

func (t testFile) Name() string       { return "testfile" }            // base name of the file
func (t testFile) Size() int64        { return int64(t.Buffer.Len()) } // length in bytes for regular files; system-dependent for others
func (t testFile) Mode() fs.FileMode  { return fs.FileMode(0) }        // file mode bits
func (t testFile) ModTime() time.Time { return time.Time{} }           // modification time
func (t testFile) IsDir() bool        { return false }                 // abbreviation for Mode().IsDir()
func (t testFile) Sys() any           { return nil }                   // underlying data source (can return nil)

func Test_localAssetFile_Read(t *testing.T) {
	type fields struct {
		f         fs.File
		preloaded []byte
		pos       int
	}
	type args struct {
		b []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			name: "empty",
			fields: fields{
				f: testFile{Buffer: bytes.NewBufferString("")},
			},
			args: args{
				b: []byte{},
			},
			want:    0,
			wantErr: false,
		},
		{
			name: "readall",
			fields: fields{
				f: testFile{Buffer: bytes.NewBufferString("Lorem ipsum dolor sit amet, consectetur adipiscing elit, sed do eiusmod tempor incididunt ut labore et dolore magna aliqua. Ut enim ad minim veniam, quis nostrud exercitation ullamco laboris nisi ut aliquip ex ea commodo consequat. Duis aute irure dolor in reprehenderit in voluptate velit esse cillum dolore eu fugiat nulla pariatur. Excepteur sint occaecat cupidatat non proident, sunt in culpa qui officia deserunt mollit anim id est laborum.")},
			},
			args: args{
				b: []byte{},
			},
			want:    100,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &localAssetFile{
				f:         tt.fields.f,
				preloaded: tt.fields.preloaded,
				pos:       tt.fields.pos,
			}
			got, err := f.Read(tt.args.b)
			if (err != nil) != tt.wantErr {
				t.Errorf("localAssetFile.Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("localAssetFile.Read() = %v, want %v", got, tt.want)
			}
		})
	}
}
*/
