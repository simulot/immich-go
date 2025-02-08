package filenames

import (
	"path"
	"strings"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/filetypes"
)

type InfoCollector struct {
	TZ *time.Location
	SM filetypes.SupportedMedia
}

// NewInfoCollector creates a new InfoCollector
func NewInfoCollector(tz *time.Location, sm filetypes.SupportedMedia) *InfoCollector {
	return &InfoCollector{
		TZ: tz,
		SM: sm,
	}
}

// nameMatcher analyze the name and return
// bool -> true when name is a part of a burst
// NameInfo -> the information extracted from the name
type nameMatcher func(name string) (bool, assets.NameInfo)

// GetInfo analyze the name and return the information extracted from the name
func (ic InfoCollector) GetInfo(name string) assets.NameInfo {
	base := path.Base(name)
	for _, m := range []nameMatcher{ic.Pixel, ic.Samsung, ic.Nexus, ic.Huawei, ic.SonyXperia} {
		if ok, i := m(base); ok {
			return i
		}
	}

	// no matcher found, return a basic info
	t := TakeTimeFromPath(name, ic.TZ)
	ext := path.Ext(base)

	return assets.NameInfo{
		Base:    base,
		Radical: strings.TrimSuffix(base, ext),
		Ext:     strings.ToLower(ext),
		Taken:   t,
		Type:    ic.SM.TypeFromExt(ext),
	}
}
