package yt_test

import (
	"encoding/json"
	"io/fs"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/simulot/immich-go/browser/yt"
	"github.com/simulot/immich-go/helpers/fshelper"
	"github.com/simulot/immich-go/helpers/tzone"
)

func TestReadYouTubeChannel(t *testing.T) {
	want := []yt.YouTubeChannel{
		yt.YouTubeChannel{
			ChannelID:  "kb3ZF7Rwt2jc2MvVG1kyaze9",
			Title:      "Jonathan Stafford",
			Visibility: "Public",
		},
	}

	var got []yt.YouTubeChannel
	fs := os.DirFS("TEST_DATA/20240623T224719/Takeout/YouTube and YouTube Music/channels/")
	got, err := fshelper.ReadCSV[yt.YouTubeChannel](fs, "channel.csv")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	if len(got) != len(want) {
		t.Errorf("ReadCSV returned %d rows instead of %d", len(got), len(want))
	}

	for i := 0; i < len(want); i++ {
		if !reflect.DeepEqual(got[i], want[i]) {
			want_json, _ := json.Marshal(want[i])
			got_json, _  := json.Marshal(got[i])
			t.Fatalf("ReadCSV returned\n%s\ninstead of\n%s\nfor index %d", got_json, want_json, i)
		}
	}
}

