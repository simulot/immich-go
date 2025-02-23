package upload

import (
	"path"
	"strings"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
)

type AssetIndex struct {
	assets          []*immich.Asset
	byHash          map[string][]*immich.Asset
	byName          map[string][]*immich.Asset
	byDeviceAssetID map[string]*immich.Asset
	// albums []immich.AlbumSimplified
}

func (ai *AssetIndex) ReIndex() {
	ai.byHash = map[string][]*immich.Asset{}
	ai.byName = map[string][]*immich.Asset{}
	ai.byDeviceAssetID = map[string]*immich.Asset{}

	for _, a := range ai.assets {
		ID := a.DeviceAssetID
		l := ai.byHash[a.Checksum]
		l = append(l, a)
		ai.byHash[a.Checksum] = l

		n := a.OriginalFileName
		l = ai.byName[n]
		l = append(l, a)
		ai.byName[n] = l
		ai.byDeviceAssetID[ID] = a
	}
}

func (ai *AssetIndex) Len() int {
	return len(ai.assets)
}

func (ai *AssetIndex) AddLocalAsset(la *assets.Asset, immichID string) {
	sa := &immich.Asset{
		ID:               immichID,
		DeviceAssetID:    la.DeviceAssetID,
		OriginalFileName: strings.TrimSuffix(path.Base(la.OriginalFileName), path.Ext(la.OriginalFileName)),
		ExifInfo: immich.ExifInfo{
			FileSizeInByte:   int64(la.FileSize),
			DateTimeOriginal: immich.ImmichExifTime{Time: la.CaptureDate},
			Latitude:         la.Latitude,
			Longitude:        la.Longitude,
		},
	}
	ai.assets = append(ai.assets, sa)
	ai.byDeviceAssetID[sa.DeviceAssetID] = sa
	l := ai.byName[sa.OriginalFileName]
	l = append(l, sa)
	ai.byName[sa.OriginalFileName] = l
}
