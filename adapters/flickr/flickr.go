package flickr

import (
	"context"
	"errors"
	"io/fs"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/fshelper"
)

// Compile-time assertion that FlickrCmd satisfies adapters.Reader.
var _ adapters.Reader = (*FlickrCmd)(nil)

// Package-level compiled regexes — compiled once, never per-call.
var (
	// rePhotoIDPrimary matches the current Flickr export format: slug_ID_o.ext
	// e.g. "my-photo_12345678_o.jpg"
	rePhotoIDPrimary = regexp.MustCompile(`_(\d+)_o\.\w+$`)

	// rePhotoIDFallback matches the older Flickr export format: ID_hash_o.ext
	// e.g. "12345678_ab1cd2ef3g_o.jpg"
	rePhotoIDFallback = regexp.MustCompile(`^(\d+)_[0-9a-f]+_o\.\w+$`)
)

// assetFile keeps information collected during pass one about a single image file.
type assetFile struct {
	fsys   fs.FS     // the FS partition (archive) in which the file lives
	base   string    // the original filename within that FS
	length int       // file size in bytes
	date   time.Time // file modification time
}

// FlickrCmd holds the runtime state for a Flickr export import operation.
// catalog is keyed by Flickr photo ID (not by directory like the GP adapter) because
// Flickr exports are flat — all images live in a single root with no subdirectories.
type FlickrCmd struct {
	// CLI flags
	CreateAlbums bool // --sync-albums: create Immich albums matching Flickr albums

	// internal state
	app        *app.Application
	processor  *fileprocessor.FileProcessor
	fsyss      []fs.FS
	catalog    map[string]*assetFile     // photo ID → image file entry
	photoMeta  map[string]*FlickrMetadata // photo ID → parsed per-photo JSON
	albumIndex map[string][]string       // photo ID → album titles (from albums.json)
}

// classifyArchives partitions the provided FSes into exactly one metadata FS
// (containing albums.json) and zero or more image FSes.
// It uses fs.Stat rather than WalkDir — O(1) per FS, no directory traversal needed.
func classifyArchives(fsyss []fs.FS) (metaFS fs.FS, imageFS []fs.FS, err error) {
	var metaCandidates []fs.FS

	for _, fsys := range fsyss {
		_, statErr := fs.Stat(fsys, "albums.json")
		if statErr == nil {
			// albums.json present → this is the metadata archive
			metaCandidates = append(metaCandidates, fsys)
		} else if errors.Is(statErr, fs.ErrNotExist) {
			// albums.json absent → this is an image archive
			imageFS = append(imageFS, fsys)
		} else {
			// Unexpected stat error (permissions, I/O, etc.)
			return nil, nil, statErr
		}
	}

	switch len(metaCandidates) {
	case 0:
		return nil, nil, errors.New("no metadata archive found (missing albums.json)")
	case 1:
		return metaCandidates[0], imageFS, nil
	default:
		return nil, nil, errors.New("multiple metadata archives found")
	}
}

// extractPhotoID extracts the Flickr numeric photo ID from an image filename.
// It strips any directory prefix first, then tries the primary pattern (slug_ID_o.ext)
// followed by the fallback pattern (ID_hash_o.ext).
// Returns the ID string and true on success, or "", false if neither pattern matches.
func extractPhotoID(filename string) (photoID string, ok bool) {
	base := path.Base(filename)

	if m := rePhotoIDPrimary.FindStringSubmatch(base); len(m) == 2 {
		return m[1], true
	}
	if m := rePhotoIDFallback.FindStringSubmatch(base); len(m) == 2 {
		return m[1], true
	}
	return "", false
}

// Browse satisfies adapters.Reader. It runs a two-pass goroutine:
//   - passOneImageFS walks each image archive and builds f.catalog
//   - passOneMetaFS walks the metadata archive, reads photo_*.json files and albums.json
//   - passTwo iterates the catalog and emits one assets.Group per photo
func (f *FlickrCmd) Browse(ctx context.Context) chan *assets.Group {
	ctx, cancel := context.WithCancelCause(ctx)
	gOut := make(chan *assets.Group)
	go func() {
		defer close(gOut)

		f.catalog = make(map[string]*assetFile)
		f.photoMeta = make(map[string]*FlickrMetadata)
		f.albumIndex = nil

		metaFS, imageFSes, err := classifyArchives(f.fsyss)
		if err != nil {
			cancel(err)
			return
		}

		// passOne: walk all image archives, then the metadata archive
		for _, imgFS := range imageFSes {
			if err := f.passOneImageFS(ctx, imgFS); err != nil {
				cancel(err)
				return
			}
		}
		if err := f.passOneMetaFS(ctx, metaFS); err != nil {
			cancel(err)
			return
		}

		// passTwo: emit one group per catalog entry
		if err := f.passTwo(ctx, gOut); err != nil {
			cancel(err)
			return
		}
		cancel(nil)
	}()
	return gOut
}

