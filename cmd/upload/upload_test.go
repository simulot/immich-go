package upload

import (
	"cmp"
	"context"
	"errors"
	"io"
	"io/fs"
	"reflect"
	"slices"
	"testing"

	"github.com/charmbracelet/log"
	"github.com/kr/pretty"
	"github.com/simulot/immich-go/browser"
	"github.com/simulot/immich-go/cmd"
	"github.com/simulot/immich-go/helpers/gen"
	"github.com/simulot/immich-go/immich"
)

type stubIC struct{}

func (c *stubIC) GetAllAssetsWithFilter(context.Context, func(*immich.Asset)) error {
	return nil
}

func (c *stubIC) AssetUpload(context.Context, *browser.LocalAssetFile) (immich.AssetResponse, error) {
	return immich.AssetResponse{}, nil
}

func (c *stubIC) DeleteAssets(context.Context, []string, bool) error {
	return nil
}

func (c *stubIC) GetAllAlbums(context.Context) ([]immich.AlbumSimplified, error) {
	return nil, nil
}

func (c *stubIC) AddAssetToAlbum(context.Context, string, []string) ([]immich.UpdateAlbumResult, error) {
	return nil, nil
}

func (c *stubIC) CreateAlbum(context.Context, string, []string) (immich.AlbumSimplified, error) {
	return immich.AlbumSimplified{}, nil
}

func (c *stubIC) UpdateAssets(ctx context.Context, ids []string, isArchived bool, isFavorite bool, latitude float64, longitude float64, removeParent bool, stackParentID string) error {
	return nil
}

func (c *stubIC) StackAssets(ctx context.Context, cover string, ids []string) error {
	return nil
}

func (c *stubIC) UpdateAsset(ctx context.Context, id string, a *browser.LocalAssetFile) (*immich.Asset, error) {
	return nil, nil
}

func (c *stubIC) EnableAppTrace(bool) {}

func (c *stubIC) GetServerStatistics(ctx context.Context) (immich.ServerStatistics, error) {
	return immich.ServerStatistics{}, nil
}

func (c *stubIC) PingServer(ctx context.Context) error {
	return nil
}

func (c *stubIC) SetDeviceUUID(string) {}

func (c *stubIC) SetEndPoint(string) {}

func (c *stubIC) ValidateConnection(ctx context.Context) (immich.User, error) {
	return immich.User{}, nil
}

func (c *stubIC) GetAssetAlbums(ctx context.Context, id string) ([]immich.AlbumSimplified, error) {
	return nil, nil
}

func (c *stubIC) GetAllAssets(ctx context.Context) ([]*immich.Asset, error) {
	return nil, nil
}

func (c *stubIC) DeleteAlbum(ctx context.Context, id string) error {
	return nil
}

func (c *stubIC) SupportedMedia() immich.SupportedMedia {
	return immich.DefaultSupportedMedia
}

type icCatchUploadsAssets struct {
	stubIC

	assets []string
	albums map[string][]string
}

func (c *icCatchUploadsAssets) AssetUpload(ctx context.Context, a *browser.LocalAssetFile) (immich.AssetResponse, error) {
	c.assets = append(c.assets, a.FileName)
	return immich.AssetResponse{
		ID: a.FileName,
	}, nil
}

func (c *icCatchUploadsAssets) AddAssetToAlbum(ctx context.Context, album string, ids []string) ([]immich.UpdateAlbumResult, error) {
	return nil, nil
}

func (c *icCatchUploadsAssets) CreateAlbum(ctx context.Context, album string, ids []string) (immich.AlbumSimplified, error) {
	if album == "" {
		panic("can't create album without name")
	}
	if c.albums == nil {
		c.albums = map[string][]string{}
	}
	l := c.albums[album]
	c.albums[album] = append(l, ids...)
	return immich.AlbumSimplified{
		ID:        album,
		AlbumName: album,
	}, nil
}

