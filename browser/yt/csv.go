package yt

import (
	"golang.org/x/text/unicode/norm"
	"strings"
	"time"
	"unicode/utf16"

	"github.com/simulot/immich-go/helpers/tzone"
)

// Read from "channels/channel.csv", this file provides metadata about the
// channel
type YouTubeChannel struct {
	ChannelID  string `csv:"Channel ID"`
	Title      string `csv:"Channel Title (Original)"`
	Visibility string `csv:"Channel Visibility"`
}

// Read from "playlists/playlists.csv", this file provides a list of all of
// the playlists.  Note that this doesn't provide the filename in which the
// playlist's videos can be found, and that is handled by Filename() because
// the process is insane.
type YouTubePlaylist struct {
	PlaylistID      string `csv:"Playlist ID"`
	Description     string `csv:"Playlist Description (Original)"`
	Title           string `csv:"Playlist Title (Original)"`
	TitleLanguage   string `csv:"Playlist Title (Original) Language"`
	CreateTimestamp string `csv:"Playlist Create Timestamp"`
	UpdateTimestamp string `csv:"Playlist Update Timestamp"`
	VideoOrder      string `csv:"Playlist Video Order"`
	Visibility      string `csv:"Playlist Visibility"`
}

// Read from "playlists/$playlist-videos.csv", this file provides a list of
// the videos of a particular playlist
type YouTubePlaylistVideo struct {
	VideoID         string `csv:"Video ID"`
	CreateTimestamp string `csv:"Playlist Video Creation Timestamp"`
}

// Read from "video metadata/video recordings.csv", this file provides
// metadata about the location the video was recorded
type YouTubeVideoRecording struct {
	VideoID   string  `csv:"Video ID"`
	Address   string  `csv:"Video Recording Address"`
	Altitude  float64 `csv:"Video Recording Altitude"`
	Latitude  float64 `csv:"Video Recording Latitude"`
	Longitude float64 `csv:"Video Recording Longitude"`
	PlaceID   string  `csv:"Place ID derived from Google's Places API"`
}

// Read from "video metadata/videos.csv", this file provides metadata
// about the video.  Note that neither this type nor YouTubeVideoRecording
// provides the filename of the video.  There is seemingly no relationship
// between the Duration and the file size, nor between the CreateTimestamp and
// the numbering, so I have chosen to believe that videos are numbered based
// on the order in which they appear in this file because I don't care.
type YouTubeVideo struct {
	VideoID          string `csv:"Video ID"`
	Duration         int    `csv:"Approx Duration (ms)"`
	Language         string `csv:"Video Audio Language"`
	Category         string `csv:"Video Category"`
	Description      string `csv:"Video Description (Original)"`
	ChannelID        string `csv:"Channel ID"`
	Title            string `csv:"Video Title (Original)"`
	Privacy          string `csv:"Privacy"`
	State            string `csv:"Video State"`
	CreateTimestamp  string `csv:"Video Create Timestamp"`
	PublishTimestamp string `csv:"Video Publish Timestamp"`
}

func (ytmd YouTubeChannel) CleanTitle() (string, bool) {
	title := strings.Trim(ytmd.Title, " ")
	if title != "" {
		return title, true
	} else {
		return "", false
	}
}

func (ytmd YouTubePlaylist) CleanTitle() (string, bool) {
	title := strings.Trim(ytmd.Title, " ")
	if title != "" {
		return title, true
	} else {
		return "", false
	}
}