// passOneImageFS walks a single image archive and populates f.catalog.
// Every file receives exactly one fileevent log entry.
func (f *FlickrCmd) passOneImageFS(ctx context.Context, imgFS fs.FS) error {
	return fs.WalkDir(imgFS, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		finfo, err := fs.Stat(imgFS, name)
		if err != nil {
			f.processor.RecordNonAsset(ctx, fshelper.FSName(imgFS, name), 0, fileevent.ErrorFileAccess, "error", err.Error())
			return nil
		}

		ext := strings.ToLower(path.Ext(name))
		mediaType := filetypes.DefaultSupportedMedia.TypeFromExt(ext)

		if mediaType != filetypes.TypeImage && mediaType != filetypes.TypeVideo {
			f.processor.RecordNonAsset(ctx, fshelper.FSName(imgFS, name), finfo.Size(), fileevent.DiscoveredUnsupported, "reason", "unsupported file type")
			return nil
		}

		photoID, ok := extractPhotoID(name)
		if !ok {
			f.processor.RecordNonAsset(ctx, fshelper.FSName(imgFS, name), finfo.Size(), fileevent.DiscoveredUnsupported, "reason", "no photo ID in filename")
			return nil
		}

		if _, exists := f.catalog[photoID]; exists {
			f.processor.RecordNonAsset(ctx, fshelper.FSName(imgFS, name), finfo.Size(), fileevent.DiscoveredUnsupported, "reason", "duplicate photo ID")
			return nil
		}

		f.catalog[photoID] = &assetFile{
			fsys:   imgFS,
			base:   path.Base(name),
			length: int(finfo.Size()),
			date:   finfo.ModTime(),
		}

		code := fileevent.DiscoveredImage
		if mediaType == filetypes.TypeVideo {
			code = fileevent.DiscoveredVideo
		}
		f.processor.RecordAssetDiscovered(ctx, fshelper.FSName(imgFS, name), finfo.Size(), code)
		return nil
	})
}

// passOneMetaFS walks the metadata archive and populates f.photoMeta and f.albumIndex.
func (f *FlickrCmd) passOneMetaFS(ctx context.Context, metaFS fs.FS) error {
	return fs.WalkDir(metaFS, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		base := path.Base(name)

		switch {
		case base == "albums.json":
			parsed, err := fshelper.ReadJSON[FlickrAlbums](metaFS, name)
			if err != nil {
				return err
			}
			f.albumIndex = albumIndex(parsed)
			f.processor.RecordNonAsset(ctx, fshelper.FSName(metaFS, name), 0, fileevent.DiscoveredSidecar, "type", "albums.json")

		case strings.HasPrefix(base, "photo_") && strings.HasSuffix(base, ".json"):
			parsed, err := fshelper.ReadJSON[FlickrMetadata](metaFS, name)
			if err != nil {
				f.processor.RecordNonAsset(ctx, fshelper.FSName(metaFS, name), 0, fileevent.ErrorFileAccess, "error", err.Error())
				return nil
			}
			// Derive photo ID from filename: photo_<id>.json → <id>
			photoID := strings.TrimSuffix(strings.TrimPrefix(base, "photo_"), ".json")
			f.photoMeta[photoID] = parsed
			f.processor.RecordNonAsset(ctx, fshelper.FSName(metaFS, name), 0, fileevent.DiscoveredSidecar, "type", "photo metadata", "id", photoID)

		default:
			f.processor.RecordNonAsset(ctx, fshelper.FSName(metaFS, name), 0, fileevent.DiscoveredUnsupported, "reason", "unrecognised metadata file")
		}

		return nil
	})
}

// passTwo iterates the catalog and emits one assets.Group per photo.
// Photos with no matching JSON are still emitted (with ProcessedMissingMetadata logged).
// Albums from albumIndex are attached when present.
func (f *FlickrCmd) passTwo(ctx context.Context, gOut chan *assets.Group) error {
	// Ensure albumIndex is never nil (handles missing albums.json gracefully)
	if f.albumIndex == nil {
		f.albumIndex = make(map[string][]string)
	}

	for photoID, entry := range f.catalog {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		a := &assets.Asset{
			File:             fshelper.FSName(entry.fsys, entry.base),
			FileSize:         entry.length,
			OriginalFileName: entry.base,
			FileDate:         entry.date,
		}

		if md, ok := f.photoMeta[photoID]; ok {
			converted := md.AsMetadata(fshelper.FSName(entry.fsys, entry.base))
			a.FromApplication = a.UseMetadata(converted)
			// Override OriginalFileName with the human-readable Flickr photo title when available.
			if md.Name != "" {
				a.OriginalFileName = md.Name
			}
		} else {
			f.processor.RecordNonAsset(ctx, fshelper.FSName(entry.fsys, entry.base), int64(entry.length), fileevent.ProcessedMissingMetadata)
		}

		// Attach album membership sourced from albums.json (never from per-photo JSON).
		if titles, ok := f.albumIndex[photoID]; ok {
			albumSlice := make([]assets.Album, 0, len(titles))
			for _, title := range titles {
				albumSlice = append(albumSlice, assets.Album{Title: title})
			}
			a.MergeAlbums(albumSlice)
		}

		select {
		case gOut <- assets.NewGroup(assets.GroupByNone, a):
		case <-ctx.Done():
			return ctx.Err()
		}
	}
	return nil
}
