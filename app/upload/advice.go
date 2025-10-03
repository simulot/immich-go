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
	case AlreadyProcessed:
		return "AlreadyProcessed"
	case ForceUpload:
		return "ForceUpload"
	}
	return fmt.Sprintf("advice(%d)", a)
}

const (
	IDontKnow AdviceCode = iota
	SmallerOnServer
	BetterOnServer
	SameOnServer
	NotOnServer
	AlreadyProcessed
	ForceUpload
)

type immichIndex struct {
	lock sync.Mutex

	// map of assetID to asset, local and server ones
	immichAssets *syncmap.SyncMap[string, *assets.Asset]

	// set of Uploaded Checksums
	uploadsChecksum *syncset.Set[string]

	// map of base name to assetID
	byName *syncmap.SyncMap[string, []string]

	// map of SHA1 to assetID
	byChecksum *syncmap.SyncMap[string, *assets.Asset]

	assetNumber int64
}

func newAssetIndex() *immichIndex {
	return &immichIndex{
		immichAssets:    syncmap.New[string, *assets.Asset](),
		byChecksum:      syncmap.New[string, *assets.Asset](),
		byName:          syncmap.New[string, []string](),
		uploadsChecksum: syncset.New[string](),
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
	return ii.add(a, false), true
}

func (ii *immichIndex) addLocalAsset(ia *assets.Asset) (*assets.Asset, bool) {
	ii.lock.Lock()
	defer ii.lock.Unlock()

	if existing, ok := ii.immichAssets.Load(ia.ID); ok {
		return existing, false
	}
	if existing, ok := ii.byChecksum.Load(ia.Checksum); ok {
		return existing, false
	}
	return ii.add(ia, true), true
}

func (ii *immichIndex) getByID(id string) *assets.Asset {
	a, _ := ii.immichAssets.Load(id)
	return a
}

func (ii *immichIndex) len() int {
	return int(atomic.LoadInt64(&ii.assetNumber))
}

func (ii *immichIndex) add(a *assets.Asset, local bool) *assets.Asset {
	if a.ID == "" {
		panic("asset ID is empty")
	}
	if a.Checksum == "" {
		panic("asset checksum is empty")
	}

	if _, ok := ii.byChecksum.Load(a.Checksum); ok {
		panic("asset checksum already exists")
	}

	if ii.uploadsChecksum.Contains(a.Checksum) {
		panic("asset checksum already exists in uploads")
	}

	atomic.AddInt64(&ii.assetNumber, 1)
	ii.immichAssets.Store(a.ID, a)
	ii.byChecksum.Store(a.Checksum, a)
	filename := a.OriginalFileName

	if local {
		ii.uploadsChecksum.Add(a.Checksum)
	}

	l, _ := ii.byName.Load(filename)
	l = append(l, a.ID)
	ii.byName.Store(filename, l)
	return a
}

func (ii *immichIndex) replaceAsset(newA *assets.Asset, oldA *assets.Asset) *assets.Asset {
	if newA.ID == "" {
		panic("asset ID is empty")
	}
	if newA.Checksum == "" {
		panic("asset checksum is empty")
	}
	ii.lock.Lock()
	defer ii.lock.Unlock()
	oldA.Trashed = true
	ii.immichAssets.Store(newA.ID, newA)     // Store the new asset
	ii.byChecksum.Store(newA.Checksum, newA) // Store the new SHA1
	ii.uploadsChecksum.Add(newA.Checksum)

	filename := newA.OriginalFileName
	l, _ := ii.byName.Load(filename)
	l = append(l, newA.ID)
	ii.byName.Store(filename, l)
	return newA
}

func (ii *immichIndex) isAlreadyProcessed(checksum string) bool {
	return ii.uploadsChecksum.Contains(checksum)
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

func (ii *immichIndex) adviceAlreadyProcessed(sa *assets.Asset) *Advice {
	return &Advice{
		Advice:      AlreadyProcessed,
		Message:     fmt.Sprintf("An asset with the same checksum:%q has been already processed. No need to upload.", sa.Checksum),
		ServerAsset: sa,
	}
}

func (ii *immichIndex) adviceNotOnServer() *Advice {
	return &Advice{
		Advice:  NotOnServer,
		Message: "This a new asset, upload it.",
	}
}

func (ii *immichIndex) adviceForceUpload(sa *assets.Asset) *Advice {
	return &Advice{
		Advice:      ForceUpload,
		Message:     "This asset is marked for force upload.",
		ServerAsset: sa,
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

func (ii *immichIndex) ShouldUpload(la *assets.Asset, upCmd *UpCmd) (*Advice, error) {
	checksum, err := la.GetChecksum()
	if err != nil {
		return nil, err
	}

	if sa, ok := ii.byChecksum.Load(checksum); ok {
		if ii.isAlreadyProcessed(checksum) {
			return ii.adviceAlreadyProcessed(sa), nil
		}
		return ii.adviceSameOnServer(sa), nil
	}

	filename := path.Base(la.File.Name())

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
			case compareDate == 0 && upCmd.Overwrite:
				return ii.adviceForceUpload(sa), nil
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
	case diff < -5*time.Second:
		return -1
	case diff >= 5*time.Second:
		return +1
	}
	return 0
}
