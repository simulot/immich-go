package nextcloudmemories

import (
	"context"
	"fmt"
	"path"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/nextcloud"
)

type metadataMappingOptions struct {
	OwnerUID           string
	SyncAlbums         bool
	SyncTags           bool
	TagAlbumMembership bool
}

const memoriesClusterAlbums = "albums"

type memoriesAssetRecord struct {
	Photo         nextcloud.MemoriesPhoto
	CanonicalPath string
	Metadata      *assets.Metadata
}

type indexedMemoriesPhoto struct {
	record   *memoriesAssetRecord
	loaded   bool
	loading  bool
	queued   bool
	ready    chan struct{}
	metadata *assets.Metadata
	loadErr  error
}

type memoriesMetadataIndex struct {
	photosByPath  map[string]*indexedMemoriesPhoto
	photosByID    map[int]*indexedMemoriesPhoto
	photosByBase  map[string][]*indexedMemoriesPhoto
	options       metadataMappingOptions
	infoQuery     nextcloud.MemoriesImageInfoQuery
	client        *nextcloud.Client
	selectedRoots []string
	mu            sync.Mutex
	warmQueue     chan *indexedMemoriesPhoto
	warmWorkers   int
}

func newMemoriesMetadataIndex() *memoriesMetadataIndex {
	return &memoriesMetadataIndex{
		photosByPath: map[string]*indexedMemoriesPhoto{},
		photosByID:   map[int]*indexedMemoriesPhoto{},
		photosByBase: map[string][]*indexedMemoriesPhoto{},
		warmWorkers:  6,
	}
}

func (idx *memoriesMetadataIndex) Get(ctx context.Context, name string) (*assets.Metadata, bool, error) {
	if idx == nil {
		return nil, false, nil
	}
	key := normalizeIndexedPath(name)
	idx.mu.Lock()
	entry, ok := idx.photosByPath[key]
	if !ok {
		entry, ok = idx.lookupByBaseLocked(key)
	}
	if ok {
		idx.enqueueWarmLocked(entry)
	}
	idx.mu.Unlock()
	if !ok {
		return nil, false, nil
	}
	return idx.loadEntry(ctx, entry)
}

func (idx *memoriesMetadataIndex) Put(name string, photo nextcloud.MemoriesPhoto) {
	if idx == nil {
		return
	}
	key := normalizeIndexedPath(name)
	if key == "" && photo.FileID == 0 {
		return
	}
	idx.mu.Lock()
	defer idx.mu.Unlock()
	idx.putLocked(key, photo)
}

func (idx *memoriesMetadataIndex) putLocked(key string, photo nextcloud.MemoriesPhoto) {
	if existing, ok := idx.photosByID[photo.FileID]; ok && photo.FileID != 0 {
		if key != "" {
			idx.photosByPath[key] = existing
			existing.record.CanonicalPath = key
			idx.addBaseIndexLocked(path.Base(key), existing)
		}
		return
	}
	record := &memoriesAssetRecord{Photo: photo, CanonicalPath: key}
	entry := &indexedMemoriesPhoto{record: record}
	if key != "" {
		idx.photosByPath[key] = entry
		idx.addBaseIndexLocked(path.Base(key), entry)
	} else if photo.Basename != "" {
		idx.addBaseIndexLocked(photo.Basename, entry)
	}
	if photo.FileID != 0 {
		idx.photosByID[photo.FileID] = entry
	}
}

func (idx *memoriesMetadataIndex) addBaseIndexLocked(base string, entry *indexedMemoriesPhoto) {
	base = strings.TrimSpace(base)
	if base == "" || entry == nil {
		return
	}
	entries := idx.photosByBase[base]
	for _, existing := range entries {
		if existing == entry {
			return
		}
	}
	idx.photosByBase[base] = append(entries, entry)
}

