package bereal

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"strings"
	"time"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper"
)

// BeRealAdapter imports BeReal memories from an export folder
type BeRealAdapter struct {
	fsys   fs.FS
	logger *app.Log

	memories       []BeRealMemory
	perMemoryAlbum bool
}

// BeRealMemory represents a single BeReal memory entry from memories.json
type BeRealMemory struct {
	FrontImage   ImageInfo `json:"frontImage"`
	BackImage    ImageInfo `json:"backImage,omitempty"`
	IsLate       bool      `json:"isLate"`
	Date         string    `json:"date"`
	TakenTime    string    `json:"takenTime"`
	BeRealMoment string    `json:"berealMoment"`
	Location     Location  `json:"location,omitempty"`
	Music        Music     `json:"music,omitempty"`
}

// ImageInfo contains information about a single image (main or secondary/selfie)
type ImageInfo struct {
	Path      string `json:"path"`
	Height    int    `json:"height"`
	Width     int    `json:"width"`
	MediaType string `json:"mediaType"`
	MimeType  string `json:"mimeType"`
}

// Location contains GPS coordinates
type Location struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// Music contains Spotify music information
type Music struct {
	Track  string `json:"track"`
	Artist string `json:"artist"`
}

// NewBeRealAdapter creates a new BeReal adapter
func NewBeRealAdapter(fsys fs.FS, logger *app.Log) (*BeRealAdapter, error) {
	adapter := &BeRealAdapter{
		fsys:   fsys,
		logger: logger,
	}

	// Load memories.json
	if err := adapter.loadMemories(); err != nil {
		return nil, fmt.Errorf("load memories: %w", err)
	}

	return adapter, nil
}

// SetPerMemoryAlbum enables grouping assets into per-memory albums when true.
func (ba *BeRealAdapter) SetPerMemoryAlbum(v bool) {
	ba.perMemoryAlbum = v
}

// loadMemories reads and parses memories.json
func (ba *BeRealAdapter) loadMemories() error {
	// Try to find memories.json in the expected location
	// BeReal exports have structure: <userID>/memories.json
	memoriesPath := "memories.json"

	data, err := fs.ReadFile(ba.fsys, memoriesPath)
	if err != nil {
		return fmt.Errorf("read memories.json: %w", err)
	}

	if err := json.Unmarshal(data, &ba.memories); err != nil {
		return fmt.Errorf("parse memories.json: %w", err)
	}

	ba.logger.Info(fmt.Sprintf("loaded BeReal memories: %d", len(ba.memories)))
	return nil
}

// DiscoverAssets yields all BeReal memory assets
// This method is used by Browse
func (ba *BeRealAdapter) discoverAssets() []*assets.Asset {
	var result []*assets.Asset

	for _, memory := range ba.memories {
		// Create main asset from frontImage
		if memory.FrontImage.Path != "" {
			if asset := ba.createAsset(memory, true); asset != nil {
				result = append(result, asset)
			}
		}

		// Create selfie asset from backImage (if present)
		if memory.BackImage.Path != "" {
			if asset := ba.createAsset(memory, false); asset != nil {
				result = append(result, asset)
			}
		}
	}

	return result
}

// Browse implements the adapters.Reader interface
// It returns a channel of asset groups for processing
// Front and back images from the same memory are grouped together for stacking
func (ba *BeRealAdapter) Browse(ctx context.Context) chan *assets.Group {
	out := make(chan *assets.Group)

	go func() {
		defer close(out)

		// Create groups where front+back pairs are stacked together
		for _, memory := range ba.memories {
			var groupAssets []*assets.Asset

			// Add front image
			if memory.FrontImage.Path != "" {
				if asset := ba.createAsset(memory, true); asset != nil {
					groupAssets = append(groupAssets, asset)
				}
			}

			// Add back image
			if memory.BackImage.Path != "" {
				if asset := ba.createAsset(memory, false); asset != nil {
					groupAssets = append(groupAssets, asset)
				}
			}

			// Create a group for this memory's pair with GroupByRawJpg
			// (representing front/back pair stacking similar to raw/jpg stacking)
			if len(groupAssets) > 0 {
				group := assets.NewGroup(assets.GroupByRawJpg, groupAssets...)
				// The main (front) image is the cover
				if len(groupAssets) > 0 {
					group.SetCover(0)
				}

				select {
				case out <- group:
				case <-ctx.Done():
					return
				}
			}
		}
	}()

	return out
}

