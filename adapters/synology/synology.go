package synology

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/simulot/immich-go/internal/fshelper"
)

// Adapter implements the adapters.Reader interface for Synology Photos
type Adapter struct {
	// Configuration
	ServerURL      string
	Account        string
	Password       string
	IncludeShared  bool
	Albums         []string // Filter: import only these albums
	Tags           []string // Filter: import only items with these tags
	People         []string // Filter: import only items with these people
	SkipFaceData   bool     // Skip face recognition data

	// Internal
	client     *Client
	app        *app.Application
	processor  *fileprocessor.FileProcessor
	albumCache map[string]Album // album name -> album
	tagCache   map[string]int   // tag name -> tag id
	peopleCache map[string]int  // person name -> person id
}

// NewAdapter creates a new Synology Photos adapter
func NewAdapter(app *app.Application, serverURL, account, password string) *Adapter {
	return &Adapter{
		ServerURL:   serverURL,
		Account:     account,
		Password:    password,
		app:         app,
		processor:   app.FileProcessor(),
		albumCache:  make(map[string]Album),
		tagCache:    make(map[string]int),
		peopleCache: make(map[string]int),
	}
}

// Open initializes the connection to Synology Photos
func (sa *Adapter) Open(ctx context.Context) error {
	client, err := NewClient(sa.ServerURL, sa.Account, sa.Password,
		WithInsecureSkipVerify(true),
		WithLogger(sa.app.Log().Logger),
	)
	if err != nil {
		return fmt.Errorf("create client: %w", err)
	}

	// Query API info first to verify connection
	if _, err := client.QueryAPIInfo(ctx, "SYNO.API.Auth"); err != nil {
		sa.app.Log().Warn("Failed to query API info", "error", err, "tip", "check if URL is correct and Synology is accessible")
	}

	if err := client.Login(ctx); err != nil {
		return fmt.Errorf("login: %w", err)
	}

	sa.client = client
	return nil
}

// Close closes the connection
func (sa *Adapter) Close(ctx context.Context) error {
	if sa.client != nil {
		return sa.client.Logout(ctx)
	}
	return nil
}

// Browse implements the adapters.Reader interface
func (sa *Adapter) Browse(ctx context.Context) chan *assets.Group {
	gOut := make(chan *assets.Group)

	go func() {
		defer close(gOut)

		// Build caches if filtering
		if err := sa.buildCaches(ctx); err != nil {
			sa.app.Log().Error("Failed to build caches", "error", err)
			return
		}

		// Determine which albums to process
		albumsToProcess, err := sa.getAlbumsToProcess(ctx)
		if err != nil {
			sa.app.Log().Error("Failed to get albums", "error", err)
			return
		}

		// Process albums or all items
		// If user specified specific albums, process only those
		// Otherwise process ALL items (including those not in any album)
		if len(sa.Albums) > 0 && len(albumsToProcess) > 0 {
			// User requested specific albums, process only items in those albums
			for _, album := range albumsToProcess {
				if err := sa.processAlbum(ctx, album, gOut); err != nil {
					sa.app.Log().Error("Failed to process album", "album", album.Name, "error", err)
					continue
				}
			}
		} else {
			// No specific albums requested - process ALL items
			// This includes both album items and non-album items
			if err := sa.processAllItems(ctx, gOut); err != nil {
				sa.app.Log().Error("Failed to process items", "error", err)
				return
			}
		}
	}()

	return gOut
}