func (idx *memoriesMetadataIndex) lookupByBaseLocked(key string) (*indexedMemoriesPhoto, bool) {
	base := path.Base(key)
	if base == "." || base == "" {
		return nil, false
	}
	entries := idx.photosByBase[base]
	if len(entries) != 1 {
		return nil, false
	}
	return entries[0], true
}

func (idx *memoriesMetadataIndex) startWarmup(ctx context.Context) {
	if idx == nil || idx.client == nil {
		return
	}
	idx.mu.Lock()
	if idx.warmQueue != nil {
		idx.mu.Unlock()
		return
	}
	queue := make(chan *indexedMemoriesPhoto, 256)
	workers := idx.warmWorkers
	if workers <= 0 {
		workers = 6
	}
	idx.warmQueue = queue
	idx.mu.Unlock()

	for range workers {
		go func() {
			for {
				select {
				case <-ctx.Done():
					return
				case entry := <-queue:
					if entry == nil {
						continue
					}
					_, _, _ = idx.loadEntry(ctx, entry)
				}
			}
		}()
	}
}

func (idx *memoriesMetadataIndex) enqueueWarmLocked(entry *indexedMemoriesPhoto) {
	if idx == nil || entry == nil || idx.warmQueue == nil {
		return
	}
	if entry.loaded || entry.loading || entry.queued {
		return
	}
	entry.queued = true
	select {
	case idx.warmQueue <- entry:
	default:
		entry.queued = false
	}
}

func (idx *memoriesMetadataIndex) loadEntry(ctx context.Context, entry *indexedMemoriesPhoto) (*assets.Metadata, bool, error) {
	if idx == nil || entry == nil {
		return nil, false, nil
	}

	idx.mu.Lock()
	for entry.loading && !entry.loaded {
		ready := entry.ready
		idx.mu.Unlock()
		select {
		case <-ctx.Done():
			return nil, false, ctx.Err()
		case <-ready:
		}
		idx.mu.Lock()
	}
	if entry.loaded {
		md, err := entry.metadata, entry.loadErr
		idx.mu.Unlock()
		if err != nil {
			return nil, true, err
		}
		if md == nil {
			return nil, false, nil
		}
		return md, true, nil
	}
	entry.queued = false
	entry.loading = true
	entry.ready = make(chan struct{})
	ready := entry.ready
	fileID := entry.record.Photo.FileID
	photo := entry.record.Photo
	canonicalPath := entry.record.CanonicalPath
	infoQuery := idx.infoQuery
	client := idx.client
	selectedRoots := slices.Clone(idx.selectedRoots)
	options := idx.options
	idx.mu.Unlock()

	info, err := nextcloud.GetMemoriesImageInfo(ctx, client, fileID, infoQuery)
	if err != nil {
		loadErr := fmt.Errorf("failed to load memories image info for file %d: %w", fileID, err)
		idx.mu.Lock()
		entry.loaded = true
		entry.loading = false
		entry.loadErr = loadErr
		close(ready)
		entry.ready = nil
		idx.mu.Unlock()
		return nil, true, loadErr
	}

	indexedPath := normalizeIndexedPath(info.FileName)
	if indexedPath == "" {
		indexedPath = canonicalPath
	}
	if indexedPath == "" {
		loadErr := fmt.Errorf("memories image info for file %d did not include a filename", fileID)
		idx.mu.Lock()
		entry.loaded = true
		entry.loading = false
		entry.loadErr = loadErr
		close(ready)
		entry.ready = nil
		idx.mu.Unlock()
		return nil, true, loadErr
	}
	if !isSelectedRootPath(indexedPath, selectedRoots) {
		idx.mu.Lock()
		entry.record.CanonicalPath = indexedPath
		idx.photosByPath[indexedPath] = entry
		idx.addBaseIndexLocked(path.Base(indexedPath), entry)
		entry.loaded = true
		entry.loading = false
		entry.metadata = nil
		entry.loadErr = nil
		close(ready)
		entry.ready = nil
		idx.mu.Unlock()
		return nil, false, nil
	}

	md := metadataFromMemories(photo, info, options)
	idx.mu.Lock()
	entry.record.CanonicalPath = indexedPath
	entry.record.Metadata = md
	idx.photosByPath[indexedPath] = entry
	idx.addBaseIndexLocked(path.Base(indexedPath), entry)
	entry.loaded = true
	entry.loading = false
	entry.metadata = md
	entry.loadErr = nil
	close(ready)
	entry.ready = nil
	idx.mu.Unlock()
	return md, true, nil
}

