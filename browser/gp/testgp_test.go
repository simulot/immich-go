package gp

import (
	"context"
	"io"
	"io/fs"
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
		name string
		gen  func() []fs.FS
		want []fileResult // file name / title
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
				{name: "PXL_20230922_144936660.jpg", size: 10, title: "PXL_20230922_144936660.jpg"},
				{name: "PXL_20230922_144934440.jpg", size: 15, title: "PXL_20230922_144934440.jpg"},
				{name: "IMG_8172.jpg", size: 25, title: "IMG_8172.jpg"},
				{name: "IMG_8172.jpg", size: 52, title: "IMG_8172.jpg"},
				{name: "IMG_8172.jpg", size: 52, title: "IMG_8172.jpg"},
			}),
		},

		{
			"albumWithoutImage", albumWithoutImage,
			sortFileResult([]fileResult{
				{name: "PXL_20230922_144936660.jpg", size: 10, title: "PXL_20230922_144936660.jpg"},
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
				{name: "27_06_12 - 2.jpg", size: 24, title: "27/06/12 - 2.jpg"},
			}),
		},
		{
			"namesIssue39", namesIssue39,
			sortFileResult([]fileResult{
				{name: "Backyard_ceremony_wedding_photography_xxxxxxx_m.jpg", size: 1, title: "Backyard_ceremony_wedding_photography_xxxxxxx_magnoliastudios-371.jpg"},
				{name: "Backyard_ceremony_wedding_photography_xxxxxxx_m.jpg", size: 1, title: "Backyard_ceremony_wedding_photography_xxxxxxx_magnoliastudios-371.jpg"},
				{name: "Backyard_ceremony_wedding_photography_xxxxxxx_m(1).jpg", size: 181, title: "Backyard_ceremony_wedding_photography_xxxxxxx_magnoliastudios-181.jpg"},
				{name: "Backyard_ceremony_wedding_photography_xxxxxxx_m(1).jpg", size: 181, title: "Backyard_ceremony_wedding_photography_xxxxxxx_magnoliastudios-181.jpg"},
				{name: "Backyard_ceremony_wedding_photography_xxxxxxx_m(494).jpg", size: 494, title: "Backyard_ceremony_wedding_photography_markham_magnoliastudios-19.jpg"},
				{name: "Backyard_ceremony_wedding_photography_xxxxxxx_m(494).jpg", size: 494, title: "Backyard_ceremony_wedding_photography_markham_magnoliastudios-19.jpg"},
			}),
		},
		{
			"issue68MPFiles", issue68MPFiles,
			sortFileResult([]fileResult{
				{name: "PXL_20221228_185930354.MP", size: 1, title: "PXL_20221228_185930354.MP"},
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
		{
			"issue390WrongCount", issue390WrongCount,
			sortFileResult([]fileResult{
				{name: "image000000.gif", size: 10, title: "image000000.gif"},
				{name: "image000000.jpg", size: 20, title: "image000000.jpg"},
			}),
		},
		{
			"issue390WrongCount2", issue390WrongCount2,
			sortFileResult([]fileResult{
				{name: "IMG_0170.jpg", size: 514963, title: "IMG_0170.jpg"},
				{name: "IMG_0170.HEIC", size: 1332980, title: "IMG_0170.HEIC"},
				{name: "IMG_0170.JPG", size: 4570661, title: "IMG_0170.JPG"},
				{name: "IMG_0170.MP4", size: 6024972, title: "IMG_0170.MP4"},
				{name: "IMG_0170.HEIC", size: 4443973, title: "IMG_0170.HEIC"},
				{name: "IMG_0170.MP4", size: 2288647, title: "IMG_0170.MP4"},
				{name: "IMG_0170.JPG", size: 4570661, title: "IMG_0170.JPG"},
				{name: "IMG_0170.MP4", size: 6024972, title: "IMG_0170.MP4"},
				{name: "IMG_0170.HEIC", size: 4443973, title: "IMG_0170.HEIC"},
				{name: "IMG_0170.jpg", size: 514963, title: "IMG_0170.jpg"},
			}),
		},
	}
	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			fsys := c.gen()

			ctx := context.Background()

			log := slog.New(slog.NewTextHandler(io.Discard, nil))

			b, err := NewTakeout(ctx, fileevent.NewRecorder(log, false), immich.DefaultSupportedMedia, fsys...)
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
				if a.LivePhoto != nil {
					results = append(results, fileResult{name: path.Base(a.LivePhoto.FileName), size: a.LivePhoto.FileSize, title: a.LivePhoto.Title})
				}
			}
			results = sortFileResult(results)

			if !reflect.DeepEqual(results, c.want) {
				t.Errorf("difference\n")
				pretty.Ldiff(t, c.want, results)
			}
		})
	}
}

func TestAlbums(t *testing.T) {
	type album map[string][]fileResult
	tc := []struct {
		name string
		gen  func() []fs.FS
		want album
	}{
		{
			name: "simpleYear",
			gen:  simpleYear,
			want: album{},
		},
		{
			name: "simpleAlbum",
			gen:  simpleAlbum,
			want: album{
				"Album": sortFileResult([]fileResult{
					{name: "IMG_8172.jpg", size: 52, title: "IMG_8172.jpg"},
					{name: "PXL_20230922_144936660.jpg", size: 10, title: "PXL_20230922_144936660.jpg"},
				}),
			},
		},
		{
			name: "albumWithoutImage",
			gen:  albumWithoutImage,
			want: album{
				"Album": sortFileResult([]fileResult{
					{name: "PXL_20230922_144936660.jpg", size: 10, title: "PXL_20230922_144936660.jpg"},
				}),
			},
		},
		{
			name: "namesIssue39",
			gen:  namesIssue39,
			want: album{
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

			b, err := NewTakeout(ctx, fileevent.NewRecorder(nil, false), immich.DefaultSupportedMedia, fsys...)
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
						l := albums[al.Title]
						l = append(l, fileResult{name: path.Base(a.FileName), size: a.FileSize, title: a.Title})
						albums[al.Title] = l
					}
				}
			}

			for k, al := range albums {
				albums[k] = sortFileResult(al)
			}

			if !reflect.DeepEqual(albums, c.want) {
				t.Errorf("difference\n")
				pretty.Ldiff(t, c.want, albums)
			}
		})
	}
}

