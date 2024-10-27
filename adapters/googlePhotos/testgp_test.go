package gp

import (
	"context"
	"io/fs"
	"path"
	"reflect"
	"testing"
	"time"

	"github.com/kr/pretty"
	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/commands/application"
	"github.com/simulot/immich-go/helpers/configuration"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/metadata"
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
				{name: "PXL_20230922_144934440.jpg", size: 15, title: "PXL_20230922_144934440.jpg"},
				{name: "IMG_8172.jpg", size: 25, title: "IMG_8172.jpg"},
				{name: "IMG_8172.jpg", size: 52, title: "IMG_8172.jpg"},
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
				{name: "27_06_12 - 2.jpg", size: 24, title: "27/06/12 - 2.jpg"},
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
			}),
		},
	}

	logFile := configuration.DefaultLogFile()
	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			fsys := c.gen()

			ctx := context.Background()
			log := application.Log{
				File:  logFile,
				Level: "INFO",
			}
			err := log.OpenLogFile()
			if err != nil {
				t.Error(err)
				return
			}
			flags := &ImportFlags{
				SupportedMedia: metadata.DefaultSupportedMedia,
				CreateAlbums:   true,
				InfoCollector:  filenames.NewInfoCollector(time.Local, metadata.DefaultSupportedMedia),
			}
			log.Logger.Info("\n\n\ntest case: " + c.name)
			recorder := fileevent.NewRecorder(log.Logger)
			b, err := NewTakeout(ctx, recorder, flags, fsys...)
			if err != nil {
				t.Error(err)
				return
			}

			gChan := b.Browse(ctx)

			results := []fileResult{}
			for g := range gChan {
				if err = g.Validate(); err != nil {
					t.Error(err)
					return
				}
				for _, a := range g.Assets {
					if a, ok := a.(*adapters.LocalAssetFile); ok {
						results = append(results, fileResult{name: path.Base(a.FileName), size: a.FileSize, title: a.Title})
					}
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

	logFile := configuration.DefaultLogFile()
	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()

			log := application.Log{
				File:  logFile,
				Level: "INFO",
			}
			err := log.OpenLogFile()
			if err != nil {
				t.Error(err)
				return
			}
			log.Logger.Info("\n\n\ntest case: " + c.name)
			recorder := fileevent.NewRecorder(log.Logger)

			fsys := c.gen()
			flags := &ImportFlags{
				SupportedMedia: metadata.DefaultSupportedMedia,
				CreateAlbums:   true,
				InfoCollector:  filenames.NewInfoCollector(time.Local, metadata.DefaultSupportedMedia),
			}
			log.Logger.Info("\n\n\ntest case: " + c.name)
			b, err := NewTakeout(ctx, recorder, flags, fsys...)
			if err != nil {
				t.Error(err)
				return
			}
			gChan := b.Browse(ctx)

			albums := album{}
			for g := range gChan {
				for _, a := range g.Assets {
					if a, ok := a.(*adapters.LocalAssetFile); ok {
						if len(g.Albums) > 0 {
							for _, al := range g.Albums {
								l := albums[al.Title]
								l = append(l, fileResult{name: path.Base(a.FileName), size: a.FileSize, title: a.Title})
								albums[al.Title] = l
							}
						}
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

/* Live photos are not supported anymore
func TestArchives(t *testing.T) {
	type photo map[fileKeyTracker]fileKeyTracker
	type album map[string][]fileKeyTracker
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
				fileKeyTracker{baseName: "PXL_20231118_035751175.MP.jpg", size: 8025699}: fileKeyTracker{baseName: "PXL_20231118_035751175.MP", size: 3478685},
				fileKeyTracker{baseName: "20231227_152817.jpg", size: 7426453}:           fileKeyTracker{baseName: "20231227_152817.MP4", size: 5192477},
			},
			wantAlbum: album{},
		},
		{
			name:      "checkLivePhotoPixil",
			gen:       checkLivePhotoPixil,
			wantAsset: photo{},
			wantLivePhotos: photo{
				fileKeyTracker{baseName: "IMG_4573.HEIC", size: 3530351}: fileKeyTracker{baseName: "IMG_4573.MP4", size: 2232086},
			},
			wantAlbum: album{
				"2022 - Germany - Private": []fileKeyTracker{{baseName: "IMG_4573.HEIC", size: 3530351}},
				"2022 - Germany":           []fileKeyTracker{{baseName: "IMG_4573.HEIC", size: 3530351}},
			},
		},
		{
			name: "checkMissingJSON-No",
			gen:  checkMissingJSON,
			wantAsset: photo{
				fileKeyTracker{baseName: "IMG_4573.HEIC", size: 3530351}: fileKeyTracker{},
			},
			wantLivePhotos: photo{},
			wantAlbum:      album{},
		},
		{
			name:              "checkMissingJSON-Yes",
			gen:               checkMissingJSON,
			acceptMissingJSON: true,
			wantAsset: photo{
				fileKeyTracker{baseName: "IMG_4573.HEIC", size: 3530351}:            fileKeyTracker{},
				fileKeyTracker{baseName: "IMG-20161201-WA0035.jpeg", size: 1352455}: fileKeyTracker{},
				fileKeyTracker{baseName: "IMG_4553.HEIC", size: 3530351}:            fileKeyTracker{},
			},
			wantLivePhotos: photo{
				fileKeyTracker{baseName: "IMG_1234.HEIC", size: 3530351}: fileKeyTracker{baseName: "IMG_1234.MP4", size: 2232086},
			},
			wantAlbum: album{
				"2022 - Germany": []fileKeyTracker{{baseName: "IMG_1234.HEIC", size: 3530351}},
			},
		},
		{
			name: "checkDuplicates",
			gen:  checkDuplicates,
			wantAsset: photo{
				fileKeyTracker{baseName: "20160105_121621_LLS.jpg", size: 365022}: fileKeyTracker{},
				fileKeyTracker{baseName: "20160105_121621_LLS.jpg", size: 364041}: fileKeyTracker{},
			},
			wantLivePhotos: photo{},
			wantAlbum:      album{},
		},
		{ // #405
			name: "checkMP_405",
			gen:  checkMPissue405,
			wantLivePhotos: photo{
				fileKeyTracker{baseName: "PXL_20210102_221126856.MP.jpg", size: 6486725}:   fileKeyTracker{baseName: "PXL_20210102_221126856.MP", size: 3242290},
				fileKeyTracker{baseName: "PXL_20210102_221126856.MP~2.jpg", size: 4028710}: fileKeyTracker{baseName: "PXL_20210102_221126856.MP~2", size: 1214365},
			},
			wantAlbum: album{},
			wantAsset: photo{},
		},
	}
	for _, c := range tc {
		t.Run(
			c.name,
			func(t *testing.T) {
				ctx := context.Background()
				fsys := c.gen()
				flags := &ImportFlags{
					SupportedMedia: metadata.DefaultSupportedMedia,
					KeepJSONLess:   c.acceptMissingJSON,
					CreateAlbums:   true,
				}
				b, err := NewTakeout(ctx, fileevent.NewRecorder(nil), flags, fsys...)
				if err != nil {
					t.Error(err)
					return
				}
				gChan := b.Browse(ctx)

				livePhotos := photo{}
				assets := photo{}
				albums := album{}
				for g := range gChan {
					var (
						photo *adapters.LocalAssetFile
						video *adapters.LocalAssetFile
					)
					for _, a := range g.Assets {
						ext := path.Ext(a.FileName)
						switch b.flags.SupportedMedia.TypeFromExt(ext) {
						case metadata.TypeImage:
							photo = a
						case metadata.TypeVideo:
							video = a
						}
						a.Close()
					}
					switch g.Kind {
					case adapters.GroupKindNone:
						assets[fileKeyTracker{baseName: photo.Title, size: photo.Size()}] = fileKeyTracker{}
					case adapters.GroupKindMotionPhoto:
						livePhotos[fileKeyTracker{baseName: photo.Title, size: photo.Size()}] = fileKeyTracker{baseName: video.Title, size: video.Size()}
					}
					for _, al := range g.Albums {
						l := albums[al.Title]
						l = append(l, fileKeyTracker{baseName: photo.Title, size: photo.Size()})
						albums[al.Title] = l
					}
				}

				if !equalPhotos(assets, c.wantAsset) {
					t.Errorf("difference assets\n")
					pretty.Ldiff(t, c.wantAsset, assets)
				}
				if !equalPhotos(livePhotos, c.wantLivePhotos) {
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


func equalPhotos(a, b map[fileKeyTracker]fileKeyTracker) bool {
	if len(a) != len(b) {
		return false
	}
	ka := gen.MapKeys(a)
	kb := gen.MapKeys(b)

	slices.SortFunc(ka, trackerKeySortFunc)
	slices.SortFunc(kb, trackerKeySortFunc)

	if !reflect.DeepEqual(ka, kb) {
		return false
	}

	for key, value := range a {
		if val, ok := b[key]; !ok || val != value {
			return false
		}
	}
	return true
}

*/
