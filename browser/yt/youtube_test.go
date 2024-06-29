package yt_test

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"reflect"
	"slices"
	"sort"
	"testing"
	"time"

	"github.com/simulot/immich-go/browser"
	"github.com/simulot/immich-go/browser/yt"
	"github.com/simulot/immich-go/helpers/fileevent"
	"github.com/simulot/immich-go/helpers/tzone"
	"github.com/simulot/immich-go/immich"
)

type SynthesizedYouTubeVideosByPlaylistID []*yt.YouTubePlaylist
func (a SynthesizedYouTubeVideosByPlaylistID) Len() int           { return len(a) }
func (a SynthesizedYouTubeVideosByPlaylistID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a SynthesizedYouTubeVideosByPlaylistID) Less(i, j int) bool { return a[i].PlaylistID < a[j].PlaylistID }

type LocalAlbumsByName []browser.LocalAlbum
func (a LocalAlbumsByName ) Len() int           { return len(a) }
func (a LocalAlbumsByName ) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a LocalAlbumsByName ) Less(i, j int) bool { return a[i].Name < a[j].Name }

func TestPrepareAndBrowse(t *testing.T) {
	channel := yt.YouTubeChannel {
		ChannelID:  "kb3ZF7Rwt2jc2MvVG1kyaze9",
		Title:      "Jonathan Stafford",
		Visibility: "Public",
	}

	// YouTubePlaylist.Filename() => []YouTubePlaylist
	playlists := map[string][]*yt.YouTubePlaylist{
		"A playlist-videos.csv": {
			&yt.YouTubePlaylist{
				PlaylistID:      "/63ek5JSj2ZcQSXaBiTslzKRSb+kK015UI",
				Description:     "This is my playlist",
				Title:           "A playlist",
				TitleLanguage:   "",
				CreateTimestamp: "2023-12-15T01:47:18+00:00",
				UpdateTimestamp: "2023-12-15T01:48:39+00:00",
				VideoOrder:      "Manual",
				Visibility:      "Private",
			},
		},
		"My playlist with a duplicate name-videos.csv": {
			&yt.YouTubePlaylist{
				PlaylistID:      "89uIs9+aFZ0rj76rdGXO2xeTQh/kz0aMCB",
				Description:     "",
				Title:           "My playlist with a duplicate name",
				TitleLanguage:   "",
				CreateTimestamp: "2023-12-17T14:03:27+00:00",
				UpdateTimestamp: "2023-12-17T14:16:09+00:00",
				VideoOrder:      "Manual",
				Visibility:      "Private",
			},
			&yt.YouTubePlaylist{
				PlaylistID:      "qfyrziKWJZA/u+O7qJ6b4xxx3zjH4zjpED",
				Description:     "",
				Title:           "My playlist with a duplicate name",
				TitleLanguage:   "",
				CreateTimestamp: "2023-12-17T14:03:21+00:00",
				UpdateTimestamp: "2023-12-17T14:27:03+00:00",
				VideoOrder:      "Manual",
				Visibility:      "Private",
			},
			&yt.YouTubePlaylist{
				PlaylistID:      "ozVxmuJGoBR+aT0FBghvOk+/j3a3JmwZPL",
				Description:     "",
				Title:           "My playlist with a duplicate name",
				TitleLanguage:   "",
				CreateTimestamp: "2023-12-17T14:03:17+00:00",
				UpdateTimestamp: "2023-12-17T14:27:18+00:00",
				VideoOrder:      "Manual",
				Visibility:      "Private",
			},
		},
		"My very long playlist title 0123456789 ABCD-vid.csv": {
			&yt.YouTubePlaylist{
				PlaylistID:      "Vrhwj5jEY4aJ9mssxYIlFS0YEd+YzbSCq3",
				Description:     "",
				Title:           "My very long playlist title 0123456789 ABCD",
				TitleLanguage:   "",
				CreateTimestamp: "2023-12-18T23:26:38+00:00",
				UpdateTimestamp: "2023-12-18T23:26:44+00:00",
				VideoOrder:      "Manual",
				Visibility:      "Private",
			},
		},
		"My very long playlist title 0123456789 ABCDE-vi.csv": {
			&yt.YouTubePlaylist{
				PlaylistID:      "/dvNHrUBJP17nJrNqt/HaQXHohMf0pR8ZA",
				Description:     "",
				Title:           "My very long playlist title 0123456789 ABCDE",
				TitleLanguage:   "",
				CreateTimestamp: "2023-12-18T23:25:46+00:00",
				UpdateTimestamp: "2023-12-18T23:26:27+00:00",
				VideoOrder:      "Manual",
				Visibility:      "Private",
			},
		},
		"My very long playlist title 0123456789 ABCDEF-v.csv": {
			&yt.YouTubePlaylist{
				PlaylistID:      "yrM0LUMa/lEHDnJ7UR2cIetuzOcNCWxBnD",
				Description:     "",
				Title:           "My very long playlist title 0123456789 ABCDEF",
				TitleLanguage:   "",
				CreateTimestamp: "2023-12-18T23:25:40+00:00",
				UpdateTimestamp: "2023-12-18T23:26:19+00:00",
				VideoOrder:      "Manual",
				Visibility:      "Private",
			},
		},
		"My very long playlist title 0123456789 ABCDEFG-.csv": {
			&yt.YouTubePlaylist{
				PlaylistID:      "19XtqwT0XgdNWjT3W9JbQfMEYI8uFiW9yI",
				Description:     "",
				Title:           "My very long playlist title 0123456789 ABCDEFG",
				TitleLanguage:   "",
				CreateTimestamp: "2023-12-18T23:25:31+00:00",
				UpdateTimestamp: "2023-12-18T23:26:09+00:00",
				VideoOrder:      "Manual",
				Visibility:      "Private",
			},
		},
		"My very long playlist title 0123456789 ABCDEFGH.csv": {
			&yt.YouTubePlaylist{
				PlaylistID:      "8eN2mETnoqHuqDoxXq3nLApeBxdqg7r9qU",
				Description:     "",
				Title:           "My very long playlist title 0123456789 ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmnopqrstuvwxyz 0123456789 ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmnopqrs",
				TitleLanguage:   "",
				CreateTimestamp: "2023-12-17T14:37:14+00:00",
				UpdateTimestamp: "2023-12-17T14:37:31+00:00",
				VideoOrder:      "Manual",
				Visibility:      "Private",
			},
			&yt.YouTubePlaylist{
				PlaylistID:      "yojzPdFgHjBNXsUcmTsmo6g1hTqsIkZUSB",
				Description:     "",
				Title:           "My very long playlist title 0123456789 ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmnopqrstuvwxyz 0123456789 ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmnopqrs",
				TitleLanguage:   "",
				CreateTimestamp: "2023-12-17T14:02:54+00:00",
				UpdateTimestamp: "2023-12-17T14:27:38+00:00",
				VideoOrder:      "Manual",
				Visibility:      "Private",
			},
		},
		"Z̤͔ͧ̑̓ä͖̭̈̇lͮ̒ͫǧ̗͚̚o̙̔ͮ̇͐̇ اختبار النص-videos.csv": {
			&yt.YouTubePlaylist{
				PlaylistID:      "NnnqWkLMzsQ40on1sPO3D5egybOaP2cra/",
				Description:     "",
				Title:           "Z̤͔ͧ̑̓ä͖̭̈̇lͮ̒ͫǧ̗͚̚o̙̔ͮ̇͐̇ اختبار النص",
				TitleLanguage:   "",
				CreateTimestamp: "2023-12-17T14:33:16+00:00",
				UpdateTimestamp: "2023-12-17T14:33:22+00:00",
				VideoOrder:      "Manual",
				Visibility:      "Private",
			},
		},
		"`-=[]_,._~!@#$_^&_()_+{}_-videos.csv": {
			&yt.YouTubePlaylist{
				PlaylistID:      "ykzP/AtWUAgtD6kfpZyk+CCipFNPlh27FA",
				Description:     "",
				Title:           "`-=[]\\;',./~!@#$%^&*()_+{}|:\"?",
				TitleLanguage:   "",
				CreateTimestamp: "2023-12-17T13:57:31+00:00",
				UpdateTimestamp: "2023-12-17T14:32:42+00:00",
				VideoOrder:      "Manual",
				Visibility:      "Private",
			},
		},
		"👱👱🏻👱🏼👱🏽👱🏾👱🏿 🧟‍♀️🧟‍♂️ 👨‍❤️‍💋‍👨👩.csv": {
			&yt.YouTubePlaylist{
				PlaylistID:      "CagoNmT9b8xzQ/JDLR1cTyjQ+R2dWIlkho",
				Description:     "",
				Title:           "👱👱🏻👱🏼👱🏽👱🏾👱🏿 🧟‍♀️🧟‍♂️ 👨‍❤️‍💋‍👨👩‍👩‍👧‍👦🏳️‍⚧️🇵🇷",
				TitleLanguage:   "",
				CreateTimestamp: "2023-12-17T14:32:59+00:00",
				UpdateTimestamp: "2023-12-17T14:33:08+00:00",
				VideoOrder:      "Manual",
				Visibility:      "Private",
			},
		},
		"😀Lorem😁ipsum😂dolor😃sit😄amet😅😆😇😈😉😊😋😌.csv": {
			&yt.YouTubePlaylist{
				PlaylistID:      "NdhO/BxkyoiY1OaA98FdsEKotGUIIkenBX",
				Description:     "",
				Title:           "😀Lorem😁ipsum😂dolor😃sit😄amet😅😆😇😈😉😊😋😌😍😎😏😐😑😒😓😕😖😗😘😙😚😛😜😝😞😟😠😡😢😣😤😥😦😧😨😩😪😫😬😭😮😯😰😱😲😳😴😵😶😷😸😹😺😻😼😽😾😿🙀🙁🙂🙃🙄🙅🙆🙇🙈🙉🙊🙋🙌🙍🙎🙏",
				TitleLanguage:   "",
				CreateTimestamp: "2023-12-19T01:06:07+00:00",
				UpdateTimestamp: "2023-12-19T01:12:15+00:00",
				VideoOrder:      "Manual",
				Visibility:      "Private",
			},
		},
		"😀orem😁ipsum😂dolor😃sit😄amet😅😆😇😈😉😊😋😌.csv": {
			&yt.YouTubePlaylist{
				PlaylistID:      "0q2fQnYVgBZ97Pxa0dUTqdHBtk3B/xeyta",
				Description:     "",
				Title:           "😀orem😁ipsum😂dolor😃sit😄amet😅😆😇😈😉😊😋😌😍😎😏😐😑😒😓😕😖😗😘😙😚😛😜😝😞😟😠😡😢😣😤😥😦😧😨😩😪😫😬😭😮😯😰😱😲😳😴😵😶😷😸😹😺😻😼😽😾😿🙀🙁🙂🙃🙄🙅🙆🙇🙈🙉🙊🙋🙌🙍🙎🙏",
				TitleLanguage:   "",
				CreateTimestamp: "2023-12-19T22:19:00+00:00",
				UpdateTimestamp: "2023-12-19T22:19:09+00:00",
				VideoOrder:      "Manual",
				Visibility:      "Private",
			},
		},
		"😀😁😂😃😄😅😆😇😈😉😊😋😌😍😎😏😐😑😒😓😕😖😗😘.csv": {
			&yt.YouTubePlaylist{
				PlaylistID:      "+RSTyiTrFuyMrjlZpuQxw7d3LPzCMc8LaD",
				Description:     "",
				Title:           "😀😁😂😃😄😅😆😇😈😉😊😋😌😍😎😏😐😑😒😓😕😖😗😘😙😚😛😜😝😞😟😠😡😢😣😤😥😦😧😨😩😪😫😬😭😮😯😰😱😲😳😴😵😶😷😸😹😺😻😼😽😾😿🙀🙁🙂🙃🙄🙅🙆🙇🙈🙉🙊tema🙋tis🙌rolod🙍muspi🙎meroL🙏",
				TitleLanguage:   "",
				CreateTimestamp: "2023-12-19T01:07:46+00:00",
				UpdateTimestamp: "2023-12-19T01:12:45+00:00",
				VideoOrder:      "Manual",
				Visibility:      "Private",
			},
		},
	}

	fsys := os.DirFS("TEST_DATA/20240623T224719")
	takeout, err := fs.Sub(fsys, "Takeout")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	youtube, err := fs.Sub(takeout, "YouTube and YouTube Music")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	videos, err := fs.Sub(youtube, "videos")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	wantVideos := []*yt.SynthesizedYouTubeVideo{
		&yt.SynthesizedYouTubeVideo{
			Channel: &channel,
			Playlists: slices.Concat(playlists["A playlist-videos.csv"],
			                         playlists["My playlist with a duplicate name-videos.csv"],
			                         playlists["My very long playlist title 0123456789 ABCDEFGH.csv"]),
			Video: &yt.YouTubeVideo{
				VideoID:          "PJvZQ6mMSBf",
				Duration:         55000,
				Language:         "",
				Category:         "People",
				Description:      "A description of Serenade #2",
				ChannelID:        "kb3ZF7Rwt2jc2MvVG1kyaze9",
				Title:            "Serenade #2",
				Privacy:          "Private",
				State:            "Processed",
				CreateTimestamp:  "2016-03-11T11:19:17+00:00",
				PublishTimestamp: "",
			},
			Recording: &yt.YouTubeVideoRecording{
				VideoID:   "PJvZQ6mMSBf",
				Address:   "",
				Altitude:  0,
				Latitude:  0,
				Longitude: 0,
				PlaceID:   "",
			},
			Fsys: videos,
			Filename: "Serenade #2.mp4",
		},
		&yt.SynthesizedYouTubeVideo{
			Channel: &channel,
			Playlists: slices.Concat(playlists["😀😁😂😃😄😅😆😇😈😉😊😋😌😍😎😏😐😑😒😓😕😖😗😘.csv"],
			                         playlists["😀Lorem😁ipsum😂dolor😃sit😄amet😅😆😇😈😉😊😋😌.csv"],
			                         playlists["My very long playlist title 0123456789 ABCDEFG-.csv"],
			                         playlists["My very long playlist title 0123456789 ABCDEF-v.csv"],
			                         playlists["My very long playlist title 0123456789 ABCDE-vi.csv"],
			                         playlists["My very long playlist title 0123456789 ABCD-vid.csv"],
			                         playlists["😀orem😁ipsum😂dolor😃sit😄amet😅😆😇😈😉😊😋😌.csv"]),
			Video: &yt.YouTubeVideo{
				VideoID:          "rl1vcIiguJV",
				Duration:         78000,
				Language:         "",
				Category:         "People",
				Description:      "",
				ChannelID:        "kb3ZF7Rwt2jc2MvVG1kyaze9",
				Title:            "Serenade #1",
				Privacy:          "Private",
				State:            "Processed",
				CreateTimestamp:  "2016-03-11T11:20:49+00:00",
				PublishTimestamp: "",
			},
			Recording: &yt.YouTubeVideoRecording{
				VideoID:   "rl1vcIiguJV",
				Address:   "",
				Altitude:  0,
				Latitude:  0,
				Longitude: 0,
				PlaceID:   "",
			},
			Fsys: videos,
			Filename: "Serenade #1.mp4",
		},
		&yt.SynthesizedYouTubeVideo{
			Channel: &channel,
			Playlists: slices.Concat(playlists["`-=[]_,._~!@#$_^&_()_+{}_-videos.csv"],
			                         playlists["A playlist-videos.csv"],
			                         playlists["👱👱🏻👱🏼👱🏽👱🏾👱🏿 🧟‍♀️🧟‍♂️ 👨‍❤️‍💋‍👨👩.csv"],
			                         playlists["Z̤͔ͧ̑̓ä͖̭̈̇lͮ̒ͫǧ̗͚̚o̙̔ͮ̇͐̇ اختبار النص-videos.csv"]),
			Video: &yt.YouTubeVideo{
				VideoID:          "pI3tVoMUwz5",
				Duration:         16000,
				Language:         "",
				Category:         "People",
				Description:      "",
				ChannelID:        "kb3ZF7Rwt2jc2MvVG1kyaze9",
				Title:            "I manually set the location",
				Privacy:          "Private",
				State:            "Processed",
				CreateTimestamp:  "2023-12-14T00:34:22+00:00",
				PublishTimestamp: "2023-12-14T05:00:21+00:00",
			},
			Recording: &yt.YouTubeVideoRecording{
				VideoID:   "pI3tVoMUwz5",
				Address:   "The White House",
				Altitude:  0,
				Latitude:  38.8977,
				Longitude: -77.0365,
				PlaceID:   "ChIJ37HL3ry3t4kRv3YLbdhpWXE",
			},
			Fsys: videos,
			Filename: "I manually set the location.mp4",
		},
		&yt.SynthesizedYouTubeVideo{
			Channel: &channel,
			Playlists: slices.Concat(playlists["My playlist with a duplicate name-videos.csv"]),
			Video: &yt.YouTubeVideo{
				VideoID:          "d5IMr4n6DIh",
				Duration:         16000,
				Language:         "en-US",
				Category:         "People",
				Description:      "",
				ChannelID:        "kb3ZF7Rwt2jc2MvVG1kyaze9",
				Title:            "`-=[]\\;',./~!@#$%^\u0026*()_+{}|:\"? 👱🏻🧟‍♀️👨‍❤️‍💋‍👨👩‍👩‍👧‍👦🏳️‍⚧️🇵🇷 Z̤͔ͧ̑̓ä͖̭̈̇lͮ̒ͫǧ̗͚̚o̙̔ͮ̇͐̇ اختبار النص",
				Privacy:          "Private",
				State:            "Processed",
				CreateTimestamp:  "2023-12-14T00:36:14+00:00",
				PublishTimestamp: "",
			},
			Recording: &yt.YouTubeVideoRecording{
				VideoID:   "d5IMr4n6DIh",
				Address:   "",
				Altitude:  0,
				Latitude:  0,
				Longitude: 0,
				PlaceID:   "",
			},
			Fsys: videos,
			Filename: "`-=[]_,._~!@#$_^&_()_+{}_ 👱🏻🧟‍♀️👨‍❤️‍💋.mp4",
		},
		&yt.SynthesizedYouTubeVideo{
			Channel: &channel,
			Playlists: slices.Concat(playlists["A playlist-videos.csv"],
			                         playlists["My playlist with a duplicate name-videos.csv"]),
			Video: &yt.YouTubeVideo{
				VideoID:          "a+q6oaXj7dH",
				Duration:         16000,
				Language:         "en-US",
				Category:         "People",
				Description:      "A description of a Short video.",
				ChannelID:        "kb3ZF7Rwt2jc2MvVG1kyaze9",
				Title:            "`-=[]\\;',./~!@#$%^\u0026*()_+{}|:\"? 👱🏻🧟‍♀️👨‍❤️‍💋‍👨👩‍👩‍👧‍👦🏳️‍⚧️🇵🇷 Z̤͔ͧ̑̓ä͖̭̈̇lͮ̒ͫǧ̗͚̚o̙̔ͮ̇͐̇ اختبار النص",
				Privacy:          "Private",
				State:            "Processed",
				CreateTimestamp:  "2023-12-14T01:05:57+00:00",
				PublishTimestamp: "",
			},
			Recording: &yt.YouTubeVideoRecording{
				VideoID:   "a+q6oaXj7dH",
				Address:   "",
				Altitude:  0,
				Latitude:  0,
				Longitude: 0,
				PlaceID:   "",
			},
			Fsys: videos,
			Filename: "`-=[]_,._~!@#$_^&_()_+{}_ 👱🏻🧟‍♀️👨‍❤️‍💋(1).mp4",
		},
		&yt.SynthesizedYouTubeVideo{
			Channel: &channel,
			Playlists: slices.Concat(playlists["My playlist with a duplicate name-videos.csv"]),
			Video: &yt.YouTubeVideo{
				VideoID:          "NTOBfooePHb",
				Duration:         16000,
				Language:         "en-US",
				Category:         "People",
				Description:      "",
				ChannelID:        "kb3ZF7Rwt2jc2MvVG1kyaze9",
				Title:            "`-=[]\\;',./~!@#$%^\u0026*()_+{}|:\"? 👱🏻🧟‍♀️👨‍❤️‍💋‍👨👩‍👩‍👧‍👦🏳️‍⚧️🇵🇷 Z̤͔ͧ̑̓ä͖̭̈̇lͮ̒ͫǧ̗͚̚o̙̔ͮ̇͐̇ اختبار النص",
				Privacy:          "Private",
				State:            "Processed",
				CreateTimestamp:  "2023-12-17T14:14:46+00:00",
				PublishTimestamp: "",
			},
			Recording: &yt.YouTubeVideoRecording{
				VideoID:   "NTOBfooePHb",
				Address:   "",
				Altitude:  0,
				Latitude:  0,
				Longitude: 0,
				PlaceID:   "",
			},
			Fsys: videos,
			Filename: "`-=[]_,._~!@#$_^&_()_+{}_ 👱🏻🧟‍♀️👨‍❤️‍💋(2).mp4",
		},
	}

	ctx := context.Background()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	l := fileevent.NewRecorder(log, false)
	sm := immich.DefaultSupportedMedia

	to, err := yt.NewTakeout(ctx, l, sm, fsys)
	if err != nil{
		t.Fatalf("unexpected error: %s", err)
	}

	err = to.Prepare(ctx)
	if err != nil{
		t.Fatalf("unexpected error: %s", err)
	}

	// Test Prepare
	gotVideos := to.Videos()
	if len(gotVideos) != len(wantVideos) {
		t.Errorf("Prepare returned %d videos instead of %d", len(gotVideos), len(wantVideos))
	}

	for i, _ := range gotVideos {
		// The order of the playlists in the data we read depends on
		// the order of the playlists in playlists.csv, which seems to
		// be random.  Also we don't really care in the first place,
		// so just make it predictable for the test:
		sort.Sort(SynthesizedYouTubeVideosByPlaylistID(gotVideos[i].Playlists))
		sort.Sort(SynthesizedYouTubeVideosByPlaylistID(wantVideos[i].Playlists))

		if !reflect.DeepEqual(gotVideos[i], wantVideos[i]) {
			want_json, _ := json.MarshalIndent(wantVideos[i], "", "	")
			got_json, _  := json.MarshalIndent(gotVideos[i], "", "	")
			t.Fatalf("Prepare returned\n%s\ninstead of\n%s\nfor index %d", got_json, want_json, i)
		}
	}

	local, _ := tzone.Local()

	wantLafs := []*browser.LocalAssetFile{
		&browser.LocalAssetFile{
			FileName:    "Serenade #2.mp4",
			Title:       "Serenade #2",
			Description: "A description of Serenade #2",
			Albums:      []browser.LocalAlbum{
				browser.LocalAlbum{
					//Path: "A playlist-videos.csv",
					Path: "A playlist",
					Name: "A playlist",
				},
				browser.LocalAlbum{
					//Path: "My playlist with a duplicate name-videos.csv",
					Path: "My playlist with a duplicate name",
					Name: "My playlist with a duplicate name",
				},
				browser.LocalAlbum{
					//Path: "My playlist with a duplicate name-videos.csv",
					Path: "My playlist with a duplicate name",
					Name: "My playlist with a duplicate name",
				},
				browser.LocalAlbum{
					//Path: "My playlist with a duplicate name-videos.csv",
					Path: "My playlist with a duplicate name",
					Name: "My playlist with a duplicate name",
				},
				browser.LocalAlbum{
					//Path: "My very long playlist title 0123456789 ABCDEFGH.csv",
					Path: "My very long playlist title 0123456789 ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmnopqrstuvwxyz 0123456789 ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmnopqrs",
					Name: "My very long playlist title 0123456789 ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmnopqrstuvwxyz 0123456789 ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmnopqrs",
				},
				browser.LocalAlbum{
					//Path: "My very long playlist title 0123456789 ABCDEFGH.csv",
					Path: "My very long playlist title 0123456789 ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmnopqrstuvwxyz 0123456789 ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmnopqrs",
					Name: "My very long playlist title 0123456789 ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmnopqrstuvwxyz 0123456789 ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmnopqrs",
				},
			},

			DateTaken:   time.Date(int(2016), time.March, int(11), int(11), int(19), int(17), int(0), time.UTC).In(local),
			Latitude:    0,
			Longitude:   0,
			Altitude:    0,

			Trashed:     false,
			Archived:    false,
			FromPartner: false,
			Favorite:    false,

			FSys:        videos,
			FileSize:    7,
		},
		&browser.LocalAssetFile{
			FileName:    "Serenade #1.mp4",
			Title:       "Serenade #1",
			Description: "",
			Albums:      []browser.LocalAlbum{
				browser.LocalAlbum{
					//Path: "😀😁😂😃😄😅😆😇😈😉😊😋😌😍😎😏😐😑😒😓😕😖😗😘.csv"
					Path: "😀😁😂😃😄😅😆😇😈😉😊😋😌😍😎😏😐😑😒😓😕😖😗😘😙😚😛😜😝😞😟😠😡😢😣😤😥😦😧😨😩😪😫😬😭😮😯😰😱😲😳😴😵😶😷😸😹😺😻😼😽😾😿🙀🙁🙂🙃🙄🙅🙆🙇🙈🙉🙊tema🙋tis🙌rolod🙍muspi🙎meroL🙏",
					Name: "😀😁😂😃😄😅😆😇😈😉😊😋😌😍😎😏😐😑😒😓😕😖😗😘😙😚😛😜😝😞😟😠😡😢😣😤😥😦😧😨😩😪😫😬😭😮😯😰😱😲😳😴😵😶😷😸😹😺😻😼😽😾😿🙀🙁🙂🙃🙄🙅🙆🙇🙈🙉🙊tema🙋tis🙌rolod🙍muspi🙎meroL🙏",
				},
				browser.LocalAlbum{
					//Path: "😀Lorem😁ipsum😂dolor😃sit😄amet😅😆😇😈😉😊😋😌.csv"
					Path: "😀Lorem😁ipsum😂dolor😃sit😄amet😅😆😇😈😉😊😋😌😍😎😏😐😑😒😓😕😖😗😘😙😚😛😜😝😞😟😠😡😢😣😤😥😦😧😨😩😪😫😬😭😮😯😰😱😲😳😴😵😶😷😸😹😺😻😼😽😾😿🙀🙁🙂🙃🙄🙅🙆🙇🙈🙉🙊🙋🙌🙍🙎🙏",
					Name: "😀Lorem😁ipsum😂dolor😃sit😄amet😅😆😇😈😉😊😋😌😍😎😏😐😑😒😓😕😖😗😘😙😚😛😜😝😞😟😠😡😢😣😤😥😦😧😨😩😪😫😬😭😮😯😰😱😲😳😴😵😶😷😸😹😺😻😼😽😾😿🙀🙁🙂🙃🙄🙅🙆🙇🙈🙉🙊🙋🙌🙍🙎🙏",
				},
				browser.LocalAlbum{
					//Path: "My very long playlist title 0123456789 ABCDEFG-.csv"
					Path: "My very long playlist title 0123456789 ABCDEFG",
					Name: "My very long playlist title 0123456789 ABCDEFG",
				},
				browser.LocalAlbum{
					//Path: "My very long playlist title 0123456789 ABCDEF-v.csv"
					Path: "My very long playlist title 0123456789 ABCDEF",
					Name: "My very long playlist title 0123456789 ABCDEF",
				},
				browser.LocalAlbum{
					//Path: "My very long playlist title 0123456789 ABCDE-vi.csv"
					Path: "My very long playlist title 0123456789 ABCDE",
					Name: "My very long playlist title 0123456789 ABCDE",
				},
				browser.LocalAlbum{
					//Path: "My very long playlist title 0123456789 ABCD-vid.csv"
					Path: "My very long playlist title 0123456789 ABCD",
					Name: "My very long playlist title 0123456789 ABCD",
				},
				browser.LocalAlbum{
					//Path: "😀orem😁ipsum😂dolor😃sit😄amet😅😆😇😈😉😊😋😌.csv"
					Path: "😀orem😁ipsum😂dolor😃sit😄amet😅😆😇😈😉😊😋😌😍😎😏😐😑😒😓😕😖😗😘😙😚😛😜😝😞😟😠😡😢😣😤😥😦😧😨😩😪😫😬😭😮😯😰😱😲😳😴😵😶😷😸😹😺😻😼😽😾😿🙀🙁🙂🙃🙄🙅🙆🙇🙈🙉🙊🙋🙌🙍🙎🙏",
					Name: "😀orem😁ipsum😂dolor😃sit😄amet😅😆😇😈😉😊😋😌😍😎😏😐😑😒😓😕😖😗😘😙😚😛😜😝😞😟😠😡😢😣😤😥😦😧😨😩😪😫😬😭😮😯😰😱😲😳😴😵😶😷😸😹😺😻😼😽😾😿🙀🙁🙂🙃🙄🙅🙆🙇🙈🙉🙊🙋🙌🙍🙎🙏",
				},
			},

			DateTaken:   time.Date(int(2016), time.March, int(11), int(11), int(20), int(49), int(0), time.UTC).In(local),
			Latitude:    0,
			Longitude:   0,
			Altitude:    0,

			Trashed:     false,
			Archived:    false,
			FromPartner: false,
			Favorite:    false,

			FSys:        videos,
			FileSize:    6,
		},
		&browser.LocalAssetFile{
			FileName:    "I manually set the location.mp4",
			Title:       "I manually set the location",
			Description: "",
			Albums:      []browser.LocalAlbum{
				browser.LocalAlbum{
					//Path: "`-=[]_,._~!@#$_^&_()_+{}_-videos.csv"
					Path: "`-=[]\\;',./~!@#$%^&*()_+{}|:\"?",
					Name: "`-=[]\\;',./~!@#$%^&*()_+{}|:\"?",
				},
				browser.LocalAlbum{
					//Path: "A playlist-videos.csv"
					Path: "A playlist",
					Name: "A playlist",
				},
				browser.LocalAlbum{
					//Path: "👱👱🏻👱🏼👱🏽👱🏾👱🏿 🧟‍♀️🧟‍♂️ 👨‍❤️‍💋‍👨👩.csv"
					Path: "👱👱🏻👱🏼👱🏽👱🏾👱🏿 🧟‍♀️🧟‍♂️ 👨‍❤️‍💋‍👨👩‍👩‍👧‍👦🏳️‍⚧️🇵🇷",
					Name: "👱👱🏻👱🏼👱🏽👱🏾👱🏿 🧟‍♀️🧟‍♂️ 👨‍❤️‍💋‍👨👩‍👩‍👧‍👦🏳️‍⚧️🇵🇷",
				},
				browser.LocalAlbum{
					//Path: "Z̤͔ͧ̑̓ä͖̭̈̇lͮ̒ͫǧ̗͚̚o̙̔ͮ̇͐̇ اختبار النص-videos.csv"
					Path: "Z̤͔ͧ̑̓ä͖̭̈̇lͮ̒ͫǧ̗͚̚o̙̔ͮ̇͐̇ اختبار النص",
					Name: "Z̤͔ͧ̑̓ä͖̭̈̇lͮ̒ͫǧ̗͚̚o̙̔ͮ̇͐̇ اختبار النص",
				},
			},

			DateTaken:   time.Date(int(2023), time.December, int(14), int(0), int(34), int(22), int(0), time.UTC).In(local),
			Latitude:    38.8977,
			Longitude:   -77.0365,
			Altitude:    0,

			Trashed:     false,
			Archived:    false,
			FromPartner: false,
			Favorite:    false,

			FSys:        videos,
			FileSize:    5,
		},
		&browser.LocalAssetFile{
			FileName:    "`-=[]_,._~!@#$_^&_()_+{}_ 👱🏻🧟‍♀️👨‍❤️‍💋.mp4",
			Title:       "`-=[]\\;',./~!@#$%^\u0026*()_+{}|:\"? 👱🏻🧟‍♀️👨‍❤️‍💋‍👨👩‍👩‍👧‍👦🏳️‍⚧️🇵🇷 Z̤͔ͧ̑̓ä͖̭̈̇lͮ̒ͫǧ̗͚̚o̙̔ͮ̇͐̇ اختبار النص",
			Description: "",
			Albums:      []browser.LocalAlbum{
				browser.LocalAlbum{
					//Path: "My playlist with a duplicate name-videos.csv
					Path: "My playlist with a duplicate name",
					Name: "My playlist with a duplicate name",
				},
				browser.LocalAlbum{
					//Path: "My playlist with a duplicate name-videos.csv
					Path: "My playlist with a duplicate name",
					Name: "My playlist with a duplicate name",
				},
				browser.LocalAlbum{
					//Path: "My playlist with a duplicate name-videos.csv
					Path: "My playlist with a duplicate name",
					Name: "My playlist with a duplicate name",
				},
			},

			DateTaken:   time.Date(int(2023), time.December, int(14), int(0), int(36), int(14), int(0), time.UTC).In(local),
			Latitude:    0,
			Longitude:   0,
			Altitude:    0,

			Trashed:     false,
			Archived:    false,
			FromPartner: false,
			Favorite:    false,

			FSys:        videos,
			FileSize:    2,
		},
		&browser.LocalAssetFile{
			FileName:    "`-=[]_,._~!@#$_^&_()_+{}_ 👱🏻🧟‍♀️👨‍❤️‍💋(1).mp4",
			Title:       "`-=[]\\;',./~!@#$%^\u0026*()_+{}|:\"? 👱🏻🧟‍♀️👨‍❤️‍💋‍👨👩‍👩‍👧‍👦🏳️‍⚧️🇵🇷 Z̤͔ͧ̑̓ä͖̭̈̇lͮ̒ͫǧ̗͚̚o̙̔ͮ̇͐̇ اختبار النص",
			Description: "A description of a Short video.",
			Albums:      []browser.LocalAlbum{
				browser.LocalAlbum{
					//Path: "A playlist-videos.csv"
					Path: "A playlist",
					Name: "A playlist",
				},
				browser.LocalAlbum{
					//Path: "My playlist with a duplicate name-videos.csv
					Path: "My playlist with a duplicate name",
					Name: "My playlist with a duplicate name",
				},
				browser.LocalAlbum{
					//Path: "My playlist with a duplicate name-videos.csv
					Path: "My playlist with a duplicate name",
					Name: "My playlist with a duplicate name",
				},
				browser.LocalAlbum{
					//Path: "My playlist with a duplicate name-videos.csv
					Path: "My playlist with a duplicate name",
					Name: "My playlist with a duplicate name",
				},
			},

			DateTaken:   time.Date(int(2023), time.December, int(14), int(1), int(5), int(57), int(0), time.UTC).In(local),
			Latitude:    0,
			Longitude:   0,
			Altitude:    0,

			Trashed:     false,
			Archived:    false,
			FromPartner: false,
			Favorite:    false,

			FSys:        videos,
			FileSize:    3,
		},
		&browser.LocalAssetFile{
			FileName:    "`-=[]_,._~!@#$_^&_()_+{}_ 👱🏻🧟‍♀️👨‍❤️‍💋(2).mp4",
			Title:       "`-=[]\\;',./~!@#$%^\u0026*()_+{}|:\"? 👱🏻🧟‍♀️👨‍❤️‍💋‍👨👩‍👩‍👧‍👦🏳️‍⚧️🇵🇷 Z̤͔ͧ̑̓ä͖̭̈̇lͮ̒ͫǧ̗͚̚o̙̔ͮ̇͐̇ اختبار النص",
			Description: "",
			Albums:      []browser.LocalAlbum{
				browser.LocalAlbum{
					//Path: "My playlist with a duplicate name-videos.csv
					Path: "My playlist with a duplicate name",
					Name: "My playlist with a duplicate name",
				},
				browser.LocalAlbum{
					//Path: "My playlist with a duplicate name-videos.csv
					Path: "My playlist with a duplicate name",
					Name: "My playlist with a duplicate name",
				},
				browser.LocalAlbum{
					//Path: "My playlist with a duplicate name-videos.csv
					Path: "My playlist with a duplicate name",
					Name: "My playlist with a duplicate name",
				},
			},

			DateTaken:   time.Date(int(2023), time.December, int(17), int(14), int(14), int(46), int(0), time.UTC).In(local),
			Latitude:    0,
			Longitude:   0,
			Altitude:    0,

			Trashed:     false,
			Archived:    false,
			FromPartner: false,
			Favorite:    false,

			FSys:        videos,
			FileSize:    4,
		},
	}

	// Test Browse
	gotLafs := []*browser.LocalAssetFile{}
	assetChan := to.Browse(ctx)
assetLoop:
	for {
		laf, ok := <-assetChan;
		if !ok {
			break assetLoop;
		}
		gotLafs = append(gotLafs, laf)
	}

	if len(gotLafs) != len(wantLafs) {
		t.Errorf("Browse returned %d LocalAssetFiles instead of %d", len(gotLafs), len(wantLafs))
	}
	for i, _ := range gotLafs {
		// The order of the playlists in the data we read depends on
		// the order of the playlists in playlists.csv, which seems to
		// be random.  Also we don't really care in the first place,
		// so just make it predictable for the test:
		sort.Sort(LocalAlbumsByName(gotLafs[i].Albums))
		sort.Sort(LocalAlbumsByName(wantLafs[i].Albums))

		if !reflect.DeepEqual(gotLafs[i], wantLafs[i]) {
			want_json, _ := json.MarshalIndent(wantLafs[i], "", "	")
			got_json, _  := json.MarshalIndent(gotLafs[i], "", "	")
			t.Fatalf("Prepare returned\n%s\ninstead of\n%s\nfor index %d", got_json, want_json, i)
		}
	}
}
