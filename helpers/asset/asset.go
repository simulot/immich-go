package asset

import (
	"fmt"
	"math"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/simulot/immich-go/browser"
	"github.com/simulot/immich-go/immich"
)

type AssetIndex struct {
	assets []*immich.Asset
	ByHash map[string][]*immich.Asset
	ByName map[string][]*immich.Asset
	ByID   map[string]*immich.Asset
	// TODO ByTag
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

func (ai *AssetIndex) AddLocalAsset(la *browser.LocalAssetFile, immichID string) {
	sa := &immich.Asset{
		ID:               immichID,
		DeviceAssetID:    la.DeviceAssetID(),
		OriginalFileName: strings.TrimSuffix(path.Base(la.Title), path.Ext(la.Title)),
		ExifInfo: immich.ExifInfo{
			FileSizeInByte:   int(la.Size()),
			DateTimeOriginal: immich.ImmichTime{Time: la.Metadata.DateTaken},
			Latitude:         la.Metadata.Latitude,
			Longitude:        la.Metadata.Longitude,
		},
		JustUploaded: true,
	}
	ai.assets = append(ai.assets, sa)
	ai.ByID[sa.DeviceAssetID] = sa
	l := ai.ByName[sa.OriginalFileName]
	l = append(l, sa)
	ai.ByName[sa.OriginalFileName] = l
}

func (ai *AssetIndex) adviceSmallerOnServer(sa *immich.Asset) *Advice {
	return &Advice{
		Advice: SmallerOnServer,
		Message: fmt.Sprintf(
			"An asset with the same name:%q and date:%q but with smaller size:%s exists on the server. Replace it.",
			sa.OriginalFileName,
			sa.ExifInfo.DateTimeOriginal.Format(time.DateTime),
			formatBytes(sa.ExifInfo.FileSizeInByte),
		),
		ServerAsset: sa,
	}
}

func (ai *AssetIndex) adviceBetterOnServer(sa *immich.Asset) *Advice {
	return &Advice{
		Advice: BetterOnServer,
		Message: fmt.Sprintf(
			"An asset with the same name:%q and date:%q but with bigger size:%s exists on the server. No need to upload.",
			sa.OriginalFileName,
			sa.ExifInfo.DateTimeOriginal.Format(time.DateTime),
			formatBytes(sa.ExifInfo.FileSizeInByte),
		),
		ServerAsset: sa,
	}
}

func (ai *AssetIndex) adviceNotOnServer() *Advice {
	return &Advice{
		Advice:  NotOnServer,
		Message: "This a new asset, upload it.",
	}
}

func (ai *AssetIndex) adviceSameOnServer(sa *immich.Asset) *Advice {
	return &Advice{
		Advice: SameOnServer,
		Message: fmt.Sprintf(
			"An asset with the same name:%q, date:%q and size:%s exists on the server. No need to upload.",
			sa.OriginalFileName,
			sa.ExifInfo.DateTimeOriginal.Format(time.DateTime),
			formatBytes(sa.ExifInfo.FileSizeInByte),
		),
		ServerAsset: sa,
	}
}

// ShouldUpload check if the server has this asset
//
// The server may have different assets with the same name. This happens with photos produced by digital cameras.
// The server may have the asset, but in lower resolution. Compare the taken date and resolution
func (ai *AssetIndex) GetAdvice(la *browser.LocalAssetFile) *Advice {
	filename := la.Title
	if path.Ext(filename) == "" {
		filename += path.Ext(la.FileName)
	}

	ID := la.DeviceAssetID()

	sa := ai.ByID[ID]
	if sa != nil {
		// the same ID exist on the server
		return ai.adviceSameOnServer(sa)
	}

	var l []*immich.Asset

	// check all files with the same name

	n := filepath.Base(filename)
	l = ai.ByName[n]

	if len(l) > 0 {
		dateTaken := la.Metadata.DateTaken
		size := int(la.Size())

		for _, sa = range l {
			compareDate := compareDate(dateTaken, sa.ExifInfo.DateTimeOriginal.Time)
			compareSize := size - sa.ExifInfo.FileSizeInByte

			switch {
			case compareDate == 0 && compareSize == 0:
				return ai.adviceSameOnServer(sa)
			case compareDate == 0 && compareSize > 0:
				return ai.adviceSmallerOnServer(sa)
			case compareDate == 0 && compareSize < 0:
				return ai.adviceBetterOnServer(sa)
			}
		}
	}
	return ai.adviceNotOnServer()
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

func formatBytes(s int) string {
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
