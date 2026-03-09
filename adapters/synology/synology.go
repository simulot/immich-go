package synology

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"path"
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
	client, err := NewClient(sa.ServerURL, sa.Account, sa.Password, WithInsecureSkipVerify(true))
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
		if len(albumsToProcess) > 0 {
			for _, album := range albumsToProcess {
				if err := sa.processAlbum(ctx, album, gOut); err != nil {
					sa.app.Log().Error("Failed to process album", "album", album.Name, "error", err)
					continue
				}
			}
		} else {
			// Process all items (not album-based)
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
func (sa *Adapter) getAlbumsToProcess(ctx context.Context) ([]Album, error) {
	// Get all albums
	allAlbums := make([]Album, 0)
	offset := 0
	for {
		albums, err := sa.client.ListAlbums(ctx, offset, 1000)
		if err != nil {
			return nil, fmt.Errorf("list albums: %w", err)
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
func (sa *Adapter) processItem(ctx context.Context, item *Item, album *Album, gOut chan *assets.Group) error {
	// Check tag filter
	if len(sa.Tags) > 0 && !sa.matchesTagFilter(item) {
		return nil
	}

	// Check people filter
	if len(sa.People) > 0 && !sa.matchesPeopleFilter(item) {
		return nil
	}

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

	// Add album if specified
	if album != nil {
		asset.Albums = []assets.Album{
			{
				Title: album.Name,
			},
		}
	}

	// Add tags
	for _, tag := range item.Additional.Tag {
		asset.Tags = append(asset.Tags, assets.Tag{
			Name: tag.Name,
		})
	}

	// Add people as tags (prefixed with "Person: ") since Immich handles faces separately
	if !sa.SkipFaceData {
		for _, person := range item.Additional.Person {
			asset.Tags = append(asset.Tags, assets.Tag{
				Name: fmt.Sprintf("Person: %s", person.Name),
			})
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
}

// Open implements fs.FS
func (s *synologyFS) Open(name string) (fs.File, error) {
	if name != s.filename {
		return nil, fmt.Errorf("file not found: %s", name)
	}

	ctx := context.Background()
	reader, err := s.client.DownloadItem(ctx, s.itemID, s.cacheKey)
	if err != nil {
		return nil, err
	}

	return &synologyFile{
		ReadCloser: reader,
		name:       s.filename,
		size:       int64(s.size),
	}, nil
}

// synologyFile implements fs.File
type synologyFile struct {
	io.ReadCloser
	name string
	size int64
	offset int64
}

func (f *synologyFile) Stat() (fs.FileInfo, error) {
	return &synologyFileInfo{
		name: f.name,
		size: f.size,
	}, nil
}

func (f *synologyFile) Read(p []byte) (n int, err error) {
	n, err = f.ReadCloser.Read(p)
	f.offset += int64(n)
	return n, err
}

func (f *synologyFile) Close() error {
	return f.ReadCloser.Close()
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
