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
	"strconv"
	"strings"
	"time"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/fshelper"

	mapset "github.com/deckarep/golang-set/v2"
)

// Adapter implements the adapters.Reader interface for Synology Photos
type Adapter struct {
	// Configuration
	ServerURL     string
	Account       string
	Password      string
	IncludeShared bool
	Albums        []string // Filter: import only these albums
	Tags          []string // Filter: import only items with these tags
	People        []string // Filter: import only items with these people
	SkipFaceData  bool     // Skip face recognition data

	// Internal
	client          *Client
	app             *app.Application
	processor       *fileprocessor.FileProcessor
	albumCache      map[string]Album           // album name -> album
	tagCache        map[string]int             // tag name -> tag id
	peopleCache     map[string]int             // person name -> person id
	albumImageCache map[string]mapset.Set[int] //album name -> set of image ids
}

// NewAdapter creates a new Synology Photos adapter
func NewAdapter(app *app.Application, serverURL, account, password string) *Adapter {
	return &Adapter{
		ServerURL:       serverURL,
		Account:         account,
		Password:        password,
		app:             app,
		processor:       app.FileProcessor(),
		albumCache:      make(map[string]Album),
		tagCache:        make(map[string]int),
		peopleCache:     make(map[string]int),
		albumImageCache: make(map[string]mapset.Set[int]),
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

		// list all image of albums
		for _, album := range albumsToProcess {
			if err := sa.fetchAlbumImages(ctx, album); err != nil {
				sa.app.Log().Error("Failed to fetch album images", "album", album.Name, "error", err)
				continue
			}
		}
		// No specific albums requested - process ALL items
		// This includes both album items and non-album items
		if err := sa.processAllItems(ctx, gOut); err != nil {
			sa.app.Log().Error("Failed to process items", "error", err)
			return
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
		sa.app.Log().Debug("Caching album", "name", album.Name, "id", album.ID)
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

func (sa *Adapter) fetchAlbumImages(ctx context.Context, album Album) error {
	offset := 0
	limit := 100
	for {
		items, err := sa.client.GetAlbumItems(ctx, album.ID, offset, limit, nil)
		if err != nil {
			return fmt.Errorf("get album items: %w", err)
		}
		if sa.albumImageCache[album.Name] == nil {
			sa.albumImageCache[album.Name] = mapset.NewSet[int]()
		}
		for _, item := range items {
			sa.albumImageCache[album.Name].Add(item.ID)
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
			if err := sa.processItem(ctx, &item, gOut); err != nil {
				sa.app.Log().Error("Failed to process item", "filename", item.Filename, "error", err)
			}
		}

		if len(items) < limit {
			sa.app.Log().Info("All items processed")
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
		"tag",
	}

	// Request people if we need them
	if !sa.SkipFaceData || len(sa.People) > 0 {
		additional = append(additional, "person")
	}

	return additional
}

// processItem processes a single item and sends it to the output channel
// For live photos, this creates both the image and video assets
func (sa *Adapter) processItem(ctx context.Context, item *Item, gOut chan *assets.Group) error {
	// Check album filter: if specific albums were requested, skip items not in any of them
	if len(sa.Albums) > 0 && !sa.matchesAlbumFilter(item) {
		return nil
	}

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
		return sa.processLivePhoto(ctx, item, gOut)
	}

	// Regular single asset
	return sa.processSingleItem(ctx, item, gOut)
}

// processLivePhoto handles live photos by creating both image and video assets
// Uses Synology's ZIP download API with proper synchronization
// Note: Synology may return either a ZIP (with both image and video) or just the image file
func (sa *Adapter) processLivePhoto(ctx context.Context, item *Item, gOut chan *assets.Group) error {
	sa.app.Log().Debug("Processing live photo", "filename", item.Filename, "item_id", item.ID)

	// Pre-download to determine if we have a ZIP (with video) or single file
	// This is necessary because hasVideo() needs to know the response type
	var livePhotoSaveDir string
	var err error
	if livePhotoSaveDir, err = sa.preDownloadLive(item, ctx); err != nil {
		return fmt.Errorf("pre-download live photo: %w", err)
	}
	// _ = livePhotoSaveDir
	// list all file in livePhotoSaveDir
	if files, err := os.ReadDir(livePhotoSaveDir); err != nil {
		return fmt.Errorf("read dir faild: %w", err)
	} else {
		for _, file := range files {
			if !file.Type().IsRegular() {
				continue
			}
			sa.app.Log().Debug("Get file in live photo dir", "filename", file.Name(), "item_id", item.ID)
			// Map to asset
			asset, mediaType := sa.mapToAssetForLive(item, filepath.Join(livePhotoSaveDir, file.Name()))
			if asset == nil {
				continue
			}
			// Record discovery
			code := fileevent.DiscoveredImage
			if mediaType == filetypes.TypeVideo {
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
		}
	}
	return nil
}

// processSingleItem processes a regular (non-live) photo or video
func (sa *Adapter) processSingleItem(ctx context.Context, item *Item, gOut chan *assets.Group) error {
	var tempFilePath string
	var err error
	if tempFilePath, err = sa.preDownload(item, strconv.Itoa(item.ID), ctx); err != nil {
		return fmt.Errorf("pre-download item: %w", err)
	}
	// Map to asset
	asset := sa.mapToAsset(item, tempFilePath)
	if asset == nil {
		// Unsupported file type: clean up temp file and skip
		os.Remove(tempFilePath)
		return nil
	}

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

// matchesAlbumFilter checks if the item belongs to any of the requested albums
func (sa *Adapter) matchesAlbumFilter(item *Item) bool {
	for _, images := range sa.albumImageCache {
		if images.Contains(item.ID) {
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

func (sa *Adapter) mapToAssetForLive(item *Item, actuleFilePath string) (*assets.Asset, string) {
	// Determine file type
	ext := strings.ToLower(path.Ext(actuleFilePath))
	mediaType := ""
	if _mediaType, ok := sa.app.GetSupportedMedia()[ext]; !ok {
		sa.app.Log().Warn("Unsupported file type", "filename", actuleFilePath, "ext", ext)
		return nil, ""
	} else {
		mediaType = _mediaType
	}
	if mediaType != filetypes.TypeImage && mediaType != filetypes.TypeVideo {
		sa.app.Log().Warn("Unsupported file type", "filename", actuleFilePath, "ext", ext)
		return nil, ""
	}

	fs := &SimpleFileFS{
		path:   actuleFilePath,
		logger: sa.app.Log().Logger,
	}

	// file, err := os.Open(actuleFilePath)
	captureDate := time.Unix(sa.correctTimestamp(item.Time, sa.app.GetTZ()), 0).In(sa.app.GetTZ())
	asset := &assets.Asset{
		File:             fshelper.FSName(fs, item.Filename),
		OriginalFileName: item.Filename,
		Description:      item.Additional.Description,
		CaptureDate:      captureDate,
	}

	if stat, err := os.Stat(actuleFilePath); err != nil {
		sa.app.Log().Warn("Failed to get file info", "filename", actuleFilePath, "error", err)
		return nil, ""
	} else {
		asset.FileSize = int(stat.Size())
	}
	// Add album if specified
	for albumName, images := range sa.albumImageCache {
		if images.Contains(item.ID) {
			asset.Albums = append(asset.Albums, assets.Album{
				Title: albumName,
			})
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
	asset.AddCloseOption(func(asset *assets.Asset) error {
		sa.app.Log().Debug("Removing file", "filename", actuleFilePath)
		if err := os.Remove(actuleFilePath); err != nil {
			sa.app.Log().Error("Error removing file", "filename", actuleFilePath, "err", err)
			// ignore error
			return nil
		}
		dir := filepath.Dir(actuleFilePath)
		if files, err := os.ReadDir(dir); err != nil {
			sa.app.Log().Error("Error reading dir", "dir", dir, "err", err)
		} else if len(files) == 0 {
			sa.app.Log().Debug("Removing dir", "dir", dir)
			if err := os.RemoveAll(dir); err != nil {
				sa.app.Log().Error("Error removing dir", "dir", dir, "err", err)
			}
		}
		// ignore error
		return nil
	})
	return asset, mediaType
}

// mapToAsset converts a Synology Item to an immich-go Asset
func (sa *Adapter) mapToAsset(item *Item, actuleFilePath string) *assets.Asset {
	// Determine file type
	ext := strings.ToLower(path.Ext(item.Filename))
	if _mediaType, ok := sa.app.GetSupportedMedia()[ext]; !ok {
		sa.app.Log().Warn("Unsupported file type", "filename", item.Filename, "ext", ext)
		return nil
	} else if _mediaType != filetypes.TypeImage && _mediaType != filetypes.TypeVideo {
		sa.app.Log().Warn("Unsupported file type", "filename", item.Filename, "ext", ext)
		return nil
	}

	// Create a custom FS that can read from Synology

	fs := &SimpleFileFS{
		path:   actuleFilePath,
		logger: sa.app.Log().Logger,
	}
	captureDate := time.Unix(sa.correctTimestamp(item.Time, sa.app.GetTZ()), 0).In(sa.app.GetTZ())
	asset := &assets.Asset{
		File:             fshelper.FSName(fs, item.Filename),
		FileSize:         int(item.Filesize),
		OriginalFileName: item.Filename,
		Description:      item.Additional.Description,
		CaptureDate:      captureDate,
	}

	// Add album if specified
	for albumName, images := range sa.albumImageCache {
		if images.Contains(item.ID) {
			asset.Albums = append(asset.Albums, assets.Album{
				Title: albumName,
			})
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
	asset.AddCloseOption(func(asset *assets.Asset) error {
		sa.app.Log().Debug("Removing file", "filename", item.Filename, "filepath", actuleFilePath)
		if err := os.Remove(actuleFilePath); err != nil {
			sa.app.Log().Error("Error removing file", "filename", item.Filename, "filepath", actuleFilePath, "err", err)
		}
		// ignore error
		return nil
	})
	return asset
}
func (sa *Adapter) correctTimestamp(wrong int64, loc *time.Location) int64 {
	t := time.Unix(wrong, 0).UTC()
	for i := 0; i < 2; i++ { // to fix the daylight saving time, twice is enough
		_, offset := t.In(loc).Zone()
		t = time.Unix(wrong-int64(offset), 0).UTC()
	}
	return t.Unix()
}

// handleZipDownload processes a ZIP response containing both image and video
func (sa *Adapter) handleZipDownload(reader io.ReadCloser, tempDir string) error {
	// Save ZIP to temp file
	zipPath := filepath.Join(tempDir, "livephoto.zip")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return fmt.Errorf("create zip temp file: %w", err)
	}
	written, err := io.Copy(zipFile, reader)
	zipFile.Close()
	if err != nil {
		return fmt.Errorf("Fail to save zip, path:%s, %w", zipPath, err)
	}

	sa.app.Log().Debug("Downloaded ZIP to temp file", "path", zipPath, "bytes", written)
	// Verify ZIP by checking file header (magic number)
	if err := verifyZipFile(zipPath); err != nil {
		return fmt.Errorf("Fail to verify zip, path:%s, %w", zipPath, err)
	}

	// Extract ZIP
	zipReader, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("Fail to open zip, path:%s, %w", zipPath, err)
	}
	defer zipReader.Close()

	// s.zipFiles = make(map[string]string)
	unzipFileCount := 0
	for _, zf := range zipReader.File {
		// Extract each file
		extractPath := filepath.Join(tempDir, filepath.Base(zf.Name))
		rc, err := zf.Open()
		if err != nil {
			sa.app.Log().Error("Error extracting file", "filename", zf.Name, "err", err)
			continue
		}

		extractFile, err := os.Create(extractPath)
		if err != nil {
			rc.Close()
			sa.app.Log().Error("Error creating extract file", "filename", zf.Name, "err", err)
			continue
		}

		_, err = io.Copy(extractFile, rc)
		rc.Close()
		extractFile.Close()

		if err != nil {
			sa.app.Log().Error("Error extracting file", "filename", zf.Name, "err", err)
			continue
		}

		sa.app.Log().Debug("Extracted live photo file", "name", zf.Name, "size", zf.UncompressedSize64)
		unzipFileCount++
	}

	// Clean up ZIP file
	os.Remove(zipPath)

	sa.app.Log().Debug("Live photo ZIP extracted", "files", unzipFileCount)
	return nil
}

// handleSingleFileDownload processes a single file response (just the image)
// For non-ZIP responses, we only get the image file. The video may need separate handling.
func (sa *Adapter) handleSingleFileDownload(reader io.ReadCloser, tempDir string, filename string) error {
	// Save the single file
	filePath := filepath.Join(tempDir, filename)
	outFile, err := os.Create(filePath)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	written, err := io.Copy(outFile, reader)
	outFile.Close()

	if err != nil {
		return fmt.Errorf("save file: %w", err)
	}

	sa.app.Log().Debug("Downloaded single file", "filename", filename, "bytes", written)

	return nil
}
func (sa *Adapter) preDownload(item *Item, cacheKey string, ctx context.Context) (string, error) {
	sa.app.Log().Debug("Downloading Synology file", "filename", item.Filename, "item_id", item.ID, "cache_key", cacheKey)

	// Create temp file
	tempFile, err := os.CreateTemp("", "synology-*-"+filepath.Base(item.Filename))
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}
	defer tempFile.Close()

	// Download to temp file
	reader, err := sa.client.DownloadItem(ctx, item.ID, cacheKey)
	if err != nil {
		sa.app.Log().Error("Error downloading Synology file on start", "filename", item.Filename, "err", err)
		return "", err
	}

	n, err := io.Copy(tempFile, reader)
	reader.Close()

	if err != nil {
		sa.app.Log().Error("Error downloading Synology file on copy", "filename", item.Filename, "err", err)
		return "", fmt.Errorf("download to temp: %w", err)
	}
	sa.app.Log().Debug("Downloaded Synology file", "filename", item.Filename, "size", n)

	return tempFile.Name(), nil
}

// preDownloadLive downloads the live photo ahead of time to determine the response type
// This allows hasVideo() to return the correct value before assets are created
func (sa *Adapter) preDownloadLive(item *Item, ctx context.Context) (string, error) {

	sa.app.Log().Debug("Pre-downloading live photo", "filename", item.Filename, "item_id", item.ID)

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "synology-livephoto-*")
	if err != nil {
		return "", fmt.Errorf("create temp dir: %w", err)
	}

	// Download - may return ZIP or single file
	reader, contentType, isZip, err := sa.client.DownloadLivePhoto(ctx, item.ID, item.Filename)
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("download live photo: %w", err)
	}
	sa.app.Log().Debug("Pre-downloaded live photo", "filename", item.Filename, "content_type", contentType,
		"is_zip", isZip, "item_id", item.ID)

	if isZip {
		// reader is a zip file, unzip all file to tempDir
		// Handle ZIP download (contains mutable files)
		if err := sa.handleZipDownload(reader, tempDir); err != nil {
			reader.Close()
			os.RemoveAll(tempDir)
			return "", err
		}
	} else {
		// Handle single file download (just the image)
		if err := sa.handleSingleFileDownload(reader, tempDir, item.Filename); err != nil {
			reader.Close()
			os.RemoveAll(tempDir)
			return "", err
		}
	}
	reader.Close()

	return tempDir, nil
}

// verifyZipFile checks if the file is a valid ZIP by reading its magic number
func verifyZipFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer file.Close()

	// ZIP files start with PK\x03\x04 or PK\x05\x06
	magic := make([]byte, 4)
	_, err = file.Read(magic)
	if err != nil {
		return fmt.Errorf("read magic: %w", err)
	}

	// Check for ZIP local file header (PK\x03\x04) or empty archive (PK\x05\x06)
	if magic[0] != 'P' || magic[1] != 'K' {
		return fmt.Errorf("invalid ZIP magic bytes: %02x %02x %02x %02x", magic[0], magic[1], magic[2], magic[3])
	}

	// Check valid ZIP signatures
	if magic[2] == 0x03 && magic[3] == 0x04 {
		return nil // Local file header
	}
	if magic[2] == 0x05 && magic[3] == 0x06 {
		return nil // Empty archive
	}
	if magic[2] == 0x07 && magic[3] == 0x08 {
		return nil // Spanned archive
	}

	return fmt.Errorf("invalid ZIP signature: %02x %02x", magic[2], magic[3])
}

type SimpleFileFS struct {
	path   string
	logger *slog.Logger
}

func (fs *SimpleFileFS) Open(name string) (fs.File, error) {
	if file, err := os.Open(fs.path); err != nil {
		fs.logger.Error("Error opening file", "filename", name, "err", err)
		return nil, err
	} else {
		return file, nil
	}
}
