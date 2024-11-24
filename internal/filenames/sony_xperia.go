package filenames

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/simulot/immich-go/internal/assets"
)

var sonyXperiaRE = regexp.MustCompile(`^DSC_(\d+)_BURST(\d+)(\D+)?(\..+)$`)

func (ic InfoCollector) SonyXperia(name string) (bool, assets.NameInfo) {
	parts := sonyXperiaRE.FindStringSubmatch(name)
	if len(parts) == 0 {
		return false, assets.NameInfo{}
	}
	ext := parts[4]
	info := assets.NameInfo{
		Radical: "BURST" + parts[2],
		Base:    name,
		IsCover: strings.Contains(parts[3], "COVER"),
		Ext:     strings.ToLower(ext),
		Type:    ic.SM.TypeFromExt(ext),
		Kind:    assets.KindBurst,
	}
	info.Index, _ = strconv.Atoi(parts[1])

	info.Taken, _ = time.ParseInLocation("20060102150405.000", parts[2][:14]+"."+parts[2][14:], ic.TZ)
	return true, info
}