func TestReadYouTubePlaylist(t *testing.T) {
	want := []yt.YouTubePlaylist{
		yt.YouTubePlaylist{
			PlaylistID:      "0q2fQnYVgBZ97Pxa0dUTqdHBtk3B/xeyta",
			Description:     "",
			Title:           "😀orem😁ipsum😂dolor😃sit😄amet😅😆😇😈😉😊😋😌😍😎😏😐😑😒😓😕😖😗😘😙😚😛😜😝😞😟😠😡😢😣😤😥😦😧😨😩😪😫😬😭😮😯😰😱😲😳😴😵😶😷😸😹😺😻😼😽😾😿🙀🙁🙂🙃🙄🙅🙆🙇🙈🙉🙊🙋🙌🙍🙎🙏",
			TitleLanguage:   "",
			CreateTimestamp: "2023-12-19T22:19:00+00:00",
			UpdateTimestamp: "2023-12-19T22:19:09+00:00",
			VideoOrder:      "Manual",
			Visibility:      "Private",
		},
		yt.YouTubePlaylist{
			PlaylistID:      "+RSTyiTrFuyMrjlZpuQxw7d3LPzCMc8LaD",
			Description:     "",
			Title:           "😀😁😂😃😄😅😆😇😈😉😊😋😌😍😎😏😐😑😒😓😕😖😗😘😙😚😛😜😝😞😟😠😡😢😣😤😥😦😧😨😩😪😫😬😭😮😯😰😱😲😳😴😵😶😷😸😹😺😻😼😽😾😿🙀🙁🙂🙃🙄🙅🙆🙇🙈🙉🙊tema🙋tis🙌rolod🙍muspi🙎meroL🙏",
			TitleLanguage:   "",
			CreateTimestamp: "2023-12-19T01:07:46+00:00",
			UpdateTimestamp: "2023-12-19T01:12:45+00:00",
			VideoOrder:      "Manual",
			Visibility:      "Private",
		},
		yt.YouTubePlaylist{
			PlaylistID:      "NdhO/BxkyoiY1OaA98FdsEKotGUIIkenBX",
			Description:     "",
			Title:           "😀Lorem😁ipsum😂dolor😃sit😄amet😅😆😇😈😉😊😋😌😍😎😏😐😑😒😓😕😖😗😘😙😚😛😜😝😞😟😠😡😢😣😤😥😦😧😨😩😪😫😬😭😮😯😰😱😲😳😴😵😶😷😸😹😺😻😼😽😾😿🙀🙁🙂🙃🙄🙅🙆🙇🙈🙉🙊🙋🙌🙍🙎🙏",
			TitleLanguage:   "",
			CreateTimestamp: "2023-12-19T01:06:07+00:00",
			UpdateTimestamp: "2023-12-19T01:12:15+00:00",
			VideoOrder:      "Manual",
			Visibility:      "Private",
		},
		yt.YouTubePlaylist{
			PlaylistID:      "Vrhwj5jEY4aJ9mssxYIlFS0YEd+YzbSCq3",
			Description:     "",
			Title:           "My very long playlist title 0123456789 ABCD",
			TitleLanguage:   "",
			CreateTimestamp: "2023-12-18T23:26:38+00:00",
			UpdateTimestamp: "2023-12-18T23:26:44+00:00",
			VideoOrder:      "Manual",
			Visibility:      "Private",
		},
		yt.YouTubePlaylist{
			PlaylistID:      "/dvNHrUBJP17nJrNqt/HaQXHohMf0pR8ZA",
			Description:     "",
			Title:           "My very long playlist title 0123456789 ABCDE",
			TitleLanguage:   "",
			CreateTimestamp: "2023-12-18T23:25:46+00:00",
			UpdateTimestamp: "2023-12-18T23:26:27+00:00",
			VideoOrder:      "Manual",
			Visibility:      "Private",
		},
		yt.YouTubePlaylist{
			PlaylistID:      "yrM0LUMa/lEHDnJ7UR2cIetuzOcNCWxBnD",
			Description:     "",
			Title:           "My very long playlist title 0123456789 ABCDEF",
			TitleLanguage:   "",
			CreateTimestamp: "2023-12-18T23:25:40+00:00",
			UpdateTimestamp: "2023-12-18T23:26:19+00:00",
			VideoOrder:      "Manual",
			Visibility:      "Private",
		},
		yt.YouTubePlaylist{
			PlaylistID:      "19XtqwT0XgdNWjT3W9JbQfMEYI8uFiW9yI",
			Description:     "",
			Title:           "My very long playlist title 0123456789 ABCDEFG",
			TitleLanguage:   "",
			CreateTimestamp: "2023-12-18T23:25:31+00:00",
			UpdateTimestamp: "2023-12-18T23:26:09+00:00",
			VideoOrder:      "Manual",
			Visibility:      "Private",
		},
		yt.YouTubePlaylist{
			PlaylistID:      "8eN2mETnoqHuqDoxXq3nLApeBxdqg7r9qU",
			Description:     "",
			Title:           "My very long playlist title 0123456789 ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmnopqrstuvwxyz 0123456789 ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmnopqrs",
			TitleLanguage:   "",
			CreateTimestamp: "2023-12-17T14:37:14+00:00",
			UpdateTimestamp: "2023-12-17T14:37:31+00:00",
			VideoOrder:      "Manual",
			Visibility:      "Private",
		},
		yt.YouTubePlaylist{
			PlaylistID:      "NnnqWkLMzsQ40on1sPO3D5egybOaP2cra/",
			Description:     "",
			Title:           "Z̤͔ͧ̑̓ä͖̭̈̇lͮ̒ͫǧ̗͚̚o̙̔ͮ̇͐̇ اختبار النص",
			TitleLanguage:   "",
			CreateTimestamp: "2023-12-17T14:33:16+00:00",
			UpdateTimestamp: "2023-12-17T14:33:22+00:00",
			VideoOrder:      "Manual",
			Visibility:      "Private",
		},
		yt.YouTubePlaylist{
			PlaylistID:      "CagoNmT9b8xzQ/JDLR1cTyjQ+R2dWIlkho",
			Description:     "",
			Title:           "👱👱🏻👱🏼👱🏽👱🏾👱🏿 🧟‍♀️🧟‍♂️ 👨‍❤️‍💋‍👨👩‍👩‍👧‍👦🏳️‍⚧️🇵🇷",
			TitleLanguage:   "",
			CreateTimestamp: "2023-12-17T14:32:59+00:00",
			UpdateTimestamp: "2023-12-17T14:33:08+00:00",
			VideoOrder:      "Manual",
			Visibility:      "Private",
		},
		yt.YouTubePlaylist{
			PlaylistID:      "89uIs9+aFZ0rj76rdGXO2xeTQh/kz0aMCB",
			Description:     "",
			Title:           "My playlist with a duplicate name",
			TitleLanguage:   "",
			CreateTimestamp: "2023-12-17T14:03:27+00:00",
			UpdateTimestamp: "2023-12-17T14:16:09+00:00",
			VideoOrder:      "Manual",
			Visibility:      "Private",
		},
		yt.YouTubePlaylist{
			PlaylistID:      "qfyrziKWJZA/u+O7qJ6b4xxx3zjH4zjpED",
			Description:     "",
			Title:           "My playlist with a duplicate name",
			TitleLanguage:   "",
			CreateTimestamp: "2023-12-17T14:03:21+00:00",
			UpdateTimestamp: "2023-12-17T14:27:03+00:00",
			VideoOrder:      "Manual",
			Visibility:      "Private",
		},
		yt.YouTubePlaylist{
			PlaylistID:      "ozVxmuJGoBR+aT0FBghvOk+/j3a3JmwZPL",
			Description:     "",
			Title:           "My playlist with a duplicate name",
			TitleLanguage:   "",
			CreateTimestamp: "2023-12-17T14:03:17+00:00",
			UpdateTimestamp: "2023-12-17T14:27:18+00:00",
			VideoOrder:      "Manual",
			Visibility:      "Private",
		},
		yt.YouTubePlaylist{
			PlaylistID:      "yojzPdFgHjBNXsUcmTsmo6g1hTqsIkZUSB",
			Description:     "",
			Title:           "My very long playlist title 0123456789 ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmnopqrstuvwxyz 0123456789 ABCDEFGHIJKLMNOPQRSTUVWXYZ abcdefghijklmnopqrs",
			TitleLanguage:   "",
			CreateTimestamp: "2023-12-17T14:02:54+00:00",
			UpdateTimestamp: "2023-12-17T14:27:38+00:00",
			VideoOrder:      "Manual",
			Visibility:      "Private",
		},
		yt.YouTubePlaylist{
			PlaylistID:      "ykzP/AtWUAgtD6kfpZyk+CCipFNPlh27FA",
			Description:     "",
			Title:           "`-=[]\\;',./~!@#$%^&*()_+{}|:\"?",
			TitleLanguage:   "",
			CreateTimestamp: "2023-12-17T13:57:31+00:00",
			UpdateTimestamp: "2023-12-17T14:32:42+00:00",
			VideoOrder:      "Manual",
			Visibility:      "Private",
		},
		yt.YouTubePlaylist{
			PlaylistID:      "/63ek5JSj2ZcQSXaBiTslzKRSb+kK015UI",
			Description:     "This is my playlist",
			Title:           "A playlist",
			TitleLanguage:   "",
			CreateTimestamp: "2023-12-15T01:47:18+00:00",
			UpdateTimestamp: "2023-12-15T01:48:39+00:00",
			VideoOrder:      "Manual",
			Visibility:      "Private",
		},
		yt.YouTubePlaylist{
			PlaylistID:      "ldyZEf/SuTx9DgVls+WTopx7BG8Ufi6kl4",
			Description:     "",
			Title:           "Watch later",
			TitleLanguage:   "en_US",
			CreateTimestamp: "2015-03-22T06:43:50+00:00",
			UpdateTimestamp: "2024-05-10T01:25:36+00:00",
			VideoOrder:      "Manual",
			Visibility:      "Private",
		},
	}

	var got []yt.YouTubePlaylist
	fs := os.DirFS("TEST_DATA/20240623T224719/Takeout/YouTube and YouTube Music/playlists/")
	got, err := fshelper.ReadCSV[yt.YouTubePlaylist](fs, "playlists.csv")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(got) != len(want) {
		t.Errorf("ReadCSV returned %d rows instead of %d", len(got), len(want))
	}

	for i := 0; i < len(want); i++ {
		if !reflect.DeepEqual(got[i], want[i]) {
			got_json, _ := json.Marshal(got[i])
			want_json, _ := json.Marshal(want[i])
			t.Fatalf("ReadCSV returned\n%s\ninstead of\n%s\nfor index %d", got_json, want_json, i)
		}
	}
}

