package myimmich

import (
	"fmt"
	"path"
	"strings"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
)

type AssetIndex struct {
	assets []*immich.Asset
	ByHash map[string][]*immich.Asset
	ByName map[string][]*immich.Asset
	ByID   map[string]*immich.Asset
}

func NewAssetIndex(assets []*immich.Asset) *AssetIndex {
	res := &AssetIndex{
		assets: assets,
	}
	res.reIndex()
	return res
}

func (ai *AssetIndex) reIndex() {
	ai.ByHash = map[string][]*immich.Asset{}
	ai.ByName = map[string][]*immich.Asset{}
	ai.ByID = map[string]*immich.Asset{}

	for _, a := range ai.assets {
		ID := fmt.Sprintf("%s-%d", a.OriginalFileName, a.ExifInfo.FileSizeInByte)
		l := ai.ByHash[a.Checksum]
		l = append(l, a)
		ai.ByHash[a.Checksum] = l

		n := a.OriginalFileName
		l = ai.ByName[n]
		l = append(l, a)
		ai.ByName[n] = l
		ai.ByID[ID] = a
	}
}

func (ai *AssetIndex) Len() int {
	return len(ai.assets)
}

func (ai *AssetIndex) AddLocalAsset(la *assets.Asset, immichID string) {
	sa := &immich.Asset{
		ID:               immichID,
		DeviceAssetID:    la.DeviceAssetID(),
		OriginalFileName: strings.TrimSuffix(path.Base(la.Title), path.Ext(la.Title)),
		ExifInfo: immich.ExifInfo{
			FileSizeInByte:   la.Size(),
			DateTimeOriginal: immich.ImmichTime{Time: la.CaptureDate},
			Latitude:         la.Latitude,
			Longitude:        la.Longitude,
		},
		JustUploaded: true,
	}
	ai.assets = append(ai.assets, sa)
	ai.ByID[sa.DeviceAssetID] = sa
	l := ai.ByName[sa.OriginalFileName]
	l = append(l, sa)
	ai.ByName[sa.OriginalFileName] = l
}
