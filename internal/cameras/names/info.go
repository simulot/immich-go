package names

import "time"

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

// func GetInfo(name string) Info {
// 	for _, m := range []nameMatcher{pixelBurst, samsungBurst, nexusBurst} {
// 		if ok, i := m(name); ok {
// 			return i
// 		}
// 	}

// 	return Info{}
// }

// nameMatcher analyze the name and return
// bool -> true when name is a part of a burst
// string -> base name of the burst
// bool -> is this is the cover if the burst
type nameMatcher func(name string) (bool, NameInfo)

// // Samsung burst file name pattern
// // #99  stack: Samsung #99
// // 20231207_101605_001.jpg
// // 20231207_101605_002.jpg
// // 20231207_101605_xxx.jpg

// var samsungBurstRE = regexp.MustCompile(`^(\d{8}_\d{6})_(\d{3})$`)

// func samsungBurst(name string) *stackAsset {
// 	parts := samsungBurstRE.FindStringSubmatch(name)
// 	if len(parts) == 0 {
// 		return nil
// 	}
// 	return &stackAsset{parts[1], parts[2] == "001"}
// }

// // Nexus burst file name pattern
// // #100 stack: Huawei Nexus 6P
// //
// // Burst
// // 00001IMG_00001_BURST20171111030039.jpg
// // ...
// // 00014IMG_00014_BURST20171111030039.jpg
// // 00015IMG_00015_BURST20171111030039_COVER.jpg
// //
// // Regular
// // IMG_20171111_030055.jpg
// // IMG_20171111_030128.jpg

// var nexusBurstRE = regexp.MustCompile(`^\d{5}IMG_\d{5}_(BURST\d{14})(_COVER)?$`)

// func nexusBurst(name string) *stackAsset {
// 	parts := nexusBurstRE.FindStringSubmatch(name)
// 	if len(parts) == 0 {
// 		return nil
// 	}
// 	return &stackAsset{parts[1], parts[2] == "_COVER"}
// }

// // // Huawei burst file name pattern

// // var huaweiBurstRE = regexp.MustCompile(`^(.*)(_BURST\d+)(_COVER)?(\..*)$`)

// // func huaweiBurst(name string) (bool, string, bool) {
// // 	parts := huaweiBurstRE.FindStringSubmatch(name)
// // 	if len(parts) == 0 {
// // 		return false, "", false
// // 	}
// // 	return true, parts[1], parts[3] != ""
// // }