// Generates the filename (under playlists) of this playlist
func (ytmd YouTubePlaylist) Filename() string {
	// YouTube is no better at creating filenames than is Google Photos.
	//
	// The process for creating the name of a playlist CSV seems to be:
	// 1. Start with the playlist name
	// 2. Delete any backslashes, semicolons, asterisks, pipes,
	//    colons, or question marks
	// 3. Replace any apsotrophes, slashes, percents, or quotes with
	//    underscores
	// 4. Append "-videos" to the filename
	// 5. Encode as UTF-16
	// 6. If the encoded string is longer than 47 UTF-16 code units,
	//    truncate to the shortest string that is *at least* 47 code units.
	//    Functionally this means if the 47th code unit is the middle of a
	//    Unicode code point, take the remaining code units to create that
	//    code point. †
	// 7. Decode back to a string
	// 8. Append ".csv"
	//
	// If you're thinking murder thoughts, you are correct.
	//
	// If you're also thinking, "Can't this lead to multiple playlists
	// having the same filename?" you are also correct.
	//
	// † Despite how convoluted as this is, I'm not sure I've gotten the
	// algorithm right.  I'm sure there are more shennanigans related to
	// yet darker corners of Unicode/UTF-8/UTF-16.

	title := ytmd.Title

	title = strings.ReplaceAll(title, "\\", "")
	title = strings.ReplaceAll(title, ";", "")
	title = strings.ReplaceAll(title, "|", "")
	title = strings.ReplaceAll(title, ":", "")
	title = strings.ReplaceAll(title, "?", "")

	title = strings.ReplaceAll(title, "'", "_")
	title = strings.ReplaceAll(title, "/", "_")
	title = strings.ReplaceAll(title, "%", "_")
	title = strings.ReplaceAll(title, "*", "_")
	title = strings.ReplaceAll(title, "\"", "_")

	title = title + "-videos"

	// I'm not actually sure if this is necessary but I've had enough
	// of dealing with this.  Hopefully YouTube is already outputting
	// normalized Unicode, but there's no reason to have any faith in
	// them.
	title = norm.NFC.String(title)

	runes := []rune(title)
	if len(utf16.Encode(runes)) > 47 {
		// Truncate the string until it's <= 47 code units
		for len(utf16.Encode(runes)) > 47 {
			runes = runes[:len(runes)-1]
		}
		// If the string is < 47 then add back the last code point
		if len(utf16.Encode(runes)) != 47 {
			runes = []rune(title)[:len(runes)+1]
		}
		title = string(runes)
	}

	return title + ".csv"
}

func (ytv YouTubeVideo) Filename() string {
	// This is identical to YouTubePlaylist.Filename() except that:
	// 1. The length is different!?
	// 2. No counter is added by this function
	// 3. No file extension is added by this function
	//
	// All of the proscriptions of YouTubePlaylist.Filename() apply here
	// as well

	title := ytv.Title

	title = strings.ReplaceAll(title, "\\", "")
	title = strings.ReplaceAll(title, ";", "")
	title = strings.ReplaceAll(title, "|", "")
	title = strings.ReplaceAll(title, ":", "")
	title = strings.ReplaceAll(title, "?", "")

	title = strings.ReplaceAll(title, "'", "_")
	title = strings.ReplaceAll(title, "/", "_")
	title = strings.ReplaceAll(title, "%", "_")
	title = strings.ReplaceAll(title, "*", "_")
	title = strings.ReplaceAll(title, "\"", "_")

	// I'm not actually sure if this is necessary but I've had enough
	// of dealing with this.  Hopefully YouTube is already outputting
	// normalized Unicode, but there's no reason to have any faith in
	// them.
	title = norm.NFC.String(title)

	runes := []rune(title)
	if len(utf16.Encode(runes)) > 43 {
		// Truncate the string until it's <= 43 code units
		for len(utf16.Encode(runes)) > 43 {
			runes = runes[:len(runes)-1]
		}
		// If the string is < 43 then add back the last code point
		if len(utf16.Encode(runes)) != 43 {
			runes = []rune(title)[:len(runes)+1]
		}
		title = string(runes)
	}

	return title
}
func (ytv YouTubeVideo) CleanTitle() (string, bool) {
	title := strings.Trim(ytv.Title, " ")
	if title != "" {
		return title, true
	} else {
		return "", false
	}
}

func (ytv YouTubePlaylist) CleanDescription() (string, bool) {
	title := strings.Trim(ytv.Description, " ")
	if title != "" {
		return title, true
	} else {
		return "", false
	}
}

func (ytv YouTubeVideo) CleanDescription() (string, bool) {
	title := strings.Trim(ytv.Description, " ")
	if title != "" {
		return title, true
	} else {
		return "", false
	}
}

func (ytv YouTubeVideo) Time() time.Time {
	var t time.Time
	var err error
	t, err = time.Parse(time.RFC3339, ytv.CreateTimestamp)
	if err != nil {

	}
	local, _ := tzone.Local()
	return t.In(local)
}