func TestUpload(t *testing.T) {
	testCases := []struct {
		name           string
		args           []string
		expectedErr    bool
		expectedAssets []string
		expectedAlbums map[string][]string
	}{
		{
			name: "Simple file",
			args: []string{
				"TEST_DATA/folder/low/PXL_20231006_063000139.jpg",
			},
			expectedErr:    false,
			expectedAssets: []string{"PXL_20231006_063000139.jpg"},
			expectedAlbums: map[string][]string{},
		},
		{
			name: "Simple file in an album",
			args: []string{
				"-album=the album",
				"TEST_DATA/folder/low/PXL_20231006_063000139.jpg",
			},
			expectedErr: false,
			expectedAssets: []string{
				"PXL_20231006_063000139.jpg",
			},
			expectedAlbums: map[string][]string{
				"the album": {"PXL_20231006_063000139.jpg"},
			},
		},
		{
			name: "Folders, no album creation",
			args: []string{
				"TEST_DATA/folder/high",
			},
			expectedErr: false,
			expectedAssets: []string{
				"AlbumA/PXL_20231006_063000139.jpg",
				"AlbumA/PXL_20231006_063029647.jpg",
				"AlbumA/PXL_20231006_063108407.jpg",
				"AlbumA/PXL_20231006_063121958.jpg",
				"AlbumA/PXL_20231006_063357420.jpg",
				"AlbumB/PXL_20231006_063528961.jpg",
				"AlbumB/PXL_20231006_063536303.jpg",
				"AlbumB/PXL_20231006_063851485.jpg",
			},
			expectedAlbums: map[string][]string{},
		},
		{
			name: "Folders, in given album",
			args: []string{
				"-album=the album",
				"TEST_DATA/folder/high",
			},
			expectedErr: false,
			expectedAssets: []string{
				"AlbumA/PXL_20231006_063000139.jpg",
				"AlbumA/PXL_20231006_063029647.jpg",
				"AlbumA/PXL_20231006_063108407.jpg",
				"AlbumA/PXL_20231006_063121958.jpg",
				"AlbumA/PXL_20231006_063357420.jpg",
				"AlbumB/PXL_20231006_063528961.jpg",
				"AlbumB/PXL_20231006_063536303.jpg",
				"AlbumB/PXL_20231006_063851485.jpg",
			},
			expectedAlbums: map[string][]string{
				"the album": {
					"AlbumA/PXL_20231006_063000139.jpg",
					"AlbumA/PXL_20231006_063029647.jpg",
					"AlbumA/PXL_20231006_063108407.jpg",
					"AlbumA/PXL_20231006_063121958.jpg",
					"AlbumA/PXL_20231006_063357420.jpg",
					"AlbumB/PXL_20231006_063528961.jpg",
					"AlbumB/PXL_20231006_063536303.jpg",
					"AlbumB/PXL_20231006_063851485.jpg",
				},
			},
		},
		{
			name: "Folders, album after folder",
			args: []string{
				"-create-album-folder",
				"TEST_DATA/folder/high",
			},
			expectedErr: false,
			expectedAssets: []string{
				"AlbumA/PXL_20231006_063000139.jpg",
				"AlbumA/PXL_20231006_063029647.jpg",
				"AlbumA/PXL_20231006_063108407.jpg",
				"AlbumA/PXL_20231006_063121958.jpg",
				"AlbumA/PXL_20231006_063357420.jpg",
				"AlbumB/PXL_20231006_063528961.jpg",
				"AlbumB/PXL_20231006_063536303.jpg",
				"AlbumB/PXL_20231006_063851485.jpg",
			},
			expectedAlbums: map[string][]string{
				"AlbumA": {
					"AlbumA/PXL_20231006_063000139.jpg",
					"AlbumA/PXL_20231006_063029647.jpg",
					"AlbumA/PXL_20231006_063108407.jpg",
					"AlbumA/PXL_20231006_063121958.jpg",
					"AlbumA/PXL_20231006_063357420.jpg",
				},
				"AlbumB": {
					"AlbumB/PXL_20231006_063528961.jpg",
					"AlbumB/PXL_20231006_063536303.jpg",
					"AlbumB/PXL_20231006_063851485.jpg",
				},
			},
		},
		{
			name: "google photos, default options",
			args: []string{
				"-google-photos",
				"TEST_DATA/Takeout1",
			},
			expectedErr: false,
			expectedAssets: []string{
				"Google Photos/Album test 6-10-23/PXL_20231006_063000139.jpg",
				"Google Photos/Album test 6-10-23/PXL_20231006_063029647.jpg",
				"Google Photos/Album test 6-10-23/PXL_20231006_063108407.jpg",
				"Google Photos/Album test 6-10-23/PXL_20231006_063121958.jpg",
				"Google Photos/Album test 6-10-23/PXL_20231006_063357420.jpg",
				"Google Photos/Album test 6-10-23/PXL_20231006_063536303.jpg",
				"Google Photos/Album test 6-10-23/PXL_20231006_063851485.jpg",
				"Google Photos/Album test 6-10-23/PXL_20231006_063909898.LS.mp4",
			},
			expectedAlbums: map[string][]string{
				"Album test 6/10/23": {
					"Google Photos/Album test 6-10-23/PXL_20231006_063000139.jpg",
					"Google Photos/Album test 6-10-23/PXL_20231006_063029647.jpg",
					"Google Photos/Album test 6-10-23/PXL_20231006_063108407.jpg",
					"Google Photos/Album test 6-10-23/PXL_20231006_063121958.jpg",
					"Google Photos/Album test 6-10-23/PXL_20231006_063357420.jpg",
					"Google Photos/Album test 6-10-23/PXL_20231006_063536303.jpg",
					"Google Photos/Album test 6-10-23/PXL_20231006_063851485.jpg",
					"Google Photos/Album test 6-10-23/PXL_20231006_063909898.LS.mp4",
				},
			},
		},
		{
			name: "google photos, album name from folder",
			args: []string{
				"-google-photos",
				"-use-album-folder-as-name",
				"TEST_DATA/Takeout1",
			},
			expectedErr: false,
			expectedAssets: []string{
				"Google Photos/Album test 6-10-23/PXL_20231006_063000139.jpg",
				"Google Photos/Album test 6-10-23/PXL_20231006_063029647.jpg",
				"Google Photos/Album test 6-10-23/PXL_20231006_063108407.jpg",
				"Google Photos/Album test 6-10-23/PXL_20231006_063121958.jpg",
				"Google Photos/Album test 6-10-23/PXL_20231006_063357420.jpg",
				"Google Photos/Album test 6-10-23/PXL_20231006_063536303.jpg",
				"Google Photos/Album test 6-10-23/PXL_20231006_063851485.jpg",
				"Google Photos/Album test 6-10-23/PXL_20231006_063909898.LS.mp4",
			},
			expectedAlbums: map[string][]string{
				"Album test 6-10-23": {
					"Google Photos/Album test 6-10-23/PXL_20231006_063000139.jpg",
					"Google Photos/Album test 6-10-23/PXL_20231006_063029647.jpg",
					"Google Photos/Album test 6-10-23/PXL_20231006_063108407.jpg",
					"Google Photos/Album test 6-10-23/PXL_20231006_063121958.jpg",
					"Google Photos/Album test 6-10-23/PXL_20231006_063357420.jpg",
					"Google Photos/Album test 6-10-23/PXL_20231006_063536303.jpg",
					"Google Photos/Album test 6-10-23/PXL_20231006_063851485.jpg",
					"Google Photos/Album test 6-10-23/PXL_20231006_063909898.LS.mp4",
				},
			},
		},
		{
			name: "google photo, ignore untitled, discard partner",
			args: []string{
				"-google-photos",
				"-keep-partner=FALSE",
				"TEST_DATA/Takeout2",
			},
			expectedErr: false,
			expectedAssets: []string{
				"Google Photos/Photos from 2023/PXL_20231006_063528961.jpg",
				"Google Photos/Sans titre(9)/PXL_20231006_063108407.jpg",
			},
			expectedAlbums: map[string][]string{},
		},

		{
			name: "google photo, ignore untitled, keep partner",
			args: []string{
				"-google-photos",
				"TEST_DATA/Takeout2",
			},

			expectedErr: false,
			expectedAssets: []string{
				"Google Photos/Photos from 2023/PXL_20231006_063528961.jpg",
				"Google Photos/Photos from 2023/PXL_20231006_063000139.jpg",
				"Google Photos/Sans titre(9)/PXL_20231006_063108407.jpg",
			},
			expectedAlbums: map[string][]string{},
		},
		{
			name: "google photo, ignore untitled, keep partner, partner album",
			args: []string{
				"-google-photos",
				"-partner-album=partner",
				"TEST_DATA/Takeout2",
			},
			expectedErr: false,
			expectedAssets: []string{
				"Google Photos/Photos from 2023/PXL_20231006_063528961.jpg",
				"Google Photos/Photos from 2023/PXL_20231006_063000139.jpg",
				"Google Photos/Sans titre(9)/PXL_20231006_063108407.jpg",
			},
			expectedAlbums: map[string][]string{
				"partner": {
					"Google Photos/Photos from 2023/PXL_20231006_063000139.jpg",
				},
			},
		},
		{
			name: "google photo, keep untitled",
			args: []string{
				"-google-photos",
				"-keep-untitled-albums",
				"-partner-album=partner",
				"TEST_DATA/Takeout2",
			},
			expectedErr: false,
			expectedAssets: []string{
				"Google Photos/Photos from 2023/PXL_20231006_063528961.jpg",
				"Google Photos/Photos from 2023/PXL_20231006_063000139.jpg",
				"Google Photos/Sans titre(9)/PXL_20231006_063108407.jpg",
			},
			expectedAlbums: map[string][]string{
				"partner": {
					"Google Photos/Photos from 2023/PXL_20231006_063000139.jpg",
				},
				"Sans titre(9)": {
					"Google Photos/Sans titre(9)/PXL_20231006_063108407.jpg",
				},
			},
		},
		{
			name: "google photo, includes .mp4",
			args: []string{
				"-google-photos",
				"-create-albums=FALSE",
				"-select-types=.mp4",
				"TEST_DATA/Takeout1",
			},
			expectedErr: false,
			expectedAssets: []string{
				"Google Photos/Album test 6-10-23/PXL_20231006_063909898.LS.mp4",
			},
		},
		{
			name: "google photo, exclude .mp4",
			args: []string{
				"-google-photos",
				"-create-albums=false",
				"-exclude-types=.mp4",
				"TEST_DATA/Takeout1",
			},
			expectedErr: false,
			expectedAssets: []string{
				"Google Photos/Album test 6-10-23/PXL_20231006_063000139.jpg",
				"Google Photos/Album test 6-10-23/PXL_20231006_063029647.jpg",
				"Google Photos/Album test 6-10-23/PXL_20231006_063108407.jpg",
				"Google Photos/Album test 6-10-23/PXL_20231006_063121958.jpg",
				"Google Photos/Album test 6-10-23/PXL_20231006_063357420.jpg",
				"Google Photos/Album test 6-10-23/PXL_20231006_063536303.jpg",
				"Google Photos/Album test 6-10-23/PXL_20231006_063851485.jpg",
			},
		},
		{
			name: "folder, includes .mp4",
			args: []string{
				"-select-types=.mp4",
				"TEST_DATA/Takeout1/Google Photos/Album test 6-10-23",
			},
			expectedErr: false,
			expectedAssets: []string{
				"PXL_20231006_063909898.LS.mp4",
			},
		},
		{
			name: "folder, exclude .mp4",
			args: []string{
				"-exclude-types=.mp4",
				"TEST_DATA/Takeout1/Google Photos/Album test 6-10-23",
			},
			expectedErr: false,
			expectedAssets: []string{
				"PXL_20231006_063000139.jpg",
				"PXL_20231006_063029647.jpg",
				"PXL_20231006_063108407.jpg",
				"PXL_20231006_063121958.jpg",
				"PXL_20231006_063357420.jpg",
				"PXL_20231006_063536303.jpg",
				"PXL_20231006_063851485.jpg",
			},
		},
		{
			name: "folder and albums creation",
			args: []string{
				"-create-album-folder",
				"TEST_DATA/Takeout2",
			},
			expectedAssets: []string{
				"Google Photos/Photos from 2023/PXL_20231006_063528961.jpg",
				"Google Photos/Photos from 2023/PXL_20231006_063000139.jpg",
				"Google Photos/Sans titre(9)/PXL_20231006_063108407.jpg",
			},
			expectedAlbums: map[string][]string{
				"Photos from 2023": {
					"Google Photos/Photos from 2023/PXL_20231006_063000139.jpg",
					"Google Photos/Photos from 2023/PXL_20231006_063528961.jpg",
				},
				"Sans titre(9)": {
					"Google Photos/Sans titre(9)/PXL_20231006_063108407.jpg",
				},
			},
		},
		// {
		// 	name: "google photo, homonyms, keep partner",
		// 	args: []string{
		// 		"-google-photos",
		// 		"TEST_DATA/Takeout3",
		// 	},
		// 	expectedErr: false,
		// 	expectedAssets: []string{
		// 		"Google Photos/Photos from 2023/DSC_0238_1.JPG",
		// 		"Google Photos/Photos from 2023/DSC_0238.JPG",
		// 		"Google Photos/Photos from 2023/DSC_0238(1).JPG",
		// 	},
		// },
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ic := &icCatchUploadsAssets{
				albums: map[string][]string{},
			}
			log := log.New(io.Discard)
			ctx := context.Background()

			serv := cmd.SharedFlags{
				Immich: ic,
				Log:    log,
			}

			app, err := NewUpCmd(ctx, &serv, tc.args)
			if err != nil {
				t.Errorf("can't instantiate the UploadCmd: %s", err)
				return
			}

			for _, fsys := range app.fsyss {
				err = errors.Join(app.run(ctx, []fs.FS{fsys}))
			}
			if (tc.expectedErr && err == nil) || (!tc.expectedErr && err != nil) {
				t.Errorf("unexpected error condition: %v,%s", tc.expectedErr, err)
				return
			}

			if !cmpSlices(tc.expectedAssets, ic.assets) {
				t.Errorf("expected upload differs ")
				pretty.Ldiff(t, tc.expectedAssets, ic.assets)
			}
			if !cmpAlbums(tc.expectedAlbums, ic.albums) {
				t.Errorf("expected albums differs ")
				pretty.Ldiff(t, tc.expectedAlbums, ic.albums)
			}
		})
	}
}

func cmpAlbums(a, b map[string][]string) bool {
	ka := gen.MapKeys(a)
	kb := gen.MapKeys(b)
	if !cmpSlices(ka, kb) {
		return false
	}
	r := true
	for _, k := range ka {
		r = r && cmpSlices(a[k], b[k])
		if !r {
			return r
		}
	}
	return r
}

func cmpSlices[T cmp.Ordered](a, b []T) bool {
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	slices.Sort(a)
	slices.Sort(b)
	return reflect.DeepEqual(a, b)
}
