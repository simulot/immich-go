package upload

import (
	"fmt"
	"math"
	"path"
	"regexp"
	"sort"
	"strings"
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

	// map of conservative duplicate keys to assets
	byConservative *syncmap.SyncMap[string, []*assets.Asset]

	// map of day+type to assets (used for Snapchat fuzzy duplicate matching)
	byDayType *syncmap.SyncMap[string, []*assets.Asset]

	assetNumber int64
}

func newAssetIndex() *immichIndex {
	return &immichIndex{
		immichAssets:    syncmap.New[string, *assets.Asset](),
		byChecksum:      syncmap.New[string, *assets.Asset](),
		byConservative:  syncmap.New[string, []*assets.Asset](),
		byDayType:       syncmap.New[string, []*assets.Asset](),
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

	if key := conservativeKey(a); key != "" {
		l2, _ := ii.byConservative.Load(key)
		l2 = append(l2, a)
		ii.byConservative.Store(key, l2)
	}

	if key := dayTypeKey(a); key != "" {
		l3, _ := ii.byDayType.Load(key)
		l3 = append(l3, a)
		ii.byDayType.Store(key, l3)
	}
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

	if upCmd.ConservativeDuplicates {
		for _, key := range conservativeCandidateKeys(la) {
			candidates, ok := ii.byConservative.Load(key)
			if !ok {
				continue
			}
			for _, sa := range candidates {
				if sa == nil {
					continue
				}
				if upCmd.app != nil {
					upCmd.app.Log().Debug("conservative duplicate matched", "local", la.OriginalFileName, "server", sa.OriginalFileName, "key", key)
				}
				return ii.adviceSameOnServer(sa), nil
			}
		}

		if best := ii.findSnapchatFuzzyDuplicate(la); best != nil {
			if upCmd.app != nil {
				upCmd.app.Log().Debug("snapchat fuzzy duplicate matched", "local", la.OriginalFileName, "server", best.OriginalFileName)
			}
			return ii.adviceSameOnServer(best), nil
		}
	}
	return ii.adviceNotOnServer(), nil
}

var snapMainNameRE = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}_.+-main\.[A-Za-z0-9]+$`)

func isSnapchatMainName(name string) bool {
	return snapMainNameRE.MatchString(name)
}

func isSnapchatImmichName(name string) bool {
	return strings.HasPrefix(strings.ToLower(name), "snapchat-")
}

func normalizeAssetType(a *assets.Asset) string {
	if a == nil {
		return ""
	}
	t := strings.ToLower(strings.TrimSpace(a.Type))
	if t != "" {
		return t
	}
	ext := strings.ToLower(path.Ext(a.OriginalFileName))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".heic", ".heif", ".webp", ".gif":
		return "image"
	case ".mp4", ".mov", ".m4v", ".webm":
		return "video"
	default:
		return ""
	}
}

func dayTypeKey(a *assets.Asset) string {
	if a == nil {
		return ""
	}
	t := a.CaptureDate
	if t.IsZero() {
		t = a.FileDate
	}
	if t.IsZero() {
		return ""
	}
	k := normalizeAssetType(a)
	if k == "" {
		return ""
	}
	day := t.UTC().Format("2006-01-02")
	return k + "|" + day
}

func (ii *immichIndex) findSnapchatFuzzyDuplicate(la *assets.Asset) *assets.Asset {
	if la == nil || !isSnapchatMainName(la.OriginalFileName) {
		return nil
	}
	keys := dayTypeWindowKeys(la, 1)
	if len(keys) == 0 {
		return nil
	}

	candidateByID := map[string]*assets.Asset{}
	for _, key := range keys {
		candidates, ok := ii.byDayType.Load(key)
		if !ok || len(candidates) == 0 {
			continue
		}
		for _, sa := range candidates {
			if sa == nil || sa.ID == "" {
				continue
			}
			candidateByID[sa.ID] = sa
		}
	}
	if len(candidateByID) == 0 {
		return nil
	}

	localSize := int64(la.FileSize)
	if localSize <= 0 {
		return nil
	}
	localType := normalizeAssetType(la)
	localDay := normalizeToDay(la.CaptureDate)
	if localDay.IsZero() {
		localDay = normalizeToDay(la.FileDate)
	}

	bestScore := 2.0
	secondScore := 2.0
	var best *assets.Asset
	typeBaseThreshold := 0.33 // images
	if localType == "video" {
		typeBaseThreshold = 0.90
	}

	candidates := make([]*assets.Asset, 0, len(candidateByID))
	for _, sa := range candidateByID {
		candidates = append(candidates, sa)
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].ID < candidates[j].ID })

	for _, sa := range candidates {
		if sa == nil || sa.Trashed || !isSnapchatImmichName(sa.OriginalFileName) {
			continue
		}
		if normalizeAssetType(sa) != localType {
			continue
		}

		serverDay := normalizeToDay(sa.CaptureDate)
		if serverDay.IsZero() {
			serverDay = normalizeToDay(sa.FileDate)
		}
		if serverDay.IsZero() || localDay.IsZero() {
			continue
		}
		dayDiff := absDaysBetween(localDay, serverDay)
		if dayDiff > 1 {
			continue
		}

		serverSize := int64(sa.FileSize)
		if serverSize <= 0 {
			continue
		}
		sizeScore := float64(absInt64(localSize-serverSize)) / float64(maxInt64(localSize, serverSize))
		score := sizeScore + float64(dayDiff)*0.08

		threshold := typeBaseThreshold
		if dayDiff == 1 {
			threshold -= 0.08
		}
		if sizeScore > threshold {
			continue
		}

		if score < bestScore {
			secondScore = bestScore
			bestScore = score
			best = sa
		} else if score < secondScore {
			secondScore = score
		}
	}

	if best == nil {
		return nil
	}

	// Avoid ambiguous matches among many same-day assets.
	if secondScore-bestScore < 0.05 {
		return nil
	}

	return best
}

func dayTypeWindowKeys(a *assets.Asset, dayWindow int) []string {
	if a == nil {
		return nil
	}
	t := normalizeToDay(a.CaptureDate)
	if t.IsZero() {
		t = normalizeToDay(a.FileDate)
	}
	if t.IsZero() {
		return nil
	}
	typeName := normalizeAssetType(a)
	if typeName == "" {
		return nil
	}
	keys := make([]string, 0, dayWindow*2+1)
	for delta := -dayWindow; delta <= dayWindow; delta++ {
		day := t.AddDate(0, 0, delta).Format("2006-01-02")
		keys = append(keys, typeName+"|"+day)
	}
	return keys
}

func normalizeToDay(t time.Time) time.Time {
	if t.IsZero() {
		return time.Time{}
	}
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), u.Day(), 0, 0, 0, 0, time.UTC)
}

func absDaysBetween(a, b time.Time) int {
	if a.IsZero() || b.IsZero() {
		return 0
	}
	d := a.Sub(b)
	if d < 0 {
		d = -d
	}
	return int(d / (24 * time.Hour))
}

func absInt64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

func maxInt64(a int64, b int64) int64 {
	if a > b {
		return a
	}
	return b
}

func conservativeCandidateKeys(a *assets.Asset) []string {
	if conservativeKey(a) == "" {
		return nil
	}
	dateTaken := a.CaptureDate
	if dateTaken.IsZero() {
		dateTaken = a.FileDate
	}
	t := dateTaken.UTC()
	size := a.FileSize
	typeName := normalizeAssetType(a)
	if typeName == "" {
		typeName = "unknown"
	}
	keys := make([]string, 0, 11)
	for delta := -5; delta <= 5; delta++ {
		keys = append(keys, fmt.Sprintf("%s|%d|%d", typeName, size, t.Add(time.Duration(delta)*time.Second).Unix()))
	}
	return keys
}

func conservativeKey(a *assets.Asset) string {
	if a == nil {
		return ""
	}
	dateTaken := a.CaptureDate
	if dateTaken.IsZero() {
		dateTaken = a.FileDate
	}
	if dateTaken.IsZero() {
		return ""
	}
	typeName := normalizeAssetType(a)
	if typeName == "" {
		typeName = "unknown"
	}
	return fmt.Sprintf("%s|%d|%d", typeName, a.FileSize, dateTaken.UTC().Unix())
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
