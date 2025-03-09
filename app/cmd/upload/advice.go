package upload

import (
	"fmt"
	"math"
	"path"
	"sync"
	"sync/atomic"
	"time"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/gen/syncmap"
	"github.com/simulot/immich-go/internal/gen/syncset"
)

// - - go:generate stringer -type=AdviceCode
type AdviceCode int

func (a AdviceCode) String() string {
	switch a {
	case IDontKnow:
		return "IDontKnow"
	// case SameNameOnServerButNotSure:
	// 	return "SameNameOnServerButNotSure"
	case SmallerOnServer:
		return "SmallerOnServer"
	case BetterOnServer:
		return "BetterOnServer"
	case SameOnServer:
		return "SameOnServer"
	case NotOnServer:
		return "NotOnServer"
	}
	return fmt.Sprintf("advice(%d)", a)
}

const (
	IDontKnow AdviceCode = iota
	SmallerOnServer
	BetterOnServer
	SameOnServer
	NotOnServer
)

type immichIndex struct {
	lock sync.Mutex

	// map of assetID to asset
	immichAssets *syncmap.SyncMap[string, *assets.Asset]

	// set of uploaded assets during the current session
	uploadedAssets *syncset.Set[string]

	// map of base name to assetID
	byName *syncmap.SyncMap[string, []string]

	// map of deviceID to assetID
	byDeviceID *syncmap.SyncMap[string, string]

	assetNumber int64
}

func newAssetIndex() *immichIndex {
	return &immichIndex{
		immichAssets:   syncmap.New[string, *assets.Asset](),
		byName:         syncmap.New[string, []string](),
		byDeviceID:     syncmap.New[string, string](),
		uploadedAssets: syncset.New[string](),
	}
}

// Add adds an asset to the index.
// returns true if the asset was added, false if it was already present.
// the returned asset is the existing asset if it was already present.
func (ii *immichIndex) addImmichAsset(ia *immich.Asset) (*assets.Asset, bool) {
	ii.lock.Lock()
	defer ii.lock.Unlock()

	if ia.ID == "" {
		panic("asset ID is empty")
	}

	if existing, ok := ii.immichAssets.Load(ia.ID); ok {
		return existing, false
	}
	a := ia.AsAsset()
	return ii.add(a), true
}

func (ii *immichIndex) addLocalAsset(ia *assets.Asset) (*assets.Asset, bool) {
	ii.lock.Lock()
	defer ii.lock.Unlock()

	if ia.ID == "" {
		panic("asset ID is empty")
	}
	if existing, ok := ii.immichAssets.Load(ia.ID); ok {
		return existing, false
	}
	if !ii.uploadedAssets.Add(ia.ID) {
		panic("addLocalAsset asset already uploaded")
	}
	return ii.add(ia), true
}

func (ii *immichIndex) getByID(id string) *assets.Asset {
	a, _ := ii.immichAssets.Load(id)
	return a
}

func (ii *immichIndex) len() int {
	return int(atomic.LoadInt64(&ii.assetNumber))
}

func (ii *immichIndex) add(a *assets.Asset) *assets.Asset {
	atomic.AddInt64(&ii.assetNumber, 1)
	ii.immichAssets.Store(a.ID, a)
	filename := a.OriginalFileName

	ii.byDeviceID.Store(a.DeviceAssetID(), a.ID)
	l, _ := ii.byName.Load(filename)
	l = append(l, a.ID)
	ii.byName.Store(filename, l)
	return a
}

func (ii *immichIndex) replaceAsset(newA *assets.Asset, oldA *assets.Asset) *assets.Asset {
	ii.lock.Lock()
	defer ii.lock.Unlock()

	ii.byDeviceID.Delete(oldA.DeviceAssetID())         // remove the old AssetID
	ii.immichAssets.Store(newA.ID, newA)               // Store the new asset
	ii.byDeviceID.Store(newA.DeviceAssetID(), newA.ID) // Store the new AssetID
	return newA
}

type Advice struct {
	Advice      AdviceCode
	Message     string
	ServerAsset *assets.Asset
	LocalAsset  *assets.Asset
}

