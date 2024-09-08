package folder

import (
	"context"
	"io/fs"
	"os"
	"path"
	"reflect"
	"testing"
	"time"

	"github.com/kr/pretty"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/metadata"
	"github.com/simulot/immich-go/internal/tzone"
)

func TestE2ELocalAssets(t *testing.T) {
	tc := []struct {
		name           string
		fsys           []fs.FS
		flags          ImportFlags
		expectedFiles  map[string]fileLinks
		expectedCounts []int64
		expectedAlbums map[string][]string
	}{
		{
			name: "easy",
			flags: ImportFlags{
				SupportedMedia: metadata.DefaultSupportedMedia,
				DateHandlingFlags: cliflags.DateHandlingFlags{
					Method: cliflags.DateMethodNone,
					FilenameTimeZone: tzone.Timezone{
						TZ: time.Local,
					},
				},
			},
			fsys: []fs.FS{
				os.DirFS("DATA/date-range"),
			},
			expectedFiles: map[string]fileLinks{
				"photo1_w_exif.jpg":             {image: "photo1_w_exif.jpg"},
				"photo1_wo_exif.jpg":            {image: "photo1_wo_exif.jpg"},
				"photo1_2024-10-06_w_exif.jpg":  {image: "photo1_2024-10-06_w_exif.jpg"},
				"photo1_2023-10-06_wo_exif.jpg": {image: "photo1_2023-10-06_wo_exif.jpg"},
			},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 4).Value(),
		},
		{
			name: "date on name",
			flags: ImportFlags{
				SupportedMedia: metadata.DefaultSupportedMedia,
				DateHandlingFlags: cliflags.DateHandlingFlags{
					Method: cliflags.DateMethodName,
					FilenameTimeZone: tzone.Timezone{
						TZ: time.Local,
					},
				},
				InclusionFlags: cliflags.InclusionFlags{
					DateRange: cliflags.InitDateRange("2023-10-06"),
				},
			},
			fsys: []fs.FS{
				os.DirFS("DATA/date-range"),
			},
			expectedFiles: map[string]fileLinks{
				"photo1_2023-10-06_wo_exif.jpg": {image: "photo1_2023-10-06_wo_exif.jpg"},
			},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 4).Set(fileevent.DiscoveredDiscarded, 3).Value(),
		},
		{
			name: "select exif date not using exiftool",
			flags: ImportFlags{
				SupportedMedia: metadata.DefaultSupportedMedia,
				DateHandlingFlags: cliflags.DateHandlingFlags{
					Method: cliflags.DateMethodEXIF,
					FilenameTimeZone: tzone.Timezone{
						TZ: time.Local,
					},
				},
				InclusionFlags: cliflags.InclusionFlags{
					DateRange: cliflags.InitDateRange("2023-10-06"),
				},
				ExifToolFlags: metadata.ExifToolFlags{
					UseExifTool: true,
					Timezone:    tzone.Timezone{TZ: time.Local},
				},
			},
			fsys: []fs.FS{
				os.DirFS("DATA/date-range"),
			},
			expectedFiles: map[string]fileLinks{
				"photo1_w_exif.jpg":            {image: "photo1_w_exif.jpg"},
				"photo1_2024-10-06_w_exif.jpg": {image: "photo1_2024-10-06_w_exif.jpg"},
			},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 4).Set(fileevent.DiscoveredDiscarded, 2).Value(),
		},
		{
			name: "select exif date using exiftool",
			flags: ImportFlags{
				SupportedMedia: metadata.DefaultSupportedMedia,
				DateHandlingFlags: cliflags.DateHandlingFlags{
					Method: cliflags.DateMethodEXIF,
					FilenameTimeZone: tzone.Timezone{
						TZ: time.Local,
					},
				},
				InclusionFlags: cliflags.InclusionFlags{
					DateRange: cliflags.InitDateRange("2023-10-06"),
				},
				ExifToolFlags: metadata.ExifToolFlags{
					UseExifTool: true,
					Timezone:    tzone.Timezone{TZ: time.Local},
				},
			},
			fsys: []fs.FS{
				os.DirFS("DATA/date-range"),
			},
			expectedFiles: map[string]fileLinks{
				"photo1_w_exif.jpg":            {image: "photo1_w_exif.jpg"},
				"photo1_2024-10-06_w_exif.jpg": {image: "photo1_2024-10-06_w_exif.jpg"},
			},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 4).Set(fileevent.DiscoveredDiscarded, 2).Value(),
		},
		{
			name: "select exif date using exiftool then date",
			flags: ImportFlags{
				SupportedMedia: metadata.DefaultSupportedMedia,
				DateHandlingFlags: cliflags.DateHandlingFlags{
					Method: cliflags.DateMethodExifThenName,
					FilenameTimeZone: tzone.Timezone{
						TZ: time.Local,
					},
				},
				InclusionFlags: cliflags.InclusionFlags{
					DateRange: cliflags.InitDateRange("2023-10-06"),
				},
				ExifToolFlags: metadata.ExifToolFlags{
					UseExifTool: true,
					Timezone:    tzone.Timezone{TZ: time.Local},
				},
			},
			fsys: []fs.FS{
				os.DirFS("DATA/date-range"),
			},
			expectedFiles: map[string]fileLinks{
				"photo1_w_exif.jpg":             {image: "photo1_w_exif.jpg"},
				"photo1_2023-10-06_wo_exif.jpg": {image: "photo1_2023-10-06_wo_exif.jpg"},
				"photo1_2024-10-06_w_exif.jpg":  {image: "photo1_2024-10-06_w_exif.jpg"},
			},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 4).Set(fileevent.DiscoveredDiscarded, 1).Value(),
		},
		{
			name: "select on date in the name",
			flags: ImportFlags{
				SupportedMedia: metadata.DefaultSupportedMedia,
				DateHandlingFlags: cliflags.DateHandlingFlags{
					Method: cliflags.DateMethodName,
					FilenameTimeZone: tzone.Timezone{
						TZ: time.Local,
					},
				},
				InclusionFlags: cliflags.InclusionFlags{
					DateRange: cliflags.InitDateRange("2023-10-06"),
				},
				ExifToolFlags: metadata.ExifToolFlags{
					UseExifTool: true,
					Timezone:    tzone.Timezone{TZ: time.Local},
				},
			},
			fsys: []fs.FS{
				os.DirFS("DATA/date-range"),
			},
			expectedFiles: map[string]fileLinks{
				"photo1_2023-10-06_wo_exif.jpg": {image: "photo1_2023-10-06_wo_exif.jpg"},
			},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 4).Set(fileevent.DiscoveredDiscarded, 3).Value(),
		},
	}

	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()
			recorder := fileevent.NewRecorder(nil, false)
			b, err := NewLocalFiles(ctx, recorder, &c.flags, c.fsys...)
			if err != nil {
				t.Error(err)
			}

			err = b.Prepare(ctx)
			if err != nil {
				t.Error(err)
			}

			results := map[string]fileLinks{}
			albums := map[string][]string{}
			for a := range b.Browse(ctx) {
				links := fileLinks{}
				ext := path.Ext(a.FileName)
				if b.flags.SupportedMedia.TypeFromExt(ext) == metadata.TypeImage {
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
				if len(c.expectedAlbums) > 0 {
					for _, album := range a.Albums {
						albums[album.Title] = append(albums[album.Title], a.FileName)
					}
				}
				a.Close()
			}

			if !reflect.DeepEqual(results, c.expectedFiles) {
				t.Errorf("file list difference\n")
				pretty.Ldiff(t, c.expectedFiles, results)
			}
			if !reflect.DeepEqual(recorder.GetCounts(), c.expectedCounts) {
				t.Errorf("counters difference\n")
				pretty.Ldiff(t, c.expectedCounts, recorder.GetCounts())
			}
			if c.expectedAlbums != nil {
				compareAlbums(t, albums, c.expectedAlbums)
			}
		})
	}
}
