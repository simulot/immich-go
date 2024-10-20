package filenames

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

/*
Huawei burst file name pattern
IMG_20231014_183246_BURST001_COVER.jpg
IMG_20231014_183246_BURST002.jpg
IMG_20231014_183246_BURST003.jpg
*/
var huaweiRE = regexp.MustCompile(`^(IMG_\d{8}_\d{6})_BURST(\d{3})(?:_(\w+))?(\..+)$`)

func (ic InfoCollector) Huawei(name string) (bool, NameInfo) {
	parts := huaweiRE.FindStringSubmatch(name)
	if len(parts) == 0 {
		return false, NameInfo{}
	}
	ext := parts[4]
	info := NameInfo{
		Radical: parts[1],
		Base:    name,
		IsCover: strings.HasSuffix(parts[3], "COVER"),
		Ext:     strings.ToLower(ext),
		Type:    ic.SM.TypeFromExt(ext),
		Kind:    KindBurst,
	}
	info.Index, _ = strconv.Atoi(parts[2])
	info.Taken, _ = time.ParseInLocation("20060102_150405", parts[1][4:19], ic.TZ)
	return true, info
}