func (nc *Command) ensureMetadataIndex(ctx context.Context) error {
	if nc.metadataIndex != nil {
		return nil
	}
	if nc.client == nil || nc.discovery == nil {
		return nil
	}

	index, err := nc.buildMetadataIndex(ctx)
	if err != nil {
		// Metadata enrichment is intentionally best-effort.
		//
		// The original DAV-backed importer worked without any Memories metadata at
		// all. Keep that behavior available by falling back to plain file import if
		// the source-side metadata APIs fail.
		if nc.app != nil {
			nc.app.Log().Warn("Nextcloud Memories metadata enrichment unavailable; continuing without source metadata", "err", err)
		}
		index = newMemoriesMetadataIndex()
	}
	nc.metadataIndex = index
	return nil
}

func (nc *Command) buildMetadataIndex(ctx context.Context) (*memoriesMetadataIndex, error) {
	photoByID := map[int]nextcloud.MemoriesPhoto{}
	query := nextcloud.MemoriesTimelineQuery{
		Recursive: true,
		Hidden:    true,
	}
	if nc.app != nil {
		nc.app.Log().Message("Preparing Nextcloud Memories metadata index for %s", strings.Join(nc.selectedRoots, ", "))
	}
	if nc.app != nil {
		nc.app.Log().Info("building Nextcloud Memories metadata index", "roots", nc.selectedRoots)
	}

	for _, root := range nc.selectedRoots {
		query.Folder = root
		if nc.app != nil {
			nc.app.Log().Info("listing Memories day buckets", "root", root)
		}

		days, err := nextcloud.GetMemoriesDays(ctx, nc.client, query)
		if err != nil {
			return nil, fmt.Errorf("failed to list Memories days for %q: %w", root, err)
		}
		if nc.app != nil {
			nc.app.Log().Info("listed Memories day buckets", "root", root, "days", len(days))
		}

		dayIDs := make([]int, 0, len(days))
		for _, day := range days {
			dayIDs = append(dayIDs, day.DayID)
		}

		for start := 0; start < len(dayIDs); start += 100 {
			end := min(start+100, len(dayIDs))
			if nc.app != nil {
				nc.app.Log().Debug("listing Memories photos batch", "root", root, "start", start, "end", end, "totalDays", len(dayIDs))
			}
			photos, err := nextcloud.GetMemoriesDay(ctx, nc.client, dayIDs[start:end], query)
			if err != nil {
				return nil, fmt.Errorf("failed to list Memories photos for %q: %w", root, err)
			}
			for _, photo := range photos {
				photoByID[photo.FileID] = mergeMemoriesPhoto(photoByID[photo.FileID], photo)
			}
		}
		if nc.app != nil {
			nc.app.Log().Info("collected Memories timeline photos", "root", root, "uniqueFiles", len(photoByID))
			nc.app.Log().Message("Collected %d indexed files from %s", len(photoByID), root)
		}
	}

	archiveQuery := nextcloud.MemoriesTimelineQuery{
		Recursive: true,
		Archive:   true,
		Hidden:    true,
	}
	if nc.app != nil {
		nc.app.Log().Info("listing Memories archive day buckets")
	}
	archiveDays, err := nextcloud.GetMemoriesDays(ctx, nc.client, archiveQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to list Memories archive days: %w", err)
	}
	if nc.app != nil {
		nc.app.Log().Info("listed Memories archive day buckets", "days", len(archiveDays))
	}
	archiveDayIDs := make([]int, 0, len(archiveDays))
	for _, day := range archiveDays {
		archiveDayIDs = append(archiveDayIDs, day.DayID)
	}
	for start := 0; start < len(archiveDayIDs); start += 100 {
		end := min(start+100, len(archiveDayIDs))
		if nc.app != nil {
			nc.app.Log().Debug("listing Memories archived photos batch", "start", start, "end", end, "totalDays", len(archiveDayIDs))
		}
		photos, err := nextcloud.GetMemoriesDay(ctx, nc.client, archiveDayIDs[start:end], archiveQuery)
		if err != nil {
			return nil, fmt.Errorf("failed to list Memories archived photos: %w", err)
		}
		for _, photo := range photos {
			photo.Archived = true
			photoByID[photo.FileID] = mergeMemoriesPhoto(photoByID[photo.FileID], photo)
		}
	}

	index := newMemoriesMetadataIndex()
	if len(photoByID) == 0 {
		if nc.app != nil {
			nc.app.Log().Info("Nextcloud Memories metadata index is empty; no indexed files found in selected roots")
			nc.app.Log().Message("No indexed files found in the selected Nextcloud Memories roots")
		}
		return index, nil
	}

	options := metadataMappingOptions{
		OwnerUID:           strings.TrimSpace(nc.NextcloudUser),
		SyncAlbums:         nc.SyncAlbums,
		SyncTags:           nc.SyncTags,
		TagAlbumMembership: nc.TagAlbumMembership,
	}
	if nc.discovery.Describe.UID != nil && strings.TrimSpace(*nc.discovery.Describe.UID) != "" {
		options.OwnerUID = strings.TrimSpace(*nc.discovery.Describe.UID)
	}

	infoQuery := nextcloud.MemoriesImageInfoQuery{
		Tags: nc.SyncTags && nc.discovery.Config.SystemTagsEnabled,
	}
	if (nc.SyncAlbums || nc.TagAlbumMembership) && nc.discovery.Config.AlbumsEnabled {
		infoQuery.Clusters = []string{memoriesClusterAlbums}
	}

	index.client = nc.client
	index.selectedRoots = slices.Clone(nc.selectedRoots)
	index.options = options
	index.infoQuery = infoQuery

	for _, photo := range photoByID {
		if photo.FileID == 0 {
			continue
		}
		index.Put("", photo)
	}
	if nc.app != nil {
		nc.app.Log().Info("prepared lazy Nextcloud Memories metadata index", "files", len(index.photosByID))
		nc.app.Log().Message("Prepared lazy metadata index for %d files", len(index.photosByID))
	}
	index.startWarmup(ctx)
	return index, nil
}