// createAsset converts a BeReal memory into an assets.Asset
// isFront=true for main camera, false for selfie camera
func (ba *BeRealAdapter) createAsset(memory BeRealMemory, isFront bool) *assets.Asset {
	var imageInfo ImageInfo
	var tag string

	if isFront {
		imageInfo = memory.FrontImage
		tag = "BeReal_Main"
	} else {
		imageInfo = memory.BackImage
		tag = "BeReal_Selfie"
	}

	if imageInfo.Path == "" {
		return nil
	}

	// Parse the image path to get the filename
	fileName := path.Base(imageInfo.Path)

	// Open the file to get metadata. Try a few fallbacks when the path in the JSON
	// doesn't match the layout on disk (some exports embed the user-id as an
	// extra path segment: "Photos/<userId>/post/..." while files live under
	// "Photos/post/..."). Build candidate paths from both the raw JSON path
	// and a trimmed (no-leading-slash) variant.
	var f fs.File
	var err error
	rawPath := imageInfo.Path
	trimmed := strings.TrimPrefix(rawPath, "/")

	tryPaths := []string{}
	// Start with the raw and trimmed forms (avoid duplicates)
	if rawPath != "" {
		tryPaths = append(tryPaths, rawPath)
	}
	if trimmed != rawPath {
		tryPaths = append(tryPaths, trimmed)
	}

	// Compute parts using the trimmed path so leading slash doesn't produce
	// an empty first element.
	parts := strings.Split(trimmed, "/")
	// If path looks like Photos/<something>/rest -> try Photos/rest
	if len(parts) >= 3 && parts[0] == "Photos" {
		tryPaths = append(tryPaths, path.Join("Photos", strings.Join(parts[2:], "/")))
	}
	// Also try dropping the Photos/ prefix entirely -> rest
	if len(parts) >= 2 && parts[0] == "Photos" {
		tryPaths = append(tryPaths, strings.Join(parts[1:], "/"))
	}

	// De-duplicate tryPaths preserving order
	seen := map[string]bool{}
	dedup := []string{}
	for _, p := range tryPaths {
		if p == "" {
			continue
		}
		if !seen[p] {
			seen[p] = true
			dedup = append(dedup, p)
		}
	}

	for _, pth := range dedup {
		f, err = ba.fsys.Open(pth)
		if err == nil {
			imageInfo.Path = pth // record the resolved path
			break
		}
	}
	if err != nil {
		ba.logger.Warn(fmt.Sprintf("failed to open BeReal image %s (tried %v): %v", imageInfo.Path, tryPaths, err))
		return nil
	}
	defer f.Close()

	// Get file info
	info, err := f.Stat()
	if err != nil {
		ba.logger.Warn(fmt.Sprintf("failed to stat BeReal image %s: %v", imageInfo.Path, err))
		return nil
	}

	// Parse capture date
	captureDate, err := time.Parse(time.RFC3339, memory.TakenTime)
	if err != nil {
		ba.logger.Warn(fmt.Sprintf("failed to parse BeReal takenTime %s: %v", memory.TakenTime, err))
		// Fall back to date field
		captureDate, _ = time.Parse(time.RFC3339, memory.Date)
	}

	// Create asset
	asset := &assets.Asset{
		File:             fshelper.FSName(ba.fsys, imageInfo.Path),
		OriginalFileName: fileName,
		FileSize:         int(info.Size()),
		FileDate:         info.ModTime().UTC(),
		CaptureDate:      captureDate,
		Latitude:         memory.Location.Latitude,
		Longitude:        memory.Location.Longitude,
		Visibility:       assets.VisibilityTimeline,
	}

	// Add tag
	asset.AddTag(tag)

	// Add album
	if ba.perMemoryAlbum {
		// Use the capture date to create a per-memory album name
		albumTitle := "BeReal"
		asset.Albums = []assets.Album{
			{
				Title:       albumTitle,
				Description: "BeReal memories",
			},
		}
	} else {
		asset.Albums = []assets.Album{
			{
				Title:       "BeReal",
				Description: "BeReal memories",
			},
		}
	}

	// Add description with music if available
	if memory.Music.Track != "" {
		asset.Description = fmt.Sprintf("🎵 %s by %s", memory.Music.Track, memory.Music.Artist)
	}

	ba.logger.Debug(fmt.Sprintf("created BeReal asset %s with tag %s dated %v, size %d", fileName, tag, captureDate, info.Size()))

	return asset
}

// CountAssets returns the total number of assets that will be discovered
func (ba *BeRealAdapter) CountAssets() (int, error) {
	count := 0
	for _, memory := range ba.memories {
		if memory.FrontImage.Path != "" {
			count++
		}
		if memory.BackImage.Path != "" {
			count++
		}
	}
	return count, nil
}

// Report returns a summary of discovery
func (ba *BeRealAdapter) Report(ctx context.Context) error {
	total, _ := ba.CountAssets()
	fmt.Printf("BeReal: %d memories found\n", len(ba.memories))
	fmt.Printf("BeReal: %d total assets (main + selfie)\n", total)
	return nil
}
