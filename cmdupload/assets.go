package cmdupload

import (
	"fmt"
	"path"
	"strings"
	"sync"

	"github.com/simulot/immich-go/browser"
	"github.com/simulot/immich-go/immich"
)

type AssetIndex struct {
	lo sync.Mutex

	assets []*immich.Asset
	byHash map[string][]*immich.Asset
	byName map[string][]*immich.Asset
	byID   map[string]*immich.Asset
	// albums []immich.AlbumSimplified
}

func (ai *AssetIndex) ReIndex() {
	ai.lo.Lock()
	defer ai.lo.Unlock()

	elems := len(ai.assets)
	ai.byHash = make(map[string][]*immich.Asset, elems)
	ai.byName = make(map[string][]*immich.Asset, elems)
	ai.byID = make(map[string]*immich.Asset, elems)

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
	ai.lo.Lock()
	defer ai.lo.Unlock()

	return len(ai.assets)
}

func (ai *AssetIndex) AddLocalAsset(la *browser.LocalAssetFile, ImmichID string) {
	ai.lo.Lock()
	defer ai.lo.Unlock()

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

// ByName returns a list of [immich.Asset]s based on the given name.
// If no entity could be found, nil is returned.
func (ai *AssetIndex) ByName(name string) []*immich.Asset {
	ai.lo.Lock()
	defer ai.lo.Unlock()

	return ai.byName[name]
}

// ByHash returns a list of [immich.Asset]s based on the given hash.
// If no entity could be found, nil is returned.
func (ai *AssetIndex) ByHash(hash string) []*immich.Asset {
	ai.lo.Lock()
	defer ai.lo.Unlock()

	return ai.byHash[hash]
}

// ByID returns the [immich.Asset] based on the given ID.
// If no entity could be found, nil is returned.
func (ai *AssetIndex) ByID(id string) *immich.Asset {
	ai.lo.Lock()
	defer ai.lo.Unlock()

	return ai.byID[id]
}
