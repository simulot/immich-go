package fakefs

import (
	"fmt"
	"io/fs"
	"testing"
	"time"
)

func Test_readFileLine(t *testing.T) {
	type args struct {
		l          string
		dateFormat string
	}
	tests := []struct {
		name        string
		args        args
		wantName    string
		wantModTime time.Time
		wantSize    int64
	}{
		{
			name: "simulot",
			args: args{
				l:          "   145804  2024-05-25 22:15   Takeout/Google Photos/🇵🇹 Lisbonne ❤️ en famille 👨‍👩‍👦‍👦/😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋😛.jpg",
				dateFormat: "2006-01-02 15:04",
			},
			wantName:    "Takeout/Google Photos/🇵🇹 Lisbonne ❤️ en famille 👨‍👩‍👦‍👦/😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋😛.jpg",
			wantSize:    145804,
			wantModTime: time.Date(2024, 5, 25, 22, 15, 0, 0, time.Local),
		},
		{
			name: "pixil",
			args: args{
				l:          "   197486  07-19-2023 23:53   Takeout/Google Photos/2011 - Omaha Zoo/IMG_20110702_153447.jpg",
				dateFormat: "01-02-2006 15:04",
			},
			wantName:    "Takeout/Google Photos/2011 - Omaha Zoo/IMG_20110702_153447.jpg",
			wantSize:    197486,
			wantModTime: time.Date(2023, 7, 19, 23, 53, 0, 0, time.Local),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotSize, gotModTime := readFileLine(tt.args.l, tt.args.dateFormat)

			if gotName != tt.wantName {
				t.Errorf("readFileLine() got = %v, want %v", gotName, tt.wantName)
			}
			if gotSize != tt.wantSize {
				t.Errorf("readFileLine() got = %v, want %v", gotSize, tt.wantSize)
			}
			if !gotModTime.Equal(tt.wantModTime) {
				t.Errorf("readFileLine() got = %v, want %v", gotModTime, tt.wantModTime)
			}
		})
	}
}

type NameFS interface {
	Name() string
}

func TestFakeFS(t *testing.T) {
	fsyss, err := ScanFileList("TESTDATA/one.lst", "2006-01-02 15:04")
	if err != nil {
		t.Error(err)
		return
	}

	for _, fsys := range fsyss {
		if fsys, ok := fsys.(NameFS); ok {
			fmt.Println(fsys.Name())
		}
		err := fs.WalkDir(fsys, ".",
			func(name string, d fs.DirEntry, err error) error {
				if err != nil {
					return err
				}
				f, err := d.Info()
				if f != nil {
					fmt.Printf("%9d %s %s\n", f.Size(), f.ModTime().Format("2006-01-02 15:04"), name)
				}
				return err
			})
		if err != nil {
			t.Error(err)
			return
		}
	}
	fmt.Println()
}
