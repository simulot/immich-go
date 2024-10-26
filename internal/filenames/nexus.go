package filenames

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

/*

Nexus burst file name pattern
#100 stack: Huawei Nexus 6P

Burst
00001IMG_00001_BURST20171111030039.jpg
...
00014IMG_00014_BURST20171111030039.jpg
00015IMG_00015_BURST20171111030039_COVER.jpg
00000PORTRAIT_00000_BURST20190828181853475.jpg
00100lPORTRAIT_00100_BURST20181229213517346_COVER.jpg
00000IMG_00000_BURST20200607093330363_COVER.jpg
00000IMG_00000_BURST20190830164840873_COVER.jpg
00000IMG_00000_BURST20190830164840873.jpg


Regular
IMG_20171111_030055.jpg
IMG_20171111_030128.jpg
*/

var nexusRE = regexp.MustCompile(`^(\d+)\D+_\d+_(BURST\d+)(\D+)?(\..+)$`)

func (ic InfoCollector) Nexus(name string) (bool, NameInfo) {
	parts := nexusRE.FindStringSubmatch(name)
	if len(parts) == 0 {
		return false, NameInfo{}
	}
	ext := parts[4]
	info := NameInfo{
		Radical: parts[2],
		Base:    name,
		IsCover: strings.Contains(parts[3], "COVER"),
		Ext:     strings.ToLower(ext),
		Type:    ic.SM.TypeFromExt(ext),
		Kind:    KindBurst,
	}
	info.Index, _ = strconv.Atoi(parts[1])
	info.Taken, _ = time.ParseInLocation("20060102150405", parts[2][5:19], ic.TZ)
	return true, info
}
