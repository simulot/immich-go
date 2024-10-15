package motionpictures

import (
	"context"
	"path"
	"strings"
	"time"

	"github.com/simulot/immich-go/internal/groups"
	"github.com/simulot/immich-go/internal/metadata"
)

// roundDuration is used to group assets
const roundDuration = 2 * time.Second

// key is used to group assets
type key struct {
	base string
	date time.Time
}

// MotionPictureGrouper is a grouper that groups motion pictures
type MotionPictureGrouper struct {
	supported metadata.SupportedMedia
	unMatched map[key]groups.Asset
	groupChan chan *groups.AssetGroup
}

// New creates a new MotionPictureGrouper
func New(supported metadata.SupportedMedia) *MotionPictureGrouper {
	return &MotionPictureGrouper{
		supported: supported,
		unMatched: make(map[key]groups.Asset),
		groupChan: make(chan *groups.AssetGroup),
	}
}

// Run starts the grouper.
func (m *MotionPictureGrouper) Run(ctx context.Context, in chan groups.Asset) chan *groups.AssetGroup {
	go func() {
		defer func() {
			// Send remaining unmatched assets as groups of one
			for _, a := range m.unMatched {
				select {
				case m.groupChan <- groups.NewAssetGroup(groups.KindNone, a):
				case <-ctx.Done():
					break
				}
			}
			close(m.groupChan)
		}()

		for a := range in {
			select {
			case <-ctx.Done():
				break
			default:
				base := strings.TrimSuffix(a.Base(), path.Ext(a.Base()))
				if a.Type() == metadata.TypeImage {
					ext := path.Ext(base)
					if len(ext) > 0 && strings.HasPrefix(strings.ToLower(ext), ".mp") {
						// Remove MP extension
						base = strings.TrimSuffix(base, ext)
					}
				}
				k := key{base: base, date: a.DateTaken().Round(roundDuration)}
				if other := m.unMatched[k]; other != nil {
					// same base name and date
					if a.Type() != other.Type() {
						// and different types, that's a motion picture
						m.groupChan <- groups.NewAssetGroup(groups.KindMotionPhoto, a, other)
					} else {
						// same base, same type, same rounded date... just 2 different assets
						m.groupChan <- groups.NewAssetGroup(groups.KindNone, a)
						m.groupChan <- groups.NewAssetGroup(groups.KindNone, other)
						m.unMatched[k] = a
					}
					delete(m.unMatched, k)
				} else {
					// No match, store for later
					m.unMatched[k] = a
				}
			}
		}
	}()

	return m.groupChan
}