// buildCaches builds tag and people caches for filtering
func (sa *Adapter) buildCaches(ctx context.Context) error {
	// Build tag cache if filtering by tags
	if len(sa.Tags) > 0 {
		offset := 0
		for {
			tags, err := sa.client.ListTags(ctx, offset, 1000)
			if err != nil {
				return fmt.Errorf("list tags: %w", err)
			}
			for _, tag := range tags {
				sa.tagCache[tag.Name] = tag.ID
			}
			if len(tags) < 1000 {
				break
			}
			offset += 1000
		}
	}

	// Build people cache if filtering by people
	if len(sa.People) > 0 {
		offset := 0
		for {
			people, err := sa.client.ListPeople(ctx, offset, 1000)
			if err != nil {
				return fmt.Errorf("list people: %w", err)
			}
			for _, person := range people {
				sa.peopleCache[person.Name] = person.ID
			}
			if len(people) < 1000 {
				break
			}
			offset += 1000
		}
	}

	return nil
}

// getAlbumsToProcess returns the list of albums to import
// Note: SYNO.Foto.Browse.Album may not be available in some DSM versions
func (sa *Adapter) getAlbumsToProcess(ctx context.Context) ([]Album, error) {
	// Try to get albums, but if API doesn't exist, return empty list
	// Photos will be imported without album structure
	allAlbums := make([]Album, 0)

	offset := 0
	for {
		albums, err := sa.client.ListAlbums(ctx, offset, 1000)
		if err != nil {
			// If album API is not available, just log warning and return empty
			sa.app.Log().Warn("Album API not available, importing without album structure", "error", err)
			return nil, nil
		}
		allAlbums = append(allAlbums, albums...)
		if len(albums) < 1000 {
			break
		}
		offset += 1000
	}

	// Cache albums by name
	for _, album := range allAlbums {
		sa.albumCache[album.Name] = album
	}

	// If specific albums requested, filter them
	if len(sa.Albums) > 0 {
		filtered := make([]Album, 0, len(sa.Albums))
		for _, name := range sa.Albums {
			if album, ok := sa.albumCache[name]; ok {
				filtered = append(filtered, album)
			} else {
				sa.app.Log().Warn("Album not found", "name", name)
			}
		}
		return filtered, nil
	}

	return allAlbums, nil
}

// processAlbum processes items from a specific album
func (sa *Adapter) processAlbum(ctx context.Context, album Album, gOut chan *assets.Group) error {
	// Clean up album name (trim whitespace)
	album.Name = strings.TrimSpace(album.Name)
	if album.Name == "" {
		sa.app.Log().Warn("Skipping album with empty name", "id", album.ID)
		return nil
	}
	sa.app.Log().Info("Processing album", "name", album.Name, "item_count", album.ItemCount)

	additional := sa.getAdditionalFields()
	offset := 0
	limit := 100

	for {
		items, err := sa.client.GetAlbumItems(ctx, album.ID, offset, limit, additional)
		if err != nil {
			return fmt.Errorf("get album items: %w", err)
		}

		for _, item := range items {
			if err := sa.processItem(ctx, &item, &album, gOut); err != nil {
				sa.app.Log().Error("Failed to process item", "filename", item.Filename, "error", err)
			}
		}

		if len(items) < limit {
			break
		}
		offset += limit
	}

	return nil
}

// processAllItems processes all items without album grouping
func (sa *Adapter) processAllItems(ctx context.Context, gOut chan *assets.Group) error {
	sa.app.Log().Info("Processing all items")

	additional := sa.getAdditionalFields()
	offset := 0
	limit := 100

	for {
		items, err := sa.client.ListAllItems(ctx, offset, limit, additional)
		if err != nil {
			return fmt.Errorf("list items: %w", err)
		}

		for _, item := range items {
			if err := sa.processItem(ctx, &item, nil, gOut); err != nil {
				sa.app.Log().Error("Failed to process item", "filename", item.Filename, "error", err)
			}
		}

		if len(items) < limit {
			break
		}
		offset += limit
	}

	return nil
}

// getAdditionalFields returns the list of additional fields to request
func (sa *Adapter) getAdditionalFields() []string {
	// Always request these
	additional := []string{
		"thumbnail",
		"resolution",
		"orientation",
		"description",
		"gps",
	}

	// Request tags if we need them
	if len(sa.Tags) > 0 || len(sa.Albums) == 0 {
		additional = append(additional, "tag")
	}

	// Request people if we need them
	if !sa.SkipFaceData || len(sa.People) > 0 {
		additional = append(additional, "person")
	}

	return additional
}

