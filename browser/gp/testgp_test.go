package gp

import (
	"context"
	"io"
	"log/slog"
	"path"
	"reflect"
	"testing"

	"github.com/kr/pretty"
	"github.com/simulot/immich-go/helpers/fileevent"
	"github.com/simulot/immich-go/immich"
)

func TestBrowse(t *testing.T) {
	tc := []struct {
		name    string
		gen     func() *inMemFS
		results []fileResult // file name / title
	}{
		{
			"simpleYear", simpleYear,
			sortFileResult([]fileResult{
				{name: "PXL_20230922_144936660.jpg", size: 10, title: "PXL_20230922_144936660.jpg"},
				{name: "PXL_20230922_144956000.jpg", size: 20, title: "PXL_20230922_144956000.jpg"},
			}),
		},

		{
			"simpleAlbum", simpleAlbum,
			sortFileResult([]fileResult{
				{name: "PXL_20230922_144936660.jpg", size: 10, title: "PXL_20230922_144936660.jpg"},
				{name: "PXL_20230922_144934440.jpg", size: 15, title: "PXL_20230922_144934440.jpg"},
				{name: "IMG_8172.jpg", size: 52, title: "IMG_8172.jpg"},
				{name: "IMG_8172.jpg", size: 25, title: "IMG_8172.jpg"},
			}),
		},

		{
			"albumWithoutImage", albumWithoutImage,
			sortFileResult([]fileResult{
				{name: "PXL_20230922_144936660.jpg", size: 10, title: "PXL_20230922_144936660.jpg"},
				{name: "PXL_20230922_144934440.jpg", size: 15, title: "PXL_20230922_144934440.jpg"},
			}),
		},
		{
			"namesWithNumbers", namesWithNumbers,
			sortFileResult([]fileResult{
				{name: "IMG_3479.JPG", size: 10, title: "IMG_3479.JPG"},
				{name: "IMG_3479(1).JPG", size: 12, title: "IMG_3479.JPG"},
				{name: "IMG_3479(2).JPG", size: 15, title: "IMG_3479.JPG"},
			}),
		},
		{
			"namesTruncated", namesTruncated,
			sortFileResult([]fileResult{
				{name: "😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋😛.jpg", size: 10, title: "😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋😛😝😜🤪🤨🧐🤓😎🥸🤩🥳😏😒😞😔😟😕🙁☹️😣😖😫😩🥺😢😭😤😠😡🤬🤯😳🥵🥶.jpg"},
				{name: "PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGINA.jpg", size: 40, title: "PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGINAL.jpg"},
				{name: "05yqt21kruxwwlhhgrwrdyb6chhwszi9bqmzu16w0 2.jpg", size: 25, title: "05yqt21kruxwwlhhgrwrdyb6chhwszi9bqmzu16w0 2.jpg"},
			}),
		},

		{
			"imagesWithoutJSON", imagesEditedJSON,
			sortFileResult([]fileResult{
				{name: "PXL_20220405_090123740.PORTRAIT.jpg", size: 41, title: "PXL_20220405_090123740.PORTRAIT.jpg"},
				{name: "PXL_20220405_090123740.PORTRAIT-modifié.jpg", size: 21, title: "PXL_20220405_090123740.PORTRAIT.jpg"},
			}),
		},

		{
			"titlesWithForbiddenChars", titlesWithForbiddenChars,
			sortFileResult([]fileResult{
				{name: "27_06_12 - 1.mov", size: 52, title: "27/06/12 - 1.mov"},
				{name: "27_06_12 - 1.jpg", size: 24, title: "27/06/12 - 1.jpg"},
			}),
		},
		{
			"namesIssue39", namesIssue39,
			sortFileResult([]fileResult{
				{name: "Backyard_ceremony_wedding_photography_xxxxxxx_m.jpg", size: 1, title: "Backyard_ceremony_wedding_photography_xxxxxxx_magnoliastudios-371.jpg"},
				{name: "Backyard_ceremony_wedding_photography_xxxxxxx_m(1).jpg", size: 181, title: "Backyard_ceremony_wedding_photography_xxxxxxx_magnoliastudios-181.jpg"},
				{name: "Backyard_ceremony_wedding_photography_xxxxxxx_m(494).jpg", size: 494, title: "Backyard_ceremony_wedding_photography_markham_magnoliastudios-19.jpg"},
			}),
		},
		{
			"issue68MPFiles", issue68MPFiles,
			sortFileResult([]fileResult{
				{name: "PXL_20221228_185930354.MP.jpg", size: 2, title: "PXL_20221228_185930354.MP.jpg"},
			}),
		},
		{
			"issue68LongExposure", issue68LongExposure,
			sortFileResult([]fileResult{
				{name: "PXL_20230814_201154491.LONG_EXPOSURE-01.COVER.jpg", size: 1, title: "PXL_20230814_201154491.LONG_EXPOSURE-01.COVER.jpg"},
				{name: "PXL_20230814_201154491.LONG_EXPOSURE-02.ORIGINA.jpg", size: 2, title: "PXL_20230814_201154491.LONG_EXPOSURE-02.ORIGINAL.jpg"},
			}),
		},

		{
			"issue68ForgottenDuplicates", issue68ForgottenDuplicates,
			sortFileResult([]fileResult{
				{name: "original_1d4caa6f-16c6-4c3d-901b-9387de10e528_P.jpg", size: 1, title: "original_1d4caa6f-16c6-4c3d-901b-9387de10e528_PXL_20220516_164814158.jpg"},
				{name: "original_1d4caa6f-16c6-4c3d-901b-9387de10e528_P(1).jpg", size: 2, title: "original_1d4caa6f-16c6-4c3d-901b-9387de10e528_PXL_20220516_164814158.jpg"},
			}),
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

			log := slog.New(slog.NewTextHandler(io.Discard, nil))

			b, err := NewTakeout(ctx, fileevent.NewRecorder(log), immich.DefaultSupportedMedia, fsys)
			if err != nil {
				t.Error(err)
			}

			err = b.Prepare(ctx)
			if err != nil {
				t.Error(err)
			}

			results := []fileResult{}
			for a := range b.Browse(ctx) {
				results = append(results, fileResult{name: path.Base(a.FileName), size: a.FileSize, title: a.Title})
			}
			results = sortFileResult(results)

			if !reflect.DeepEqual(results, c.results) {
				t.Errorf("difference\n")
				pretty.Ldiff(t, c.results, results)
			}
		})
	}
}

