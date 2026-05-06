package flickr

import (
	"strconv"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper"
)

// FlickrTag is a single tag entry in a per-photo JSON file.
type FlickrTag struct {
	Tag string `json:"tag"`
}

// FlickrMetadata represents the contents of a per-photo JSON file (photo_<id>.json).
// Note: the "albums" field in per-photo JSON is always empty in real Flickr exports;
// album membership is sourced exclusively from albums.json via albumIndex().
type FlickrMetadata struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	DateTaken   string      `json:"date_taken"`
	DateUpload  string      `json:"date_upload"`
	Tags        []FlickrTag `json:"tags"`
}

// FlickrAlbum is a single album entry within albums.json.
type FlickrAlbum struct {
	ID          string   `json:"id"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Photos      []string `json:"photos"`
}

// FlickrAlbums represents the top-level albums.json file.
type FlickrAlbums struct {
	Albums []FlickrAlbum `json:"albums"`
}

// AsMetadata converts a FlickrMetadata to an *assets.Metadata value ready for
// consumption by the asset pipeline.
//
// Date resolution order:
//  1. Parse date_taken as "2006-01-02 15:04:05" in local time.
//  2. If date_taken is empty or unparseable, interpret date_upload as a Unix
//     timestamp (seconds since epoch).
//  3. If both fail, DateTaken remains zero.
func (m FlickrMetadata) AsMetadata(file fshelper.FSAndName) *assets.Metadata {
	md := assets.Metadata{
		File:        file,
		FileName:    m.Name,
		Description: m.Description,
	}

	// Attempt to parse date_taken first.
	if m.DateTaken != "" {
		if t, err := time.ParseInLocation("2006-01-02 15:04:05", m.DateTaken, time.Local); err == nil {
			md.DateTaken = t
		}
	}

	// Fall back to date_upload if date_taken was empty or unparseable.
	if md.DateTaken.IsZero() && m.DateUpload != "" {
		if ts, err := strconv.ParseInt(m.DateUpload, 10, 64); err == nil && ts != 0 {
			md.DateTaken = time.Unix(ts, 0).In(time.Local)
		}
	}

	// Attach tags.
	for _, t := range m.Tags {
		md.AddTag(t.Tag)
	}

	return &md
}

// albumIndex inverts a FlickrAlbums value into a map from photo ID to the list
// of album titles that contain that photo.  The returned map is never nil.
func albumIndex(albums *FlickrAlbums) map[string][]string {
	result := make(map[string][]string)
	if albums == nil {
		return result
	}
	for _, album := range albums.Albums {
		for _, photoID := range album.Photos {
			result[photoID] = append(result[photoID], album.Title)
		}
	}
	return result
}