func TestReadYouTubePlaylistVideo(t *testing.T) {
	want := []yt.YouTubePlaylistVideo{
		yt.YouTubePlaylistVideo{
			VideoID:         "a+q6oaXj7dH",
			CreateTimestamp: "2023-12-15T01:48:39+00:00",
		},
		yt.YouTubePlaylistVideo{
			VideoID:         "pI3tVoMUwz5",
			CreateTimestamp: "2023-12-15T01:48:39+00:00",
		},
		yt.YouTubePlaylistVideo{
			VideoID:         "PJvZQ6mMSBf",
			CreateTimestamp: "2023-12-15T01:48:39+00:00",
		},
	}

	var got []yt.YouTubePlaylistVideo
	fs := os.DirFS("TEST_DATA/20240623T224719/Takeout/YouTube and YouTube Music/playlists/")
	got, err := fshelper.ReadCSV[yt.YouTubePlaylistVideo](fs, "A playlist-videos.csv")
	if err != nil {
		t.Errorf("unexpected error: %s", err)
		return
	}

	if len(got) != len(want) {
		t.Errorf("ReadCSV returned %d rows instead of %d", len(got), len(want))
		return
	}

	for i := 0; i < len(want); i++ {
		if !reflect.DeepEqual(got[i], want[i]) {
			got_json, _ := json.Marshal(got[i])
			want_json, _ := json.Marshal(want[i])
			t.Errorf("ReadCSV returned\n%s\ninstead of\n%s\nfor index %d", got_json, want_json, i)
			return
		}
	}
}

