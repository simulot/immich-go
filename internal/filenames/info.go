package filenames

import (
	"time"
)

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
