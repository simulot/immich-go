package assets

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/fshelper/cachereader"
)

/*
	Asset structure hold information on assets used for building immich assets.

	The asset is taken into a fs.FS system which doesn't implement anything else than a strait
	reader.
	fsys can be a zip file, a DirFS, or anything else.

	It implements a way to read a minimal quantity of data to be able to take a decision
	about chose a file or discard it.

	implements fs.File and fs.FileInfo, Stat

*/

type Asset struct {
	// File system and file name
	File     fshelper.FSAndName
	FileDate time.Time // File creation date
	ID       string    // Immich ID after upload

	// Common fields
	OriginalFileName string // File name as delivered to Immich/Google
	Description      string // Google Photos may a have description
	FileSize         int    // File size in bytes

	// Metadata for the process and the upload to Immich
	CaptureDate time.Time // Date of the capture
	Trashed     bool      // The asset is trashed
	Archived    bool      // The asset is archived
	FromPartner bool      // the asset comes from a partner
	Favorite    bool      // the asset is marked as favorite
	Rating      int       // the asset is marked with stars
	Albums      []Album   // List of albums the asset is in
	Tags        []Tag     // List of tags the asset is tagged with

	// Information inferred from the original file name
	NameInfo

	FromSideCar     *Metadata // Metadata extracted from a sidecar file (XMP or JSON)
	FromSourceFile  *Metadata // Metadata extracted from the file content (embedded metadata)
	FromApplication *Metadata // Metadata extracted from the application that created the file

	// GPS location
	Latitude  float64 // GPS latitude
	Longitude float64 // GPS longitude

	// buffer management
	cacheReader *cachereader.CacheReader
}

// Kind is the probable type of the image
type Kind int

const (
	KindNone Kind = iota
	KindBurst
	KindEdited
	KindPortrait
	KindNight
	KindMotion
	KindLongExposure
)

type NameInfo struct {
	Base       string    // base name (with extension)
	Ext        string    // extension
	Radical    string    // base name usable for grouping photos
	Type       string    // type of the asset  video, image
	Kind       Kind      // type of the series
	Index      int       // index of the asset in the series
	Taken      time.Time // date taken
	IsCover    bool      // is this is the cover if the series
	IsModified bool      // is this is a modified version of the original
}

func (a *Asset) SetNameInfo(ni NameInfo) {
	a.NameInfo = ni
}

func (a *Asset) UseMetadata(md *Metadata) *Metadata {
	if md == nil {
		return nil
	}
	a.Description = md.Description
	a.Latitude = md.Latitude
	a.Longitude = md.Longitude
	a.CaptureDate = md.DateTaken
	a.FromPartner = md.FromPartner
	a.Trashed = md.Trashed
	a.Archived = md.Archived
	a.Favorite = md.Favorited
	a.Rating = int(md.Rating)
	a.MergeAlbums(md.Albums)
	a.MergeTags(md.Tags)
	return md
}

// LogValue returns a slog.Value representing the LocalAssetFile's properties.
func (a Asset) LogValue() slog.Value {
	return slog.GroupValue(
		slog.Any("FileName", a.File),
		slog.Time("FileDate", a.FileDate),
		slog.String("Description", a.Description),
		slog.String("Title", a.OriginalFileName),
		slog.Int("FileSize", a.FileSize),
		slog.String("ID", a.ID),
		slog.Time("CaptureDate", a.CaptureDate),
		slog.Bool("Trashed", a.Trashed),
		slog.Bool("Archived", a.Archived),
		slog.Bool("FromPartner", a.FromPartner),
		slog.Bool("Favorite", a.Favorite),
		slog.Int("Stars", a.Rating),
		slog.String("Latitude", fmt.Sprintf("%.0f.xxxxx", a.Latitude)),
		slog.String("Longitude", fmt.Sprintf("%.0f.xxxxx", a.Longitude)),
	)
}

func (a *Asset) MergeAlbums(a2 []Album) {
	for _, album := range a2 {
		found := false
		for _, existingAlbum := range a.Albums {
			if existingAlbum.Title == album.Title {
				found = true
				break
			}
		}
		if !found {
			a.Albums = append(a.Albums, album)
		}
	}
}

func (a *Asset) MergeTags(t2 []Tag) {
	for _, tag := range t2 {
		found := false
		for _, existingTag := range a.Tags {
			if existingTag.Name == tag.Name {
				found = true
				break
			}
		}
		if !found {
			a.Tags = append(a.Tags, tag)
		}
	}
}