func TestReadYouTubeVideoRecording(t *testing.T) {
	want := []yt.YouTubeVideoRecording{
		yt.YouTubeVideoRecording{
			VideoID:   "PJvZQ6mMSBf",
			Address:   "",
			Altitude:  0,
			Latitude:  0,
			Longitude: 0,
			PlaceID:   "",
		},
		yt.YouTubeVideoRecording{
			VideoID:   "rl1vcIiguJV",
			Address:   "",
			Altitude:  0,
			Latitude:  0,
			Longitude: 0,
			PlaceID:   "",
		},
		yt.YouTubeVideoRecording{
			VideoID:   "pI3tVoMUwz5",
			Address:   "The White House",
			Altitude:  0,
			Latitude:  38.8977,
			Longitude: -77.0365,
			PlaceID:   "ChIJ37HL3ry3t4kRv3YLbdhpWXE",
		},
		yt.YouTubeVideoRecording{
			VideoID:   "d5IMr4n6DIh",
			Address:   "",
			Altitude:  0,
			Latitude:  0,
			Longitude: 0,
			PlaceID:   "",
		},
		yt.YouTubeVideoRecording{
			VideoID:   "a+q6oaXj7dH",
			Address:   "",
			Altitude:  0,
			Latitude:  0,
			Longitude: 0,
			PlaceID:   "",
		},
		yt.YouTubeVideoRecording{
			VideoID:   "NTOBfooePHb",
			Address:   "",
			Altitude:  0,
			Latitude:  0,
			Longitude: 0,
			PlaceID:   "",
		},
	}

	var got []yt.YouTubeVideoRecording
	fs := os.DirFS("TEST_DATA/20240623T224719/Takeout/YouTube and YouTube Music/video metadata/")
	got, err := fshelper.ReadCSV[yt.YouTubeVideoRecording](fs, "video recordings.csv")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(got) != len(want) {
		t.Errorf("ReadCSV returned %d rows instead of %d", len(got), len(want))
	}

	for i := 0; i < len(want); i++ {
		if !reflect.DeepEqual(got[i], want[i]) {
			got_json, _ := json.Marshal(got[i])
			want_json, _ := json.Marshal(want[i])
			t.Fatalf("ReadCSV returned\n%s\ninstead of\n%s\nfor index %d", got_json, want_json, i)
		}
	}
}