func metadataFromMemories(photo nextcloud.MemoriesPhoto, info *nextcloud.MemoriesImageInfo, options metadataMappingOptions) *assets.Metadata {
	md := &assets.Metadata{
		FileName:  info.Basename,
		Archived:  photo.Archived,
		Favorited: bool(photo.IsFavorite),
	}

	if info.DateTaken != 0 {
		md.DateTaken = time.Unix(info.DateTaken, 0).In(time.Local)
	} else if photo.DateTaken != 0 {
		md.DateTaken = time.Unix(photo.DateTaken, 0).In(time.Local)
	}

	if description := memoriesExifString(info.Exif, "Description"); description != "" {
		md.Description = description
	}
	if rating, ok := memoriesExifInt(info.Exif, "Rating"); ok {
		if rating < 0 {
			rating = 0
		}
		if rating > 5 {
			rating = 5
		}
		md.Rating = byte(rating)
	}
	if latitude, ok := memoriesExifFloat(info.Exif, "GPSLatitude"); ok {
		md.Latitude = latitude
	}
	if longitude, ok := memoriesExifFloat(info.Exif, "GPSLongitude"); ok {
		md.Longitude = longitude
	}

	tags := make([]string, 0, len(info.Tags)+len(info.Clusters.Albums))
	if options.SyncTags && len(info.Tags) > 0 {
		for _, tag := range info.Tags {
			tag = strings.TrimSpace(tag)
			if tag == "" {
				continue
			}
			tags = append(tags, tag)
		}
	}

	if len(info.Clusters.Albums) > 0 {
		for _, album := range info.Clusters.Albums {
			if options.TagAlbumMembership {
				if tag := memoriesAlbumMembershipTag(album.AlbumID); tag != "" {
					tags = append(tags, tag)
				}
			}
		}

		if options.SyncAlbums {
			seenAlbums := map[string]struct{}{}
			for _, album := range info.Clusters.Albums {
				title := memoriesOwnedAlbumTitle(album, options.OwnerUID)
				if title == "" {
					continue
				}
				if _, ok := seenAlbums[title]; ok {
					continue
				}
				seenAlbums[title] = struct{}{}
				md.Albums = append(md.Albums, assets.NewAlbum("", title, memoriesOwnedAlbumDescription(album, options.OwnerUID)))
			}
			slices.SortFunc(md.Albums, func(a, b assets.Album) int {
				return strings.Compare(a.Title, b.Title)
			})
		}
	}
	if len(tags) > 0 {
		slices.Sort(tags)
		for _, tag := range tags {
			md.AddTag(tag)
		}
	}
	if len(md.Tags) > 1 {
		slices.SortFunc(md.Tags, func(a, b assets.Tag) int {
			return strings.Compare(a.Value, b.Value)
		})
	}

	return md
}