// processItem processes a single item and sends it to the output channel
// For live photos, this creates both the image and video assets
func (sa *Adapter) processItem(ctx context.Context, item *Item, album *Album, gOut chan *assets.Group) error {
	// Check tag filter
	if len(sa.Tags) > 0 && !sa.matchesTagFilter(item) {
		return nil
	}

	// Check people filter
	if len(sa.People) > 0 && !sa.matchesPeopleFilter(item) {
		return nil
	}

	// For live photos, we need to create two assets: image and video
	if item.IsLivePhoto() {
		return sa.processLivePhoto(ctx, item, album, gOut)
	}

	// Regular single asset
	return sa.processSingleItem(ctx, item, album, gOut)
}

// processLivePhoto handles live photos by creating both image and video assets
// Uses Synology's ZIP download API to get both files in one request
func (sa *Adapter) processLivePhoto(ctx context.Context, item *Item, album *Album, gOut chan *assets.Group) error {
	sa.app.Log().Debug("Processing live photo", "filename", item.Filename, "item_id", item.ID)

	// Create a shared FS for the live photo ZIP bundle
	liveFS := &synologyLivePhotoFS{
		client:   sa.client,
		itemID:   item.ID,
		filename: item.Filename,
		logger:   sa.app.Log().Logger,
	}

	// Step 1: Process the image part (HEIC/JPG)
	imageAsset := sa.mapToAssetForLiveImage(item, album, liveFS)
	if imageAsset != nil {
		sa.processor.RecordAssetDiscovered(ctx, imageAsset.File, int64(imageAsset.FileSize), fileevent.DiscoveredImage)
		group := assets.NewGroup(assets.GroupByNone, imageAsset)
		select {
		case gOut <- group:
		case <-ctx.Done():
			liveFS.cleanup()
			return ctx.Err()
		}
	}

	// Step 2: Process the video part (MOV)
	videoFilename := item.LivePhotoVideoFilename()
	if videoFilename != "" {
		videoAsset := sa.mapToAssetForLiveVideo(item, album, liveFS, videoFilename)
		if videoAsset != nil {
			sa.processor.RecordAssetDiscovered(ctx, videoAsset.File, int64(videoAsset.FileSize), fileevent.DiscoveredVideo)
			group := assets.NewGroup(assets.GroupByNone, videoAsset)
			select {
			case gOut <- group:
			case <-ctx.Done():
				liveFS.cleanup()
				return ctx.Err()
			}
		}
	} else {
		sa.app.Log().Warn("Live photo has no video filename", "filename", item.Filename)
	}

	// Note: Cleanup happens when files are closed after upload
	// We can't cleanup here because the files haven't been read yet

	return nil
}

// processSingleItem processes a regular (non-live) photo or video
func (sa *Adapter) processSingleItem(ctx context.Context, item *Item, album *Album, gOut chan *assets.Group) error {
	// Map to asset
	asset := sa.mapToAsset(item, album)

	// Record discovery
	code := fileevent.DiscoveredImage
	if item.IsVideo() {
		code = fileevent.DiscoveredVideo
	}
	sa.processor.RecordAssetDiscovered(ctx, asset.File, int64(asset.FileSize), code)

	// Send to output
	group := assets.NewGroup(assets.GroupByNone, asset)
	select {
	case gOut <- group:
	case <-ctx.Done():
		return ctx.Err()
	}

	return nil
}

// matchesTagFilter checks if the item matches the tag filter
func (sa *Adapter) matchesTagFilter(item *Item) bool {
	if len(sa.Tags) == 0 {
		return true
	}

	itemTags := make(map[string]bool)
	for _, tag := range item.Additional.Tag {
		itemTags[tag.Name] = true
	}

	for _, filterTag := range sa.Tags {
		if itemTags[filterTag] {
			return true
		}
	}

	return false
}