func TestReadYouTubeVideo(t *testing.T) {
	want := []yt.YouTubeVideo{
		yt.YouTubeVideo{
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
		yt.YouTubeVideo{
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
		yt.YouTubeVideo{
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
		yt.YouTubeVideo{
			VideoID:          "d5IMr4n6DIh",
			Duration:         16000,
			Language:         "en-US",
			Category:         "People",
			Description:      "",
			ChannelID:        "kb3ZF7Rwt2jc2MvVG1kyaze9",
			Title:            "`-=[]\\;',./~!@#$%^&*()_+{}|:\"? 👱🏻🧟‍♀️👨‍❤️‍💋‍👨👩‍👩‍👧‍👦🏳️‍⚧️🇵🇷 Z̤͔ͧ̑̓ä͖̭̈̇lͮ̒ͫǧ̗͚̚o̙̔ͮ̇͐̇ اختبار النص",
			Privacy:          "Private",
			State:            "Processed",
			CreateTimestamp:  "2023-12-14T00:36:14+00:00",
			PublishTimestamp: "",
		},
		yt.YouTubeVideo{
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
		yt.YouTubeVideo{
			VideoID:          "NTOBfooePHb",
			Duration:         16000,
			Language:         "en-US",
			Category:         "People",
			Description:      "",
			ChannelID:        "kb3ZF7Rwt2jc2MvVG1kyaze9",
			Title:            "`-=[]\\;',./~!@#$%^&*()_+{}|:\"? 👱🏻🧟‍♀️👨‍❤️‍💋‍👨👩‍👩‍👧‍👦🏳️‍⚧️🇵🇷 Z̤͔ͧ̑̓ä͖̭̈̇lͮ̒ͫǧ̗͚̚o̙̔ͮ̇͐̇ اختبار النص",
			Privacy:          "Private",
			State:            "Processed",
			CreateTimestamp:  "2023-12-17T14:14:46+00:00",
			PublishTimestamp: "",
		},
	}

	var got []yt.YouTubeVideo
	fs := os.DirFS("TEST_DATA/20240623T224719/Takeout/YouTube and YouTube Music/video metadata/")
	got, err := fshelper.ReadCSV[yt.YouTubeVideo](fs, "videos.csv")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	if len(got) != len(want) {
		t.Errorf("ReadCSV returned %d rows instead of %d", len(got), len(want))
	}

	for i := 0; i < len(want); i++ {
		if !reflect.DeepEqual(got[i], want[i]) {
			got_json, _ := json.Marshal(got[i])
			want_json, _ := json.Marshal(want[i])
			t.Fatalf("ReadCSV returned\n%s\ninstead of\n%s\nfor index %d", got_json, want_json, i)
		}
	}
}

func TestCleanChannelTitle(t *testing.T) {
	md := yt.YouTubeChannel{
		Title: "      ",
	}

	title, ok := md.CleanTitle()
	if ok {
		t.Errorf("CleanTitle() was ok when it should not have been")
	}

	md.Title = "    ti tle     "
	title, ok = md.CleanTitle()
	if !ok {
		t.Errorf("CleanTitle() was not ok when it should have been")
	}
	if title != "ti tle" {
		t.Errorf("CleanTitle() return `%s` when it should have returned `title`", title)
	}
}

func TestCleanPlaylistTitle(t *testing.T) {
	md := yt.YouTubePlaylist{
		Title: "      ",
	}

	title, ok := md.CleanTitle()
	if ok {
		t.Errorf("CleanTitle() was ok when it should not have been")
	}

	md.Title = "    ti tle     "
	title, ok = md.CleanTitle()
	if !ok {
		t.Errorf("CleanTitle() was not ok when it should have been")
	}
	if title != "ti tle" {
		t.Errorf("CleanTitle() return `%s` when it should have returned `title`", title)
	}
}

func TestPlaylistFilename(t *testing.T) {
	dir := os.DirFS("TEST_DATA/20240623T224719/Takeout/YouTube and YouTube Music/playlists")
	playlists, err := fshelper.ReadCSV[yt.YouTubePlaylist](dir, "playlists.csv")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}

	for i, playlist := range playlists {
		if playlist.Title != "Watch later" {
			filename := playlist.Filename()
			_, err := fs.Stat(dir, filename)
			if err != nil {
				t.Errorf("couldn't find filename\n%s\nfrom title\n%s\nat index %d:\n%s", filename, playlist.Title, i, err)
			}
		}
	}
}

func TestVideoFilename(t *testing.T) {
	testCases := []struct {
		title    string
		expected string
	}{
		{
			title:    "A description of Serenade #2",
			expected: "A description of Serenade #2",
		},
		{
			title:    "Serenade #1",
			expected: "Serenade #1",
		},
		{
			title:    "I manually set the location",
			expected: "I manually set the location",
		},
		{
			title:    "`-=[]\\;',./~!@#$%^&*()_+{}|:\"? 👱🏻🧟‍♀️👨‍❤️‍💋‍👨👩‍👩‍👧‍👦🏳️‍⚧️🇵🇷 Z̤͔ͧ̑̓ä͖̭̈̇lͮ̒ͫǧ̗͚̚o̙̔ͮ̇͐̇ اختبار النص",
			expected: "`-=[]_,._~!@#$_^&_()_+{}_ 👱🏻🧟‍♀️👨‍❤️‍💋",
		},
	}

	for _, tc := range testCases {
		sut := yt.YouTubeVideo{
			Title: tc.title,
		}
		filename := sut.Filename()
		if filename != tc.expected {
			t.Errorf("Got\n%s\ninstead of\n%s\nfrom\n%s", filename, tc.expected, tc.title)
		}
	}
}

func TestCleanVideoTitle(t *testing.T) {
	md := yt.YouTubeVideo{
		Title: "      ",
	}

	title, ok := md.CleanTitle()
	if ok {
		t.Errorf("CleanTitle() was ok when it should not have been")
	}

	md.Title = "    ti tle     "
	title, ok = md.CleanTitle()
	if !ok {
		t.Errorf("CleanTitle() was not ok when it should have been")
	}
	if title != "ti tle" {
		t.Errorf("CleanTitle() return `%s` when it should have returned `title`", title)
	}
}

func TestCleanPlaylistDescription(t *testing.T) {
	md := yt.YouTubePlaylist{
		Description: "      ",
	}

	description, ok := md.CleanDescription()
	if ok {
		t.Errorf("CleanDescription() was ok when it should not have been")
	}

	md.Description = "    desc rip tion     "
	description, ok = md.CleanDescription()
	if !ok {
		t.Errorf("CleanDescription() was not ok when it should have been")
	}
	if description != "desc rip tion" {
		t.Errorf("CleanDescription() return `%s` when it should have returned `desc rip tion`", description)
	}
}

func TestCleanVideoDescription(t *testing.T) {
	md := yt.YouTubeVideo{
		Description: "      ",
	}

	description, ok := md.CleanDescription()
	if ok {
		t.Errorf("CleanDescription() was ok when it should not have been")
	}

	md.Description = "    desc rip tion     "
	description, ok = md.CleanDescription()
	if !ok {
		t.Errorf("CleanDescription() was not ok when it should have been")
	}
	if description != "desc rip tion" {
		t.Errorf("CleanDescription() return `%s` when it should have returned `desc rip tion`", description)
	}
}

func TestYouTubeVideoTime(t *testing.T) {
	local, _ := tzone.Local()
	want := time.Date(int(2023), time.December, int(14), int(1), int(5), int(57), int(0), time.UTC).In(local)

	md := yt.YouTubeVideo{
		CreateTimestamp: "2023-12-14T01:05:57+00:00",
	}
	got := md.Time()
	if got != want {
		t.Fatalf("YouTubeVideo.Time() returned %s instead of %s", got, want)
	}
}
