package filenames

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

var sonyXperiaRE = regexp.MustCompile(`^DSC_(\d+)_BURST(\d+)(\D+)?(\..+)$`)

func (ic InfoCollector) SonyXperia(name string) (bool, NameInfo) {
	parts := sonyXperiaRE.FindStringSubmatch(name)
	if len(parts) == 0 {
		return false, NameInfo{}
	}
	ext := parts[4]
	info := NameInfo{
		Radical: "BURST" + parts[2],
		Base:    name,
		IsCover: strings.Contains(parts[3], "COVER"),
		Ext:     strings.ToLower(ext),
		Type:    ic.SM.TypeFromExt(ext),
		Kind:    KindBurst,
	}
	info.Index, _ = strconv.Atoi(parts[1])

	info.Taken, _ = time.ParseInLocation("20060102150405.000", parts[2][:14]+"."+parts[2][14:], ic.TZ)
	return true, info
}
