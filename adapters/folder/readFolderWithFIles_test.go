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
	"github.com/simulot/immich-go/app"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/configuration"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/simulot/immich-go/internal/fshelper"
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
				SupportedMedia: filetypes.DefaultSupportedMedia,
				ManageBurst:    filters.BurstNothing,
				ManageRawJPG:   filters.RawJPGNothing,
				ManageHEICJPG:  filters.HeicJpgNothing,
				TZ:             time.Local,
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
			name: "date on the path given as argument, use names",
			flags: ImportFolderOptions{
				ManageBurst:          filters.BurstNothing,
				ManageRawJPG:         filters.RawJPGNothing,
				ManageHEICJPG:        filters.HeicJpgNothing,
				SupportedMedia:       filetypes.DefaultSupportedMedia,
				TakeDateFromFilename: true,
				TZ:                   time.Local,
				InclusionFlags: cliflags.InclusionFlags{
					DateRange: cliflags.InitDateRange(time.Local, "2023-10-06"),
				},
			},
			fsys: []fs.FS{
				fshelper.NewFSWithName(os.DirFS("DATA/2023/2023-10/2023-10-06"), "2023-10-06"),
			},
			expectedFiles: []string{
				"photo1_w_exif.jpg",
				"photo1_wo_exif.jpg",
			},
			expectedCounts: fileevent.NewCounts().
				Set(fileevent.DiscoveredImage, 2).Set(fileevent.DiscoveredDiscarded, 0).Set(fileevent.Uploaded, 2).
				Set(fileevent.INFO, 1).Value(),
		},
		{
			name: "date on the path given as argument, use names, in a TZ",
			flags: ImportFolderOptions{
				ManageBurst:          filters.BurstNothing,
				ManageRawJPG:         filters.RawJPGNothing,
				ManageHEICJPG:        filters.HeicJpgNothing,
				SupportedMedia:       filetypes.DefaultSupportedMedia,
				TakeDateFromFilename: true,
				TZ:                   time.FixedZone("UTC-4", -4*60*60),
				InclusionFlags: cliflags.InclusionFlags{
					DateRange: cliflags.InitDateRange(time.FixedZone("UTC-4", -4*60*60), "2023-10-06"),
				},
			},
			fsys: []fs.FS{
				fshelper.NewFSWithName(os.DirFS("DATA/2023/2023-10/2023-10-06"), "2023-10-06"),
			},
			expectedFiles: []string{
				"photo1_w_exif.jpg",
				"photo1_wo_exif.jpg",
			},
			expectedCounts: fileevent.NewCounts().
				Set(fileevent.DiscoveredImage, 2).Set(fileevent.DiscoveredDiscarded, 0).Set(fileevent.Uploaded, 2).
				Set(fileevent.INFO, 1).Value(),
		},
		{
			name: "date on the path given as argument, don't use names",
			flags: ImportFolderOptions{
				ManageBurst:          filters.BurstNothing,
				ManageRawJPG:         filters.RawJPGNothing,
				ManageHEICJPG:        filters.HeicJpgNothing,
				SupportedMedia:       filetypes.DefaultSupportedMedia,
				TakeDateFromFilename: false,
				TZ:                   time.Local,
				InclusionFlags: cliflags.InclusionFlags{
					DateRange: cliflags.InitDateRange(time.Local, "2023-10-06"),
				},
			},
			fsys: []fs.FS{
				fshelper.NewFSWithName(os.DirFS("DATA/2023/2023-10/2023-10-06"), "2023-10-06"),
			},
			expectedFiles: []string{
				"photo1_w_exif.jpg",
			},
			expectedCounts: fileevent.NewCounts().
				Set(fileevent.DiscoveredImage, 2).Set(fileevent.DiscoveredDiscarded, 1).Set(fileevent.Uploaded, 1).
				Set(fileevent.INFO, 1).Value(),
		},
		{
			name: "icloud-takeout",
			flags: ImportFolderOptions{
				SupportedMedia:         filetypes.DefaultSupportedMedia,
				Recursive:              true,
				ICloudTakeout:          true,
				ICloudMemoriesAsAlbums: true,
				TZ:                     time.Local,
				InclusionFlags: cliflags.InclusionFlags{
					DateRange: cliflags.InitDateRange(time.Local, "2023-10-06"),
				},
			},
			fsys: []fs.FS{
				os.DirFS("DATA/icloud-takeout"),
			},
			expectedFiles: []string{
				"Photos/photo1.jpg",
				"Photos/photo2.jpg",
				"Photos/photo_wo_exif.jpg",
			},
			expectedAlbums: map[string][]string{
				"Spécial album":        {"Photos/photo1.jpg", "Photos/photo2.jpg"},
				"Spécial album 2":      {"Photos/photo2.jpg"},
				"Memory 1. April 2025": {"Photos/photo1.jpg", "Photos/photo2.jpg"},
			},
			expectedCounts: fileevent.NewCounts().
				Set(fileevent.DiscoveredImage, 3).
				Set(fileevent.Uploaded, 3).
				Value(),
		},
	}

	logFile := configuration.DefaultLogFile()
	for _, c := range tc {
		t.Run(c.name, func(t *testing.T) {
			ctx := context.Background()

			log := app.Log{
				File:  logFile,
				Level: "DEBUG",
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
					results = append(results, a.File.Name())
					if len(c.expectedAlbums) > 0 {
						for _, album := range a.Albums {
							albums[album.Title] = append(albums[album.Title], a.File.Name())
						}
					}
					recorder.Record(ctx, fileevent.Uploaded, a.File)
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
