package cmdupload

import (
	"fmt"
	"immich-go/browser"
	"immich-go/immich"
	"path"
	"strings"
)

type AssetIndex struct {
	assets []*immich.Asset
	byHash map[string][]*immich.Asset
	byName map[string][]*immich.Asset
	byID   map[string]*immich.Asset
	// albums []immich.AlbumSimplified
}

func (ai *AssetIndex) ReIndex() {
	ai.byHash = map[string][]*immich.Asset{}
	ai.byName = map[string][]*immich.Asset{}
	ai.byID = map[string]*immich.Asset{}

	for _, a := range ai.assets {
		ext := path.Ext(a.OriginalPath)
		ID := fmt.Sprintf("%s-%d", strings.ToUpper(path.Base(a.OriginalFileName)+ext), a.ExifInfo.FileSizeInByte)
		l := ai.byHash[a.Checksum]
		l = append(l, a)
		ai.byHash[a.Checksum] = l

		n := a.OriginalFileName + ext
		l = ai.byName[n]
		l = append(l, a)
		ai.byName[n] = l
		ai.byID[ID] = a
	}
}

func (ai *AssetIndex) Len() int {
	return len(ai.assets)
}

func (ai *AssetIndex) AddLocalAsset(la *browser.LocalAssetFile, ImmichID string) {
	sa := &immich.Asset{
		ID:               ImmichID,
		DeviceAssetID:    la.DeviceAssetID(),
		OriginalFileName: strings.TrimSuffix(path.Base(la.Title), path.Ext(la.Title)),
		ExifInfo: immich.ExifInfo{
			FileSizeInByte:   int(la.Size()),
			DateTimeOriginal: immich.ImmichTime{Time: la.DateTaken},
			Latitude:         la.Latitude,
			Longitude:        la.Longitude,
		},
		JustUploaded: true,
	}
	ai.assets = append(ai.assets, sa)
	ai.byID[sa.DeviceAssetID] = sa
	l := ai.byName[sa.OriginalFileName]
	l = append(l, sa)
	ai.byName[sa.OriginalFileName] = l
}
