package files

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"reflect"
	"testing"

	"github.com/kr/pretty"
	"github.com/psanford/memfs"
	"github.com/simulot/immich-go/helpers/fileevent"
	"github.com/simulot/immich-go/helpers/namematcher"
	"github.com/simulot/immich-go/immich"
)

type inMemFS struct {
	*memfs.FS
	err error
}

func newInMemFS() *inMemFS {
	return &inMemFS{
		FS: memfs.New(),
	}
}

func (mfs *inMemFS) addFile(name string) *inMemFS {
	if mfs.err != nil {
		return mfs
	}
	dir := path.Dir(name)
	mfs.err = errors.Join(mfs.err, mfs.MkdirAll(dir, 0o777))
	mfs.err = errors.Join(mfs.err, mfs.WriteFile(name, []byte(name), 0o777))
	return mfs
}

func TestLocalAssets(t *testing.T) {
	tc := []struct {
		name     string
		fsys     fs.FS
		expected map[string]fileLinks
	}{
		{
			name: "simple",
			fsys: newInMemFS().
				addFile("root_01.jpg").
				addFile("photos/photo_01.jpg").
				addFile("photos/photo_02.cr3").
				addFile("photos/photo_03.jpg").
				addFile("photos/summer 2023/20230801-001.jpg").
				addFile("photos/summer 2023/20230801-002.jpg").
				addFile("photos/summer 2023/20230801-003.cr3").
				addFile("@eaDir/thb1.jpg").
				addFile("photos/SYNOFILE_THUMB_0001.jpg").
				addFile("photos/summer 2023/.@__thumb/thb2.jpg"),
			expected: map[string]fileLinks{
				"root_01.jpg":                         {image: "root_01.jpg"},
				"photos/photo_01.jpg":                 {image: "photos/photo_01.jpg"},
				"photos/photo_02.cr3":                 {image: "photos/photo_02.cr3"},
				"photos/photo_03.jpg":                 {image: "photos/photo_03.jpg"},
				"photos/summer 2023/20230801-001.jpg": {image: "photos/summer 2023/20230801-001.jpg"},
				"photos/summer 2023/20230801-002.jpg": {image: "photos/summer 2023/20230801-002.jpg"},
				"photos/summer 2023/20230801-003.cr3": {image: "photos/summer 2023/20230801-003.cr3"},
			},
		},
		{
			name: "motion picture",
			fsys: newInMemFS().
				addFile("motion/PXL_20210102_221126856.MP~2").
				addFile("motion/PXL_20210102_221126856.MP~2.jpg").
				addFile("motion/PXL_20210102_221126856.MP.jpg").
				addFile("motion/PXL_20210102_221126856.MP").
				addFile("motion/20231227_152817.jpg").
				addFile("motion/20231227_152817.MP4"),
			expected: map[string]fileLinks{
				"motion/PXL_20210102_221126856.MP.jpg":   {image: "motion/PXL_20210102_221126856.MP.jpg", video: "motion/PXL_20210102_221126856.MP"},
				"motion/PXL_20210102_221126856.MP~2.jpg": {image: "motion/PXL_20210102_221126856.MP~2.jpg", video: "motion/PXL_20210102_221126856.MP~2"},
				"motion/20231227_152817.jpg":             {image: "motion/20231227_152817.jpg", video: "motion/20231227_152817.MP4"},
			},
		},
		{
			name: "sidecar",
			fsys: newInMemFS().
				addFile("root_01.jpg").
				addFile("root_01.XMP").
				addFile("root_02.jpg").
				addFile("root_02.jpg.XMP").
				addFile("video_01.mp4").
				addFile("video_01.mp4.XMP").
				addFile("root_03.MP.jpg").
				addFile("root_03.MP.jpg.XMP").
				addFile("root_03.MP"),
			expected: map[string]fileLinks{
				"root_01.jpg":    {image: "root_01.jpg", sidecar: "root_01.XMP"},
				"root_02.jpg":    {image: "root_02.jpg", sidecar: "root_02.jpg.XMP"},
				"root_03.MP.jpg": {image: "root_03.MP.jpg", sidecar: "root_03.MP.jpg.XMP", video: "root_03.MP"},
				"video_01.mp4":   {video: "video_01.mp4", sidecar: "video_01.mp4.XMP"},
			},
		},
	}

	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			fsys := c.fsys
			ctx := context.Background()

			b, err := NewLocalFiles(ctx, fileevent.NewRecorder(nil, false), fsys)
			if err != nil {
				t.Error(err)
			}
			l, err := namematcher.New(`@eaDir/`, `.@__thumb`, `SYNOFILE_THUMB_*.*`)
			if err != nil {
				t.Error(err)
			}
			b.SetBannedFiles(l)
			b.SetSupportedMedia(immich.DefaultSupportedMedia)
			b.SetWhenNoDate("FILE")

			err = b.Prepare(ctx)
			if err != nil {
				t.Error(err)
			}

			results := map[string]fileLinks{}
			for a := range b.Browse(ctx) {
				links := fileLinks{}
				ext := path.Ext(a.FileName)
				if b.sm.TypeFromExt(ext) == immich.TypeImage {
					links.image = a.FileName
					if a.LivePhoto != nil {
						links.video = a.LivePhoto.FileName
					}
				} else {
					links.video = a.FileName
				}
				if a.SideCar.FileName != "" {
					links.sidecar = a.SideCar.FileName
				}
				results[a.FileName] = links
			}

			if !reflect.DeepEqual(results, c.expected) {
				t.Errorf("difference\n")
				pretty.Ldiff(t, c.expected, results)
			}
		})
	}
}
