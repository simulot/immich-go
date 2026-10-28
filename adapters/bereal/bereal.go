package bereal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"path"
	"strings"
	"time"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper"
)

const (
	albumTitle       = "BeReal"
	albumDescription = "BeReal memories"

	tagMain   = "BeReal_Main"   // back camera
	tagSelfie = "BeReal_Selfie" // front camera
	tagBTS    = "BeReal_BTS"    // behind the scenes video
)

// BeRealAdapter imports BeReal memories from an export folder
type BeRealAdapter struct {
	fsys   fs.FS
	logger *app.Log

	memories []BeRealMemory
}

// BeRealMemory represents a single BeReal memory entry from memories.json
type BeRealMemory struct {
	FrontImage   ImageInfo `json:"frontImage"`
	BackImage    ImageInfo `json:"backImage,omitempty"`
	BtsMedia     ImageInfo `json:"btsMedia,omitempty"`
	Caption      string    `json:"caption"`
	IsLate       bool      `json:"isLate"`
	Date         string    `json:"date"`
	TakenTime    string    `json:"takenTime"`
	BeRealMoment string    `json:"berealMoment"`
	Location     Location  `json:"location,omitempty"`
	Music        Music     `json:"music,omitempty"`
}

// ImageInfo contains information about a single image or video (main, secondary/selfie, or behind the scenes)
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

// Browse implements the adapters.Reader interface
// It returns a channel of asset groups for processing
// Back, front and behind the scenes media from the same memory are grouped together for stacking
func (ba *BeRealAdapter) Browse(ctx context.Context) chan *assets.Group {
	out := make(chan *assets.Group)

	go func() {
		defer close(out)

		// Create groups where the main, selfie and behind the scenes media are stacked together
		for _, memory := range ba.memories {
			var groupAssets []*assets.Asset
			var mainAsset, selfieAsset *assets.Asset

			// Add Main image (back camera) - this is the cover, unless the selfie video is the only one with sound
			if memory.BackImage.Path != "" {
				mainAsset = ba.createAsset(memory, memory.BackImage, tagMain)
				if mainAsset != nil {
					groupAssets = append(groupAssets, mainAsset)
				}
			}

			// Add Selfie image (front camera) - this will be stacked
			if memory.FrontImage.Path != "" {
				selfieAsset = ba.createAsset(memory, memory.FrontImage, tagSelfie)
				if selfieAsset != nil {
					groupAssets = append(groupAssets, selfieAsset)
				}
			}

			// Add the behind the scenes video - this will be stacked
			if memory.BtsMedia.Path != "" {
				if asset := ba.createAsset(memory, memory.BtsMedia, tagBTS); asset != nil {
					groupAssets = append(groupAssets, asset)
				}
			}

			// Create a group for this memory
			// BeReal memories should ALWAYS be stacked with main camera as cover
			if len(groupAssets) > 0 {
				group := assets.NewGroup(assets.GroupByDualCamera, groupAssets...)
				group.SetCover(ba.coverIndex(groupAssets, mainAsset, selfieAsset, memory))

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

// coverIndex returns the index in groupAssets of the asset that represents the memory in Immich.
// This is the main (back camera) asset, which comes first in the group. The BeReal videos make an
// exception: the back camera records no sound, only the front camera does. As Immich shows the
// cover of a stack only, the selfie video becomes the cover when it is the only one with sound.
func (ba *BeRealAdapter) coverIndex(groupAssets []*assets.Asset, mainAsset, selfieAsset *assets.Asset, memory BeRealMemory) int {
	if mainAsset == nil || selfieAsset == nil {
		return 0
	}
	if memory.BackImage.MediaType != "video" || memory.FrontImage.MediaType != "video" {
		return 0
	}
	if hasAudio(mainAsset) || !hasAudio(selfieAsset) {
		return 0
	}
	for i, a := range groupAssets {
		if a == selfieAsset {
			ba.logger.Debug(fmt.Sprintf("BeReal video %s has no sound, the selfie video %s is the cover", mainAsset.OriginalFileName, selfieAsset.OriginalFileName))
			return i
		}
	}
	return 0
}

// hasAudio tells if the video behind the asset holds an audio track.
func hasAudio(a *assets.Asset) bool {
	f, err := a.File.Open()
	if err != nil {
		return false
	}
	defer f.Close()
	rs, ok := f.(io.ReadSeeker)
	return ok && hasAudioTrack(rs)
}

// createAsset converts a BeReal image or video of a memory into an assets.Asset
// tag identifies the role of the media: main camera (back), selfie camera (front) or behind the scenes
func (ba *BeRealAdapter) createAsset(memory BeRealMemory, imageInfo ImageInfo, tag string) *assets.Asset {
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

	// Build description from caption if available
	var description string
	if memory.Caption != "" {
		description = memory.Caption
	}

	// Create metadata with BeReal information
	// This marks the metadata as coming from the application, which is required
	// for the upload process to send it to the server
	md := &assets.Metadata{
		File:        fshelper.FSName(ba.fsys, imageInfo.Path),
		FileName:    fileName,
		DateTaken:   captureDate,
		Latitude:    memory.Location.Latitude,
		Longitude:   memory.Location.Longitude,
		Description: description,
	}

	// Create asset
	asset := &assets.Asset{
		File:             fshelper.FSName(ba.fsys, imageInfo.Path),
		OriginalFileName: fileName,
		FileSize:         int(info.Size()),
		FileDate:         info.ModTime().UTC(),
		CaptureDate:      captureDate,
		Visibility:       assets.VisibilityTimeline,
	}

	// Apply metadata and mark it as from application
	// This is required for location and description to be uploaded to the server
	asset.FromApplication = asset.UseMetadata(md)

	// Add tag
	asset.AddTag(tag)

	// All BeReal assets go into the same album
	asset.Albums = []assets.Album{
		{
			Title:       albumTitle,
			Description: albumDescription,
		},
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
		if memory.BtsMedia.Path != "" {
			count++
		}
	}
	return count, nil
}

// Report returns a summary of discovery
func (ba *BeRealAdapter) Report(ctx context.Context) error {
	total, _ := ba.CountAssets()
	fmt.Printf("BeReal: %d memories found\n", len(ba.memories))
	fmt.Printf("BeReal: %d total assets (main + selfie + behind the scenes)\n", total)
	return nil
}