func formatBytes(s int64) string {
	suffixes := []string{"B", "KB", "MB", "GB"}
	bytes := float64(s)
	base := 1024.0
	if bytes < base {
		return fmt.Sprintf("%.0f %s", bytes, suffixes[0])
	}
	exp := int64(0)
	for bytes >= base && exp < int64(len(suffixes)-1) {
		bytes /= base
		exp++
	}
	roundedSize := math.Round(bytes*10) / 10
	return fmt.Sprintf("%.1f %s", roundedSize, suffixes[exp])
}

func (ii *immichIndex) adviceSameOnServer(sa *assets.Asset) *Advice {
	return &Advice{
		Advice:      SameOnServer,
		Message:     fmt.Sprintf("An asset with the same name:%q, date:%q and size:%s exists on the server. No need to upload.", sa.OriginalFileName, sa.CaptureDate.Format(time.DateTime), formatBytes(int64(sa.FileSize))),
		ServerAsset: sa,
	}
}

func (ii *immichIndex) adviceSmallerOnServer(sa *assets.Asset) *Advice {
	return &Advice{
		Advice:      SmallerOnServer,
		Message:     fmt.Sprintf("An asset with the same name:%q and date:%q but with smaller size:%s exists on the server. Replace it.", sa.OriginalFileName, sa.CaptureDate.Format(time.DateTime), formatBytes(int64(sa.FileSize))),
		ServerAsset: sa,
	}
}

func (ii *immichIndex) adviceBetterOnServer(sa *assets.Asset) *Advice {
	return &Advice{
		Advice:      BetterOnServer,
		Message:     fmt.Sprintf("An asset with the same name:%q and date:%q but with bigger size:%s exists on the server. No need to upload.", sa.OriginalFileName, sa.CaptureDate.Format(time.DateTime), formatBytes(int64(sa.FileSize))),
		ServerAsset: sa,
	}
}

func (ii *immichIndex) adviceNotOnServer() *Advice {
	return &Advice{
		Advice:  NotOnServer,
		Message: "This a new asset, upload it.",
	}
}

// ShouldUpload check if the server has this asset
//
// The server may have different assets with the same name. This happens with photos produced by digital cameras.
// The server may have the asset, but in lower resolution. Compare the taken date and resolution
//
// la - local asset
// la.File.Name() is the full path to the file as it is on the source
// la.OriginalFileName is the name of the file as it was on the device before it was uploaded to the server

func (ii *immichIndex) ShouldUpload(la *assets.Asset) (*Advice, error) {
	filename := path.Base(la.File.Name())
	DeviceAssetID := fmt.Sprintf("%s-%d", filename, la.FileSize)

	id, ok := ii.byDeviceID.Load(DeviceAssetID)
	if ok {
		// the same ID exist on the server
		sa, ok := ii.immichAssets.Load(id)
		if ok {
			return ii.adviceSameOnServer(sa), nil
		}
	}

	// check all files with the same name
	ids, ok := ii.byName.Load(filename)

	if ok && len(ids) > 0 {
		dateTaken := la.CaptureDate
		if dateTaken.IsZero() {
			dateTaken = la.FileDate
		}
		size := int64(la.FileSize)

		for _, id := range ids {
			sa, ok := ii.immichAssets.Load(id)
			if !ok {
				continue
			}

			compareDate := compareDate(dateTaken, sa.CaptureDate)
			compareSize := size - int64(sa.FileSize)

			switch {
			case compareDate == 0 && compareSize == 0:
				return ii.adviceSameOnServer(sa), nil
			case compareDate == 0 && compareSize > 0:
				return ii.adviceSmallerOnServer(sa), nil
			case compareDate == 0 && compareSize < 0:
				return ii.adviceBetterOnServer(sa), nil
			}
		}
	}
	return ii.adviceNotOnServer(), nil
}

func compareDate(d1 time.Time, d2 time.Time) int {
	diff := d1.Sub(d2)

	switch {
	case diff < -5*time.Minute:
		return -1
	case diff >= 5*time.Minute:
		return +1
	}
	return 0
}
