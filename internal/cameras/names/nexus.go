package names

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Nexus burst file name pattern
// #100 stack: Huawei Nexus 6P
//
// Burst
// 00001IMG_00001_BURST20171111030039.jpg
// ...
// 00014IMG_00014_BURST20171111030039.jpg
// 00015IMG_00015_BURST20171111030039_COVER.jpg
//
// Regular
// IMG_20171111_030055.jpg
// IMG_20171111_030128.jpg

var nexusRE = regexp.MustCompile(`^(\d{5})IMG_\d{5}(_(BURST\d{14})?(_COVER)?)?(\..{1,4})$`)

func (ic InfoCollector) Nexus(name string) (bool, NameInfo) {
	parts := nexusRE.FindStringSubmatch(name)
	if len(parts) == 0 {
		return false, NameInfo{}
	}
	ext := parts[5]
	info := NameInfo{
		Radical: parts[3],
		Base:    name,
		IsCover: strings.HasSuffix(parts[4], "COVER"),
		Ext:     ext,
		Type:    ic.SM.TypeFromExt(ext),
		Kind:    KindBurst,
	}
	info.Index, _ = strconv.Atoi(parts[1])
	info.Taken, _ = time.ParseInLocation("20060102150405", parts[3][5:19], ic.TZ)
	return true, info
}
