package gp_test

import (
	"context"
	"immich-go/assets/gp"
	"reflect"
	"sort"
	"testing"

	"github.com/kr/pretty"
)

func TestBrowse(t *testing.T) {
	tc := []struct {
		name    string
		gen     func() *inMemFS
		results map[string]string // file name / title
	}{
		{"simpleYear", simpleYear,
			map[string]string{
				"TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144936660.jpg": "PXL_20230922_144936660.jpg",
				"TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144956000.jpg": "PXL_20230922_144956000.jpg",
			},
		},
		{"simpleAlbum", simpleAlbum,
			map[string]string{
				"TakeoutGoogle\u00a0Photos/Photos from 2023/PXL_20230922_144936660.jpg": "PXL_20230922_144936660.jpg",
				"TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144934440.jpg":      "PXL_20230922_144934440.jpg",
				"TakeoutGoogle Photos/Photos from 2023/IMG_8172.jpg":                    "IMG_8172.jpg",
				"TakeoutGoogle Photos/Photos from 2020/IMG_8172.jpg":                    "IMG_8172.jpg",
			},
		},
		{"albumWithoutImage", albumWithoutImage,
			map[string]string{
				"TakeoutGoogle\u00a0Photos/Photos from 2023/PXL_20230922_144936660.jpg": "PXL_20230922_144936660.jpg",
				"TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144934440.jpg":      "PXL_20230922_144934440.jpg",
			},
		},
		{"namesWithNumbers", namesWithNumbers,
			map[string]string{
				"TakeoutGoogle Photos/Photos from 2009/IMG_3479.JPG":    "IMG_3479.JPG",
				"TakeoutGoogle Photos/Photos from 2009/IMG_3479(1).JPG": "IMG_3479.JPG",
				"TakeoutGoogle Photos/Photos from 2009/IMG_3479(2).JPG": "IMG_3479.JPG",
			},
		},
		{"namesTruncated", namesTruncated,
			map[string]string{
				"TakeoutGoogle Photos/Photos from 2023/05yqt21kruxwwlhhgrwrdyb6chhwszi9bqmzu16w0 2.jpg":     "05yqt21kruxwwlhhgrwrdyb6chhwszi9bqmzu16w0 2.jpg",
				"TakeoutGoogle Photos/Photos from 2023/PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGINA.jpg": "PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGINAL.jpg",
				"TakeoutGoogle Photos/Photos from 2023/😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋😛.jpg":                       "😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋😛😝😜🤪🤨🧐🤓😎🥸🤩🥳😏😒😞😔😟😕🙁☹️😣😖😫😩🥺😢😭😤😠😡🤬🤯😳🥵🥶.jpg",
			},
		},
		{"imagesWithoutJSON", imagesWithoutJSON,
			map[string]string{
				"TakeoutGoogle Photos/Photos from 2023/PXL_20220405_090123740.PORTRAIT-modifié.jpg": "PXL_20220405_090123740.PORTRAIT-modifié.jpg",
				"TakeoutGoogle Photos/Photos from 2023/PXL_20220405_090123740.PORTRAIT.jpg":         "PXL_20220405_090123740.PORTRAIT.jpg",
				"TakeoutGoogle Photos/Photos from 2023/PXL_20220405_090200110.PORTRAIT-modifié.jpg": "PXL_20220405_090200110.PORTRAIT-modifié.jpg",
			},
		},
		{"titlesWithForbiddenChars", titlesWithForbiddenChars,
			map[string]string{
				"TakeoutGoogle Photos/Photos from 2012/27_06_12 - 1.mov": "27/06/12 - 1.mov",
			},
		},
	}

	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {

			fsys := c.gen()
			if fsys.err != nil {
				t.Error(fsys.err)
				return
			}
			ctx := context.Background()

			b, err := gp.NewTakeout(ctx, fsys)
			if err != nil {
				t.Error(err)
			}

			results := map[string]string{}
			for a := range b.Browse(ctx) {
				results[a.FileName] = a.Title
			}
			if !reflect.DeepEqual(results, c.results) {
				t.Errorf("difference\n")
				pretty.Ldiff(t, c.results, results)
			}
		})
	}

}

func TestAlbums(t *testing.T) {
	type key struct {
		name   string
		length int
	}
	type album map[string][]key
	tc := []struct {
		name   string
		gen    func() *inMemFS
		albums album
	}{
		{
			name:   "simpleYear",
			gen:    simpleYear,
			albums: album{},
		},
		{
			name: "simpleAlbum",
			gen:  simpleAlbum,
			albums: album{
				"Album": []key{
					{name: "IMG_8172.jpg", length: 52},
					{name: "PXL_20230922_144936660.jpg", length: 10},
				},
			},
		},
		{
			name: "albumWithoutImage",
			gen:  albumWithoutImage,
			albums: album{
				"Album": []key{
					// {name: "PXL_20230922_144934440.jpg", length: 15},
					{name: "PXL_20230922_144936660.jpg", length: 10},
				},
			},
		},
	}

	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {

			ctx := context.Background()
			fsys := c.gen()
			if fsys.err != nil {
				t.Error(fsys.err)
				return
			}
			b, err := gp.NewTakeout(ctx, fsys)
			if err != nil {
				t.Error(err)
			}
			albums := album{}
			for a := range b.Browse(ctx) {
				if len(a.Albums) > 0 {
					for _, al := range a.Albums {
						l := albums[al]
						l = append(l, key{name: a.Title, length: a.FileSize})
						albums[al] = l
					}
				}
			}

			for k, al := range albums {
				sort.Slice(al, func(i, j int) bool {
					return al[i].name < al[j].name
				})
				albums[k] = al
			}

			if !reflect.DeepEqual(albums, c.albums) {
				t.Errorf("difference\n")
				pretty.Ldiff(t, c.albums, albums)
			}

		})

	}

}