// matchesPeopleFilter checks if the item matches the people filter
func (sa *Adapter) matchesPeopleFilter(item *Item) bool {
	if len(sa.People) == 0 {
		return true
	}

	itemPeople := make(map[string]bool)
	for _, person := range item.Additional.Person {
		itemPeople[person.Name] = true
	}

	for _, filterPerson := range sa.People {
		if itemPeople[filterPerson] {
			return true
		}
	}

	return false
}

// mapToAssetForLiveImage creates an Asset for the image part of a live photo
// Uses the shared liveFS which downloads and extracts the ZIP bundle
func (sa *Adapter) mapToAssetForLiveImage(item *Item, album *Album, liveFS *synologyLivePhotoFS) *assets.Asset {
	sa.app.Log().Debug("Creating live photo image asset", "filename", item.Filename, "item_id", item.ID)

	// Use the shared live photo FS (ZIP bundle)
	// FileSize will be determined when the file is opened
	imageAsset := &assets.Asset{
		File:             fshelper.FSName(liveFS, item.Filename),
		FileSize:         int(item.Filesize), // Approximate size from API
		OriginalFileName: item.Filename,
		FileDate:         item.IndexedAt(),
		CaptureDate:      item.CaptureTime(),
		Description:      item.Additional.Description,
		Latitude:         item.Additional.GPS.Latitude,
		Longitude:        item.Additional.GPS.Longitude,
	}

	// Add album if specified
	if album != nil {
		albumName := strings.TrimSpace(album.Name)
		if albumName != "" {
			imageAsset.Albums = []assets.Album{
				{
					Title: albumName,
				},
			}
		}
	}

	// Add tags
	for _, tag := range item.Additional.Tag {
		if tag.Name != "" {
			imageAsset.Tags = append(imageAsset.Tags, assets.Tag{
				Name:  tag.Name,
				Value: tag.Name,
			})
		}
	}

	// Add people as tags
	if !sa.SkipFaceData {
		for _, person := range item.Additional.Person {
			if person.Name != "" {
				personTag := fmt.Sprintf("Person: %s", person.Name)
				imageAsset.Tags = append(imageAsset.Tags, assets.Tag{
					Name:  personTag,
					Value: personTag,
				})
			}
		}
	}

	// Store original metadata
	imageAsset.FromApplication = &assets.Metadata{
		FileName:    item.Filename,
		DateTaken:   item.CaptureTime(),
		Description: item.Additional.Description,
		Latitude:    item.Additional.GPS.Latitude,
		Longitude:   item.Additional.GPS.Longitude,
	}

	// Copy tags to metadata
	for _, tag := range imageAsset.Tags {
		imageAsset.FromApplication.Tags = append(imageAsset.FromApplication.Tags, tag)
	}

	return imageAsset
}

// mapToAssetForLiveVideo creates an Asset for the video part of a live photo
// Uses the shared liveFS which contains both HEIC and MOV from the ZIP bundle
func (sa *Adapter) mapToAssetForLiveVideo(item *Item, album *Album, liveFS *synologyLivePhotoFS, videoFilename string) *assets.Asset {
	sa.app.Log().Debug("Creating live photo video asset", "filename", videoFilename, "item_id", item.ID)

	// Use the shared live photo FS (ZIP bundle)
	// FileSize is approximate - actual size comes from extracted file
	videoAsset := &assets.Asset{
		File:             fshelper.FSName(liveFS, videoFilename),
		FileSize:         int(item.Filesize), // Approximate size
		OriginalFileName: videoFilename,
		FileDate:         item.IndexedAt(),
		CaptureDate:      item.CaptureTime(),
		Description:      item.Additional.Description,
		Latitude:         item.Additional.GPS.Latitude,
		Longitude:        item.Additional.GPS.Longitude,
	}

	// Add album if specified
	if album != nil {
		albumName := strings.TrimSpace(album.Name)
		if albumName != "" {
			videoAsset.Albums = []assets.Album{
				{
					Title: albumName,
				},
			}
		}
	}

	// Store original metadata
	videoAsset.FromApplication = &assets.Metadata{
		FileName:    videoFilename,
		DateTaken:   item.CaptureTime(),
		Description: item.Additional.Description,
		Latitude:    item.Additional.GPS.Latitude,
		Longitude:   item.Additional.GPS.Longitude,
	}

	return videoAsset
}

