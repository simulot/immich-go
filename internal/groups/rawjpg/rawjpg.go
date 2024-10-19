package rawjpg

import (
	"context"
	"time"

	"github.com/simulot/immich-go/internal/groups"
	"github.com/simulot/immich-go/internal/metadata"
	"golang.org/x/exp/constraints"
)

/*
This package implements a group builder for RAW and JPG images.

*/

type RawJpgMode int

const (
	ModeNone RawJpgMode = iota
	ModeRawCover
	ModeJpgCover
	ModeKeepRaw
	ModeKeepJpg
)
const acceptedTimeDiff = 5 * time.Second

type RawJpgGrouper struct {
	Mode RawJpgMode
}

func (gr *RawJpgGrouper) Group(ctx context.Context, in chan groups.Asset) (outg chan *groups.AssetGroup, outa chan groups.Asset) {
	outg = make(chan *groups.AssetGroup)
	outa = make(chan groups.Asset)

	go func() {
		defer close(outa)
		defer close(outg)
		var lastAsset groups.Asset
		for {
			select {
			case a, ok := <-in:
				if !ok {
					if lastAsset != nil {
						outa <- lastAsset
					}
					return
				}
				if a == nil {
					continue
				}

				// If the grouper is disabled, we send the asset to the output channel
				if gr.Mode == ModeNone {
					outa <- a
					continue
				}

				// If the asset is not an image, we send it to the output channel
				if a.Type() != metadata.TypeImage {
					// We send the last asset if any
					if lastAsset != nil {
						outa <- lastAsset
						lastAsset = nil
					}
					outa <- a
					continue
				}

				// Keep the current asset for the next iteration
				if lastAsset == nil {
					lastAsset = a
					continue
				}

				// if the last asset doesn't match the current asset, we send the last asset to the output channel
				// and keep the current asset for the next iteration
				if lastAsset.Radical() != a.Radical() {
					outa <- lastAsset
					lastAsset = a
					continue
				}

				// We have two assets of the same type with the same radical
				if metadata.IsRawFile(lastAsset.Ext()) == metadata.IsRawFile(a.Ext()) {
					outa <- lastAsset
					lastAsset = a
					continue
				}

				// We have two assets with the same radical, one is a raw file, the other is a jpg file,
				// but the time difference is too big
				if abs(lastAsset.DateTaken().Sub(a.DateTaken())) > acceptedTimeDiff {
					outa <- lastAsset
					lastAsset = a
					continue
				}
				// We have two assets with the same radical, one is a raw file, the other is a jpg file
				// with a time difference less than acceptedTimeDiff
				switch gr.Mode {
				case ModeKeepRaw:
					if metadata.IsRawFile(lastAsset.Ext()) {
						outa <- lastAsset
					} else {
						outa <- a
					}
					lastAsset = nil
				case ModeKeepJpg:
					if metadata.IsRawFile(lastAsset.Ext()) {
						outa <- a
					} else {
						outa <- lastAsset
					}
					lastAsset = nil
				case ModeRawCover:
					g := groups.NewAssetGroup(groups.KindRawJpg, lastAsset, a)
					if metadata.IsRawFile(lastAsset.Ext()) {
						g.CoverIndex = 0
					} else {
						g.CoverIndex = 1
					}
					outg <- g
					lastAsset = nil
				case ModeJpgCover:
					g := groups.NewAssetGroup(groups.KindRawJpg, lastAsset, a)
					if metadata.IsRawFile(lastAsset.Ext()) {
						g.CoverIndex = 1
					} else {
						g.CoverIndex = 0
					}
					outg <- g
					lastAsset = nil
				}

			case <-ctx.Done():
				return
			}
		}
	}()
	return outg, outa
}

func abs[T constraints.Integer](a T) T {
	if a < 0 {
		return -a
	}
	return a
}
