package upcmd

import (
	"fmt"
	"immich-go/immich"
	"immich-go/immich/assets"
	"path"
	"strings"
)

type AssetIndex struct {
	assets []*immich.Asset
	byHash map[string][]*immich.Asset
	byName map[string][]*immich.Asset
	byID   map[string]*immich.Asset
	albums []immich.AlbumSimplified
}

func (ai *AssetIndex) ReIndex() {
	ai.byHash = map[string][]*immich.Asset{}
	ai.byName = map[string][]*immich.Asset{}
	ai.byID = map[string]*immich.Asset{}

	for _, a := range ai.assets {
		ID := a.DeviceAssetID
		l := ai.byHash[a.Checksum]
		l = append(l, a)
		ai.byHash[a.Checksum] = l

		n := a.OriginalFileName
		l = ai.byName[n]
		l = append(l, a)
		ai.byName[n] = l
		ai.byID[ID] = a
	}
}

func (ai *AssetIndex) Len() int {
	return len(ai.assets)
}

func (ai *AssetIndex) AddLocalAsset(la *assets.LocalAssetFile) {
	sa := &immich.Asset{
		ID:               fmt.Sprintf("%s-%d", path.Base(la.Title), la.Size()),
		OriginalFileName: strings.TrimSuffix(path.Base(la.Title), path.Ext(la.Title)),
		ExifInfo: immich.ExifInfo{
			FileSizeInByte:   int(la.Size()),
			DateTimeOriginal: la.DateTakenCached(),
		},
		JustUploaded: true,
	}
	ID := fmt.Sprintf("%s-%d", sa.OriginalFileName, sa.ExifInfo.FileSizeInByte)
	ai.assets = append(ai.assets, sa)
	ai.byID[ID] = sa
	l := ai.byName[sa.OriginalFileName]
	l = append(l, sa)
	ai.byName[sa.OriginalFileName] = l
}
