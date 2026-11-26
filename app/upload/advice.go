package upload

import (
	"fmt"
	"math"
	"path"
	"slices"
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
	case NamesakeOnServer:
		return "NamesakeOnServer"
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
	NamesakeOnServer
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

	// map of the ID of a replaced, deleted asset to the asset that replaced it
	replacedBy *syncmap.SyncMap[string, *assets.Asset]
	// server asset ID -> primary asset ID of the stack it belongs to ("" = none/unknown)
	stackPrimaryByID *syncmap.SyncMap[string, string]
	// server asset IDs that are versions (stacked or server-edited): never replace/skip
	serverVersions *syncset.Set[string]

	assetNumber int64
}

func newAssetIndex() *immichIndex {
	return &immichIndex{
		immichAssets:     syncmap.New[string, *assets.Asset](),
		byChecksum:       syncmap.New[string, *assets.Asset](),
		byName:           syncmap.New[string, []string](),
		uploadsChecksum:  syncset.New[string](),
		replacedBy:       syncmap.New[string, *assets.Asset](),
		stackPrimaryByID: syncmap.New[string, string](),
		serverVersions:   syncset.New[string](),
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

func (ii *immichIndex) addLocalAsset(ia *assets.Asset) bool {
	ii.lock.Lock()
	defer ii.lock.Unlock()

	if _, ok := ii.immichAssets.Load(ia.ID); ok {
		return false
	}
	if _, ok := ii.byChecksum.Load(ia.Checksum); ok {
		return false
	}
	ii.add(ia, true)
	return true
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

func (ii *immichIndex) replaceAsset(newA *assets.Asset, oldA *assets.Asset) {
	if newA.ID == "" {
		panic("asset ID is empty")
	}
	if newA.Checksum == "" {
		panic("asset checksum is empty")
	}
	ii.lock.Lock()
	defer ii.lock.Unlock()
	oldA.Trashed = true
	ii.replacedBy.Store(oldA.ID, newA)
	ii.immichAssets.Store(newA.ID, newA)     // Store the new asset
	ii.byChecksum.Store(newA.Checksum, newA) // Store the new SHA1
	ii.uploadsChecksum.Add(newA.Checksum)

	filename := newA.OriginalFileName
	l, _ := ii.byName.Load(filename)
	l = append(l, newA.ID)
	ii.byName.Store(filename, l)
}

// replacement returns the asset that replaced sa, following replacements of replacements, or sa
// itself when it has not been replaced. A replaced asset has been deleted from the server, so its
// ID must not be used any more; callers use the replacement in its place.
func (ii *immichIndex) replacement(sa *assets.Asset) *assets.Asset {
	for {
		r, ok := ii.replacedBy.Load(sa.ID)
		if !ok {
			return sa
		}
		sa = r
	}
}

// nameDateCandidates returns the indexed assets whose OriginalFileName equals name and
// whose capture date matches a's, excluding a itself.
func (ii *immichIndex) nameDateCandidates(a *assets.Asset, name string) []*assets.Asset {
	if name == "" {
		return nil
	}
	ids, ok := ii.byName.Load(name)
	if !ok {
		return nil
	}
	var out []*assets.Asset
	for _, id := range ids {
		sa, ok := ii.immichAssets.Load(id)
		if !ok {
			continue
		}
		sa = ii.replacement(sa)
		if sa == nil || sa.ID == "" || sa.ID == a.ID {
			continue
		}
		if compareDate(a.CaptureDate, sa.CaptureDate) != 0 {
			continue
		}
		if !slices.Contains(out, sa) {
			out = append(out, sa)
		}
	}
	return out
}

// editBaseCandidates returns the assets that can be the original of an edited asset:
// same sidecar title and capture date.
func (ii *immichIndex) editBaseCandidates(a *assets.Asset) []*assets.Asset {
	if a.Kind != assets.KindEdited {
		return nil
	}
	return ii.nameDateCandidates(a, a.OriginalFileName)
}

// recordVersion classifies a server asset. A stacked asset, or an asset edited on the
// server, is a version: it is never replaced by the size heuristic, and its stack primary
// is kept so CreateStack merges the existing stack instead of splitting it.
func (ii *immichIndex) recordVersion(info *immich.Asset) {
	if info.Stack != nil && info.Stack.PrimaryAssetID != "" {
		ii.serverVersions.Add(info.ID)
		ii.stackPrimaryByID.Store(info.ID, info.Stack.PrimaryAssetID)
		return
	}
	if info.IsEdited {
		ii.serverVersions.Add(info.ID)
	}
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

func (ii *immichIndex) adviceNamesakeOnServer(sa *assets.Asset) *Advice {
	return &Advice{
		Advice:      NamesakeOnServer,
		Message:     fmt.Sprintf("An asset with the same name:%q and date:%q is on the server or was uploaded in this run, but the input holds several files of this name: upload this one, keep the other.", sa.OriginalFileName, sa.CaptureDate.Format(time.DateTime)),
		ServerAsset: sa,
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
//
// siblings are the assets of la's group (burst, raw+jpg, edited pair...), la itself possibly among
// them. They are never taken for a server-side variant of la: a Google-edited "X-edited.jpg" is
// indexed under the title of its sidecar, "X.jpg", with the same capture date as "X.jpg", and would
// otherwise be reported as a smaller/bigger version of the original when it is uploaded first
// (issue #877, #1285).
//
// An asset marked HasNamesake is one of several different files of one name in the input (two Takeout
// series, #1473). Name and date cannot tell which server asset is a version of it, so only checksums
// compare it: a marked local asset never replaces, nor is skipped for, an asset of its name and date,
// and a marked asset uploaded earlier in the run is never replaced by one.
func (ii *immichIndex) ShouldUpload(la *assets.Asset, upCmd *UpCmd, siblings ...*assets.Asset) (*Advice, error) {
	checksum, err := la.GetChecksum()
	if err != nil {
		return nil, err
	}

	if sa, ok := ii.byChecksum.Load(checksum); ok {
		if r := ii.replacement(sa); r != sa {
			// same content as an asset that has been replaced by another one: the replacement
			// stands for it
			return ii.adviceBetterOnServer(r), nil
		}
		if ii.isAlreadyProcessed(checksum) {
			return ii.adviceAlreadyProcessed(sa), nil
		}
		return ii.adviceSameOnServer(sa), nil
	}

	filename := path.Base(la.File.Name())

	// Server assets that hold the bytes of one of the group's edited versions are not
	// variants of this file: the versions are stacked, not compared by size. Without
	// this, importing an edit pair over a server copy of the edit would replace or skip
	// the original.
	var editedChecksums []string
	for _, s := range siblings {
		if s == nil || s.Kind != assets.KindEdited {
			continue
		}
		cs := s.Checksum
		if cs == "" {
			var err error
			cs, err = s.GetChecksum()
			if err != nil {
				continue
			}
		}
		editedChecksums = append(editedChecksums, cs)
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
			sa = ii.replacement(sa)
			if sa == la || slices.Contains(siblings, sa) {
				continue
			}
			if slices.Contains(editedChecksums, sa.Checksum) {
				continue
			}
			if ii.serverVersions.Contains(sa.ID) {
				continue
			}

			compareDate := compareDate(dateTaken, sa.CaptureDate)
			compareSize := size - int64(sa.FileSize)

			switch {
			case compareDate == 0 && (la.HasNamesake || sa.HasNamesake):
				return ii.adviceNamesakeOnServer(sa), nil
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
