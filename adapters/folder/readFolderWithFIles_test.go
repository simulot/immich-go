package folder

import (
	"context"
	"io/fs"
	"os"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/kr/pretty"
	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/commands/application"
	"github.com/simulot/immich-go/helpers/configuration"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/metadata"
	"github.com/simulot/immich-go/internal/tzone"
)

func TestLocalAssets(t *testing.T) {
	tc := []struct {
		name           string
		fsys           []fs.FS
		flags          ImportFolderOptions
		expectedFiles  []string
		expectedCounts []int64
		expectedAlbums map[string][]string
	}{
		{
			name: "easy",
			flags: ImportFolderOptions{
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
			expectedFiles: []string{
				"photo1_w_exif.jpg",
				"photo1_wo_exif.jpg",
				"photo1_2024-10-06_w_exif.jpg",
				"photo1_2023-10-06_wo_exif.jpg",
			},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 4).Set(fileevent.Uploaded, 4).Value(),
		},
		{
			name: "date on name",
			flags: ImportFolderOptions{
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
			expectedFiles: []string{
				"photo1_2023-10-06_wo_exif.jpg",
			},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 4).Set(fileevent.DiscoveredDiscarded, 3).Set(fileevent.Uploaded, 1).Value(),
		},
		{
			name: "select exif date not using exiftool",
			flags: ImportFolderOptions{
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
			expectedFiles: []string{
				"photo1_w_exif.jpg",
				"photo1_2024-10-06_w_exif.jpg",
			},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 4).Set(fileevent.DiscoveredDiscarded, 2).Set(fileevent.Uploaded, 2).Value(),
		},
		{
			name: "select exif date using exiftool",
			flags: ImportFolderOptions{
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
			expectedFiles: []string{
				"photo1_w_exif.jpg",
				"photo1_2024-10-06_w_exif.jpg",
			},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 4).Set(fileevent.DiscoveredDiscarded, 2).Set(fileevent.Uploaded, 2).Value(),
		},
		{
			name: "select exif date using exiftool then date",
			flags: ImportFolderOptions{
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
			expectedFiles: []string{
				"photo1_w_exif.jpg",
				"photo1_2023-10-06_wo_exif.jpg",
				"photo1_2024-10-06_w_exif.jpg",
			},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 4).Set(fileevent.DiscoveredDiscarded, 1).Set(fileevent.Uploaded, 3).Value(),
		},
		{
			name: "select on date in the name",
			flags: ImportFolderOptions{
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
			expectedFiles: []string{
				"photo1_2023-10-06_wo_exif.jpg",
			},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 4).Set(fileevent.DiscoveredDiscarded, 3).Set(fileevent.Uploaded, 1).Value(),
		},
		{
			name: "same name, but not live photo, select exif date using exiftool then date",
			flags: ImportFolderOptions{
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
				os.DirFS("DATA/not-motion"),
			},
			expectedFiles: []string{
				"IMG_1234.jpg",
				"IMG_1234.mp4",
			},
			expectedCounts: fileevent.NewCounts().Set(fileevent.DiscoveredImage, 1).Set(fileevent.DiscoveredVideo, 1).Set(fileevent.Uploaded, 2).Value(),
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
			log.Info("Test case: " + c.name)
			recorder := fileevent.NewRecorder(log.Logger)
			b, err := NewLocalFiles(ctx, recorder, &c.flags, c.fsys...)
			if err != nil {
				t.Error(err)
			}

			groupChan := b.Browse(ctx)

			results := []string{}
			albums := map[string][]string{}
			for g := range groupChan {
				if err := g.Validate(); err != nil {
					t.Error(err)
					return
				}

				for _, a := range g.Assets {
					if a, ok := a.(*adapters.LocalAssetFile); ok {
						results = append(results, a.FileName)
						if len(c.expectedAlbums) > 0 {
							for _, album := range g.Albums {
								albums[album.Title] = append(albums[album.Title], a.FileName)
							}
						}
						recorder.Record(ctx, fileevent.Uploaded, fileevent.AsFileAndName(a.FSys, a.Name()))
					}
				}
			}
			sort.Strings(c.expectedFiles)
			sort.Strings(results)

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