// mapToAsset converts a Synology Item to an immich-go Asset
func (sa *Adapter) mapToAsset(item *Item, album *Album) *assets.Asset {
	// Determine file type
	ext := strings.ToLower(path.Ext(item.Filename))
	if ext == "" {
		// Try to determine from type
		switch item.Type {
		case "photo":
			ext = ".jpg"
		case "video":
			ext = ".mp4"
		}
	}

	// Create a custom FS that can read from Synology
	synFS := &synologyFS{
		client:   sa.client,
		itemID:   item.ID,
		cacheKey: item.Additional.Thumbnail.CacheKey,
		filename: item.Filename,
		size:     int(item.Filesize),
		logger:   sa.app.Log().Logger,
	}

	asset := &assets.Asset{
		File:             fshelper.FSName(synFS, item.Filename),
		FileSize:         int(item.Filesize),
		OriginalFileName: item.Filename,
		FileDate:         item.IndexedAt(),
		CaptureDate:      item.CaptureTime(),
		Description:      item.Additional.Description,
		Latitude:         item.Additional.GPS.Latitude,
		Longitude:        item.Additional.GPS.Longitude,
	}

	// Add album if specified (with cleaned up name)
	if album != nil {
		albumName := strings.TrimSpace(album.Name)
		if albumName != "" {
			asset.Albums = []assets.Album{
				{
					Title: albumName,
				},
			}
		}
	}

	// Add tags (skip empty names)
	for _, tag := range item.Additional.Tag {
		if tag.Name != "" {
			asset.Tags = append(asset.Tags, assets.Tag{
				Name:  tag.Name,
				Value: tag.Name,
			})
		}
	}

	// Add people as tags (prefixed with "Person: ") since Immich handles faces separately
	// Skip empty person names
	if !sa.SkipFaceData {
		for _, person := range item.Additional.Person {
			if person.Name != "" {
				personTag := fmt.Sprintf("Person: %s", person.Name)
				asset.Tags = append(asset.Tags, assets.Tag{
					Name:  personTag,
					Value: personTag,
				})
			}
		}
	}

	// Store original metadata
	asset.FromApplication = &assets.Metadata{
		FileName:    item.Filename,
		DateTaken:   item.CaptureTime(),
		Description: item.Additional.Description,
		Latitude:    item.Additional.GPS.Latitude,
		Longitude:   item.Additional.GPS.Longitude,
	}

	// Copy tags to metadata
	for _, tag := range asset.Tags {
		asset.FromApplication.Tags = append(asset.FromApplication.Tags, tag)
	}

	return asset
}

// Ensure Adapter implements adapters.Reader
var _ adapters.Reader = (*Adapter)(nil)

// synologyFS implements fs.FS to provide access to Synology files
type synologyFS struct {
	client   *Client
	itemID   int
	cacheKey string
	filename string
	size     int
	logger   *slog.Logger // Optional logger for debugging
}

