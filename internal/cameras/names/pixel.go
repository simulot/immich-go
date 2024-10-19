package names

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Pixel burst file name pattern
// #94 stack: for Pixel 5 and Pixel 8 Pro naming schemes
// Google Pixel 5
// Normal - STACKS
// PXL_20231026_210642603.dng
// PXL_20231026_210642603.jpg

// Burst - DOES NOT STACK
// PXL_20231026_205755225.dng
// PXL_20231026_205755225.MP.jpg

// Google Pixel 8 Pro
// Normal - DOES NOT STACK
// PXL_20231207_032111247.RAW-02.ORIGINAL.dng
// PXL_20231207_032111247.RAW-01.COVER.jpg

// Burst - DOES NOT STACK
// PXL_20231207_032108788.RAW-02.ORIGINAL.dng
// PXL_20231207_032108788.RAW-01.MP.COVER.jpg

// PXL_20230330_184138390.MOTION-01.COVER.jpg
// PXL_20230330_184138390.MOTION-02.ORIGINAL.jpg
// PXL_20230330_201207251.jpg
// PXL_20230816_132648337.NIGHT.jpg
// PXL_20230817_175514506.PANO.jpg
// PXL_20230809_203029471.LONG_EXPOSURE-01.COVER.jpg
// PXL_20230809_203055470.LONG_EXPOSURE-01.COVER.jpg
// PXL_20231220_170358366.RAW-01.COVER.jpg
// PXL_20231220_170358366.RAW-02.ORIGINAL.dng

var pixelRE = regexp.MustCompile(`^(PXL_\d{8}_\d{9})((.*)?(\d{2}))?(.*)?(\..*)$`)

func (ic InfoCollector) Pixel(name string) (bool, NameInfo) {
	parts := pixelRE.FindStringSubmatch(name)
	if len(parts) == 0 {
		return false, NameInfo{}
	}
	ext := parts[6]
	info := NameInfo{
		Radical: parts[1],
		Base:    name,
		IsCover: strings.HasSuffix(parts[5], "COVER"),
		Ext:     ext,
		Type:    ic.SM.TypeFromExt(ext),
	}
	if parts[4] != "" {
		info.Index, _ = strconv.Atoi(parts[4])
	}
	switch {
	case strings.Contains(parts[3], "RAW"):
		info.Kind = KindRawJpg
	case strings.Contains(parts[3], "PORTRAIT"):
		info.Kind = KindPortrait
	case strings.Contains(parts[3], "NIGHT"):
		info.Kind = KindNight
	case strings.Contains(parts[3], "LONG_EXPOSURE"):
		info.Kind = KindLongExposure
	case strings.Contains(parts[3], "MOTION"):
		info.Kind = KindMotion
	}
	info.Taken, _ = time.ParseInLocation("20060102_150405", parts[1][4:19], time.UTC)
	return true, info
}