func mergeMemoriesPhoto(current, incoming nextcloud.MemoriesPhoto) nextcloud.MemoriesPhoto {
	if current.FileID == 0 {
		return incoming
	}
	if incoming.FileID == 0 {
		return current
	}
	if current.Basename == "" {
		current.Basename = incoming.Basename
	}
	if current.MimeType == "" {
		current.MimeType = incoming.MimeType
	}
	if current.DayID == 0 {
		current.DayID = incoming.DayID
	}
	if current.DateTaken == 0 {
		current.DateTaken = incoming.DateTaken
	}
	current.Archived = current.Archived || incoming.Archived
	current.IsFavorite = current.IsFavorite || incoming.IsFavorite
	current.IsHidden = current.IsHidden || incoming.IsHidden
	return current
}

func normalizeIndexedPath(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	cleaned := path.Clean("/" + strings.TrimPrefix(name, "/"))
	if cleaned == "/" {
		return ""
	}
	return strings.TrimPrefix(cleaned, "/")
}

func isSelectedRootPath(name string, roots []string) bool {
	if len(roots) == 0 {
		return true
	}
	for _, root := range roots {
		rootPath := timelineRootToFSPath(root)
		if rootPath == "." || name == rootPath || strings.HasPrefix(name, rootPath+"/") {
			return true
		}
	}
	return false
}

func memoriesExifString(exif map[string]any, key string) string {
	if exif == nil {
		return ""
	}
	raw, ok := exif[key]
	if !ok || raw == nil {
		return ""
	}
	value := strings.TrimSpace(fmt.Sprint(raw))
	if strings.EqualFold(value, "<nil>") {
		return ""
	}
	return value
}

func memoriesExifInt(exif map[string]any, key string) (int, bool) {
	value := memoriesExifString(exif, key)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.Atoi(value)
	if err == nil {
		return parsed, true
	}
	floatValue, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return int(floatValue), true
}

func memoriesExifFloat(exif map[string]any, key string) (float64, bool) {
	value := memoriesExifString(exif, key)
	if value == "" {
		return 0, false
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return parsed, true
}

func memoriesOwnedAlbumTitle(album nextcloud.MemoriesAlbum, ownerUID string) string {
	name := strings.TrimSpace(album.Name)
	if name == "" {
		return ""
	}
	albumUser := strings.TrimSpace(album.User)
	ownerUID = strings.TrimSpace(ownerUID)
	if albumUser != "" && albumUser != ownerUID {
		return ""
	}
	return name
}
