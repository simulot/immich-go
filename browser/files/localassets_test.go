package files_test

import (
	"context"
	"errors"
	"path"
	"reflect"
	"sort"
	"testing"

	"github.com/simulot/immich-go/browser"
	"github.com/simulot/immich-go/browser/files"
	"github.com/simulot/immich-go/logger"

	"github.com/kr/pretty"
	"github.com/psanford/memfs"
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
	mfs.err = errors.Join(mfs.err, mfs.MkdirAll(dir, 0777))
	mfs.err = errors.Join(mfs.err, mfs.WriteFile(name, []byte(name), 0777))
	return mfs
}

func generateFS() *inMemFS {
	return newInMemFS().
		addFile("root_01.jpg").
		addFile("photos/photo_01.jpg").
		addFile("photos/photo_02.cr3").
		addFile("photos/photo_03.jpg").
		addFile("photos/summer 2023/20230801-001.jpg").
		addFile("photos/summer 2023/20230801-002.jpg").
		addFile("photos/summer 2023/20230801-003.cr3")
}

func TestLocalAssets(t *testing.T) {
	tc := []struct {
		name     string
		expected []string
	}{
		{
			name: "all",
			expected: []string{
				"root_01.jpg",
				"photos/photo_01.jpg",
				"photos/photo_02.cr3",
				"photos/photo_03.jpg",
				"photos/summer 2023/20230801-001.jpg",
				"photos/summer 2023/20230801-002.jpg",
				"photos/summer 2023/20230801-003.cr3",
			},
		},
	}

	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			fsys := generateFS()
			if fsys.err != nil {
				t.Error(fsys.err)
				return
			}
			ctx := context.Background()

			b, err := files.NewLocalFiles(ctx, fsys, logger.NoLogger{}, &browser.Configuration{})
			if err != nil {
				t.Error(err)
			}

			results := []string{}
			for a := range b.Browse(ctx) {
				results = append(results, a.FileName)
			}
			sort.Strings(c.expected)
			sort.Strings(results)

			if !reflect.DeepEqual(results, c.expected) {
				t.Errorf("difference\n")
				pretty.Ldiff(t, c.expected, results)
			}

		})

	}
}