func TestArchives(t *testing.T) {
	type photo map[string]string
	type album map[string][]string
	tc := []struct {
		name              string
		gen               func() []fs.FS
		acceptMissingJSON bool
		wantLivePhotos    photo
		wantAlbum         album
		wantAsset         photo
	}{
		{
			name:      "checkLivePhoto",
			gen:       checkLivePhoto,
			wantAsset: photo{},
			wantLivePhotos: photo{
				"Motion Test/PXL_20231118_035751175.MP.jpg": "Motion Test/PXL_20231118_035751175.MP",
				"Motion test/20231227_152817.jpg":           "Motion test/20231227_152817.MP4",
			},
			wantAlbum: album{},
		},
		{
			name:      "checkLivePhotoPixil",
			gen:       checkLivePhotoPixil,
			wantAsset: photo{},
			wantLivePhotos: photo{
				"Takeout/Google Photos/2022 - Germany - Private/IMG_4573.HEIC": "Takeout/Google Photos/2022 - Germany - Private/IMG_4573.MP4",
				"Takeout/Google Photos/Photos from 2022/IMG_4573.HEIC":         "Takeout/Google Photos/Photos from 2022/IMG_4573.MP4",
				"Takeout/Google Photos/2022 - Germany/IMG_4573.HEIC":           "Takeout/Google Photos/2022 - Germany/IMG_4573.MP4",
			},
			wantAlbum: album{
				"2022 - Germany - Private": []string{"IMG_4573.HEIC"},
				"2022 - Germany":           []string{"IMG_4573.HEIC"},
			},
		},
		{
			name: "checkMissingJSON-No",
			gen:  checkMissingJSON,
			wantAsset: photo{
				"Takeout/Google Photos/Photos from 2022/IMG_4573.HEIC": "",
			},
			wantLivePhotos: photo{},
			wantAlbum:      album{},
		},
		{
			name:              "checkMissingJSON-Yes",
			gen:               checkMissingJSON,
			acceptMissingJSON: true,
			wantAsset: photo{
				"Takeout/Google Photos/Photos from 2022/IMG_4573.HEIC":          "",
				"Takeout/Google Foto/Photos from 2016/IMG-20161201-WA0035.jpeg": "",
				"Takeout/Google Photos/2022 - Germany - Private/IMG_4553.HEIC":  "",
			},
			wantLivePhotos: photo{
				"Takeout/Google Photos/2022 - Germany/IMG_1234.HEIC": "Takeout/Google Photos/2022 - Germany/IMG_1234.MP4",
			},
			wantAlbum: album{
				"2022 - Germany": []string{"IMG_1234.HEIC"},
			},
		},
		{
			name: "checkDuplicates",
			gen:  checkDuplicates,
			wantAsset: photo{
				"Takeout/Google Foto/[E&S] 2016-01-05 - Castello De Albertis e Mostra d/20160105_121621_LLS.jpg": "",
				"Takeout/Google Foto/Photos from 2016/20160105_121621_LLS.jpg":                                   "",
				"Takeout/Google Foto/2016-01-05 - _3/20160105_121621_LLS.jpg":                                    "",
			},
			wantLivePhotos: photo{},
			wantAlbum:      album{},
		},
	}
	for _, c := range tc {
		t.Run(
			c.name,
			func(t *testing.T) {
				ctx := context.Background()
				fsys := c.gen()

				b, err := NewTakeout(ctx, fileevent.NewRecorder(nil, false), immich.DefaultSupportedMedia, fsys...)
				if err != nil {
					t.Error(err)
				}
				b.SetAcceptMissingJSON(c.acceptMissingJSON)
				err = b.Prepare(ctx)
				if err != nil {
					t.Error(err)
				}

				livePhotos := photo{}
				assets := photo{}
				albums := album{}
				for a := range b.Browse(ctx) {
					if a.LivePhoto != nil {
						photo := a.FileName
						video := a.LivePhoto.FileName
						livePhotos[photo] = video
					} else {
						assets[a.FileName] = ""
					}
					for _, al := range a.Albums {
						l := albums[al.Title]
						l = append(l, path.Base(a.FileName))
						albums[al.Title] = l
					}
				}
				if !reflect.DeepEqual(assets, c.wantAsset) {
					t.Errorf("difference assets\n")
					pretty.Ldiff(t, c.wantAsset, assets)
				}
				if !reflect.DeepEqual(livePhotos, c.wantLivePhotos) {
					t.Errorf("difference LivePhotos\n")
					pretty.Ldiff(t, c.wantLivePhotos, livePhotos)
				}
				if !reflect.DeepEqual(albums, c.wantAlbum) {
					t.Errorf("difference Album\n")
					pretty.Ldiff(t, c.wantAlbum, albums)
				}
			},
		)
	}
}
