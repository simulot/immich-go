package filenames

import (
	"path"
	"strings"
	"time"

	"github.com/simulot/immich-go/internal/metadata"
)

type InfoCollector struct {
	TZ *time.Location
	SM metadata.SupportedMedia
}

// NewInfoCollector creates a new InfoCollector
func NewInfoCollector(tz *time.Location, sm metadata.SupportedMedia) *InfoCollector {
	return &InfoCollector{
		TZ: tz,
		SM: sm,
	}
}

// nameMatcher analyze the name and return
// bool -> true when name is a part of a burst
// NameInfo -> the information extracted from the name
type nameMatcher func(name string) (bool, NameInfo)

// GetInfo analyze the name and return the information extracted from the name
func (ic InfoCollector) GetInfo(name string) NameInfo {
	for _, m := range []nameMatcher{ic.Pixel, ic.Samsung, ic.Nexus, ic.Huawei} {
		if ok, i := m(name); ok {
			return i
		}
	}

	// no matcher found, return a basic info
	t := TakeTimeFromName(name, ic.TZ)
	ext := path.Ext(name)

	return NameInfo{
		Base:    name,
		Radical: strings.TrimSuffix(name, ext),
		Ext:     strings.ToLower(ext),
		Taken:   t,
		Type:    ic.SM.TypeFromExt(ext),
	}
}