// Open implements fs.FS
// Downloads the file to a local temp file first to avoid network timeout during upload
func (s *synologyFS) Open(name string) (fs.File, error) {
	if name != s.filename {
		return nil, fmt.Errorf("file not found: %s", name)
	}

	if s.logger != nil {
		s.logger.Debug("Downloading Synology file", "filename", s.filename, "item_id", s.itemID, "cache_key", s.cacheKey)
	}

	ctx := context.Background()

	// Create temp file
	tempFile, err := os.CreateTemp("", "synology-*-"+filepath.Base(s.filename))
	if err != nil {
		return nil, fmt.Errorf("create temp file: %w", err)
	}

	// Download to temp file
	reader, err := s.client.DownloadItem(ctx, s.itemID, s.cacheKey)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, err
	}

	n, err := io.Copy(tempFile, reader)
	reader.Close()

	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, fmt.Errorf("download to temp: %w", err)
	}

	if s.logger != nil {
		s.logger.Debug("Downloaded Synology file", "filename", s.filename, "size", n)
	}

	// Seek to beginning for reading
	_, err = tempFile.Seek(0, 0)
	if err != nil {
		tempFile.Close()
		os.Remove(tempFile.Name())
		return nil, fmt.Errorf("seek temp file: %w", err)
	}

	return &synologyFile{
		File:   tempFile,
		name:   s.filename,
		size:   int64(s.size),
		tempPath: tempFile.Name(),
	}, nil
}

// synologyFile implements fs.File
type synologyFile struct {
	*os.File
	name     string
	size     int64
	tempPath string
}

func (f *synologyFile) Stat() (fs.FileInfo, error) {
	return &synologyFileInfo{
		name: f.name,
		size: f.size,
	}, nil
}

func (f *synologyFile) Close() error {
	// Close the file and remove temp file
	err := f.File.Close()
	os.Remove(f.tempPath)
	return err
}

// synologyFileInfo implements fs.FileInfo
type synologyFileInfo struct {
	name string
	size int64
}

func (fi *synologyFileInfo) Name() string       { return fi.name }
func (fi *synologyFileInfo) Size() int64        { return fi.size }
func (fi *synologyFileInfo) Mode() fs.FileMode  { return 0444 } // Read-only
func (fi *synologyFileInfo) ModTime() time.Time { return time.Now() }
func (fi *synologyFileInfo) IsDir() bool        { return false }
func (fi *synologyFileInfo) Sys() interface{}   { return nil }

// synologyLivePhotoFS implements fs.FS for live photos (ZIP bundle)
// Downloads a ZIP containing both HEIC and MOV files, extracts on demand
type synologyLivePhotoFS struct {
	client     *Client
	itemID     int
	filename   string            // e.g., "IMG_9948.HEIC"
	zipFiles   map[string]string // filename -> temp path
	tempDir    string
	logger     *slog.Logger
	downloaded bool
	refCount   int // Number of open files
}