func TestAlbums(t *testing.T) {
	type album map[string][]fileResult
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
				"Album": sortFileResult([]fileResult{
					{name: "IMG_8172.jpg", size: 52, title: "IMG_8172.jpg"},
					{name: "PXL_20230922_144936660.jpg", size: 10, title: "PXL_20230922_144936660.jpg"},
				}),
			},
		},
		{
			name: "albumWithoutImage",
			gen:  albumWithoutImage,
			albums: album{
				"Album": sortFileResult([]fileResult{
					{name: "PXL_20230922_144934440.jpg", size: 15, title: "PXL_20230922_144934440.jpg"},
					{name: "PXL_20230922_144936660.jpg", size: 10, title: "PXL_20230922_144936660.jpg"},
				}),
			},
		},
		{
			name: "namesIssue39",
			gen:  namesIssue39,
			albums: album{
				"Album": sortFileResult([]fileResult{
					{name: "Backyard_ceremony_wedding_photography_xxxxxxx_m.jpg", size: 1, title: "Backyard_ceremony_wedding_photography_xxxxxxx_magnoliastudios-371.jpg"},
					{name: "Backyard_ceremony_wedding_photography_xxxxxxx_m(1).jpg", size: 181, title: "Backyard_ceremony_wedding_photography_xxxxxxx_magnoliastudios-181.jpg"},
					{name: "Backyard_ceremony_wedding_photography_xxxxxxx_m(494).jpg", size: 494, title: "Backyard_ceremony_wedding_photography_markham_magnoliastudios-19.jpg"},
				}),
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
			b, err := NewTakeout(ctx, fileevent.NewRecorder(nil), immich.DefaultSupportedMedia, fsys)
			if err != nil {
				t.Error(err)
			}
			err = b.Prepare(ctx)
			if err != nil {
				t.Error(err)
			}

			albums := album{}
			for a := range b.Browse(ctx) {
				if len(a.Albums) > 0 {
					for _, al := range a.Albums {
						l := albums[al.Name]
						l = append(l, fileResult{name: path.Base(a.FileName), size: a.FileSize, title: a.Title})
						albums[al.Name] = l
					}
				}
			}

			for k, al := range albums {
				albums[k] = sortFileResult(al)
			}

			if !reflect.DeepEqual(albums, c.albums) {
				t.Errorf("difference\n")
				pretty.Ldiff(t, c.albums, albums)
			}
		})
	}
}
