package names

import (
	"path"
	"strings"
	"time"

	"github.com/simulot/immich-go/internal/metadata"
)

type Kind int

const (
	KindNone Kind = iota
	KindBurst
	KindRawJpg
	KindEdited
	KindPortrait
	KindNight
	KindMotion
	KindLongExposure
)

type NameInfo struct {
	Base    string    // base name (with extension)
	Ext     string    // extension
	Radical string    // base name usable for grouping photos
	Type    string    // type of the asset  video, image
	Kind    Kind      // type of the series
	Index   int       // index of the asset in the series
	Taken   time.Time // date taken
	IsCover bool      // is this is the cover if the series
}

type InfoCollector struct {
	TZ *time.Location
	SM metadata.SupportedMedia
}

// nameMatcher analyze the name and return
// bool -> true when name is a part of a burst
// NameInfo -> the information extracted from the name
type nameMatcher func(name string) (bool, NameInfo)

// GetInfo analyze the name and return the information extracted from the name
func (ic InfoCollector) GetInfo(name string) NameInfo {
	for _, m := range []nameMatcher{ic.Pixel, ic.Samsung, ic.Nexus} {
		if ok, i := m(name); ok {
			return i
		}
	}

	// no matcher found, return a basic info
	t := metadata.TakeTimeFromName(name, ic.TZ)
	ext := path.Ext(name)

	return NameInfo{
		Base:    name,
		Radical: strings.TrimSuffix(name, ext),
		Taken:   t,
		Type:    ic.SM.TypeFromExt(ext),
	}
}