// Open implements fs.FS - downloads ZIP on first call, returns requested file
func (s *synologyLivePhotoFS) Open(name string) (fs.File, error) {
	// Only allow opening the main file (HEIC) or its corresponding video (MOV)
	expectedVideo := s.filename[:len(s.filename)-len(filepath.Ext(s.filename))] + ".mov"
	if name != s.filename && strings.ToLower(name) != strings.ToLower(expectedVideo) {
		return nil, fmt.Errorf("file not found: %s", name)
	}

	ctx := context.Background()

	// Download and extract ZIP on first access
	if !s.downloaded {
		if s.logger != nil {
			s.logger.Debug("Downloading live photo ZIP", "filename", s.filename, "item_id", s.itemID)
		}

		// Create temp directory
		tempDir, err := os.MkdirTemp("", "synology-livephoto-*")
		if err != nil {
			return nil, fmt.Errorf("create temp dir: %w", err)
		}
		s.tempDir = tempDir

		// Download ZIP
		reader, err := s.client.DownloadLivePhotoZip(ctx, s.itemID, s.filename)
		if err != nil {
			os.RemoveAll(s.tempDir)
			s.tempDir = ""
			return nil, fmt.Errorf("download live photo ZIP: %w", err)
		}

		// Save ZIP to temp file
		zipPath := filepath.Join(tempDir, "livephoto.zip")
		zipFile, err := os.Create(zipPath)
		if err != nil {
			reader.Close()
			os.RemoveAll(s.tempDir)
			s.tempDir = ""
			return nil, fmt.Errorf("create zip temp file: %w", err)
		}

		_, err = io.Copy(zipFile, reader)
		reader.Close()
		zipFile.Close()

		if err != nil {
			os.RemoveAll(s.tempDir)
			s.tempDir = ""
			return nil, fmt.Errorf("save zip: %w", err)
		}

		// Extract ZIP
		zipReader, err := zip.OpenReader(zipPath)
		if err != nil {
			os.RemoveAll(s.tempDir)
			s.tempDir = ""
			return nil, fmt.Errorf("open zip: %w", err)
		}

		s.zipFiles = make(map[string]string)
		for _, zf := range zipReader.File {
			// Extract each file
			extractPath := filepath.Join(tempDir, filepath.Base(zf.Name))
			rc, err := zf.Open()
			if err != nil {
				zipReader.Close()
				os.RemoveAll(s.tempDir)
				s.tempDir = ""
				return nil, fmt.Errorf("extract %s: %w", zf.Name, err)
			}

			extractFile, err := os.Create(extractPath)
			if err != nil {
				rc.Close()
				zipReader.Close()
				os.RemoveAll(s.tempDir)
				s.tempDir = ""
				return nil, fmt.Errorf("create extract file: %w", err)
			}

			_, err = io.Copy(extractFile, rc)
			rc.Close()
			extractFile.Close()

			if err != nil {
				zipReader.Close()
				os.RemoveAll(s.tempDir)
				s.tempDir = ""
				return nil, fmt.Errorf("extract %s: %w", zf.Name, err)
			}

			s.zipFiles[filepath.Base(zf.Name)] = extractPath
			if s.logger != nil {
				s.logger.Debug("Extracted live photo file", "name", zf.Name, "size", zf.UncompressedSize64)
			}
		}
		zipReader.Close()

		// Clean up ZIP file
		os.Remove(zipPath)

		s.downloaded = true

		if s.logger != nil {
			s.logger.Debug("Live photo ZIP downloaded and extracted", "files", len(s.zipFiles))
		}
	}

	// Return the requested file
	lowerName := strings.ToLower(name)
	for fname, fpath := range s.zipFiles {
		if strings.ToLower(fname) == lowerName {
			f, err := os.Open(fpath)
			if err != nil {
				return nil, err
			}

			// Get file info for size
			info, err := f.Stat()
			if err != nil {
				f.Close()
				return nil, err
			}

			s.refCount++
		return &synologyLivePhotoFile{
				File:     f,
				name:     fname,
				size:     info.Size(),
				tempPath: fpath,
				fs:       s,
			}, nil
		}
	}

	return nil, fmt.Errorf("file not found in live photo: %s", name)
}

// cleanup removes the temp directory and all extracted files
func (s *synologyLivePhotoFS) cleanup() {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
		s.tempDir = ""
		s.zipFiles = nil
		s.downloaded = false
	}
}

// synologyLivePhotoFile implements fs.File for live photo extracted files
type synologyLivePhotoFile struct {
	*os.File
	name     string
	size     int64
	tempPath string
	fs       *synologyLivePhotoFS
}

func (f *synologyLivePhotoFile) Stat() (fs.FileInfo, error) {
	return &synologyFileInfo{
		name: f.name,
		size: f.size,
	}, nil
}

func (f *synologyLivePhotoFile) Close() error {
	err := f.File.Close()
	// Decrement ref count and cleanup if this is the last file
	if f.fs != nil {
		f.fs.refCount--
		if f.fs.refCount <= 0 {
			f.fs.cleanup()
		}
	}
	return err
}
