package immich

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/fshelper"
)

var _ ImmichInterface = (*ImmichClient)(nil)

// ImmichInterface is an interface that implements the minimal immich client set of features for uploading
// interface used to mock up the client
type ImmichInterface interface {
	SetEndPoint(string)
	EnableAppTrace(w io.Writer)
	SetDeviceUUID(string)
	PingServer(ctx context.Context) error
	ValidateConnection(ctx context.Context) (User, error)
	GetServerStatistics(ctx context.Context) (ServerStatistics, error)
	GetAssetStatistics(ctx context.Context) (UserStatistics, error)
	GetAssetInfo(ctx context.Context, id string) (*Asset, error)
	DownloadAsset(ctx context.Context, id string) (io.ReadCloser, error)

	UpdateAsset(ctx context.Context, id string, param UpdAssetField) (*Asset, error)
	ReplaceAsset(ctx context.Context, ID string, la *assets.Asset) (AssetResponse, error)
	GetAllAssets(ctx context.Context) ([]*Asset, error)
	AddAssetToAlbum(context.Context, string, []string) ([]UpdateAlbumResult, error)
	UpdateAssets(
		ctx context.Context,
		IDs []string,
		isArchived bool,
		isFavorite bool,
		latitude float64,
		longitude float64,
		removeParent bool,
		stackParentID string,
	) error
	GetAllAssetsWithFilter(context.Context, *SearchMetadataQuery, func(*Asset) error) error
	GetAssetsByHash(ctx context.Context, hash string) ([]*Asset, error)
	GetAssetsByImageName(ctx context.Context, name string) ([]*Asset, error)

	AssetUpload(context.Context, *assets.Asset) (AssetResponse, error)
	DeleteAssets(context.Context, []string, bool) error

	GetAllAlbums(ctx context.Context) ([]assets.Album, error)
	GetAlbumInfo(ctx context.Context, id string, withoutAssets bool) (AlbumContent, error)
	CreateAlbum(
		ctx context.Context,
		tilte string,
		description string,
		ids []string,
	) (assets.Album, error)

	// GetAssetAlbums get all albums that an asset belongs to
	GetAssetAlbums(ctx context.Context, assetID string) ([]assets.Album, error)
	DeleteAlbum(ctx context.Context, id string) error

	SupportedMedia() filetypes.SupportedMedia

	GetJobs(ctx context.Context) (map[string]Job, error)
	SendJobCommand(
		ctx context.Context,
		jobID JobID,
		command JobCommand,
		force bool,
	) (SendJobCommandResponse, error)
	CreateJob(ctx context.Context, name JobName) error

	UpsertTags(ctx context.Context, tags []string) ([]TagSimplified, error)
	TagAssets(
		ctx context.Context,
		tagID string,
		assetIDs []string,
	) ([]TagAssetsResponse, error)
	BulkTagAssets(
		ctx context.Context,
		tagIDs []string,
		assetIDs []string,
	) (struct {
		Count int `json:"count"`
	}, error)
}

type ImmichStackInterface interface {
	ImmichInterface
	// CreateStack create a stack with the given assets, the 1st asset is the cover, return the stack ID
	CreateStack(ctx context.Context, ids []string) (string, error)
}

type UnsupportedMedia struct {
	msg string
}

func (u UnsupportedMedia) Error() string {
	return u.msg
}

func (u UnsupportedMedia) Is(target error) bool {
	_, ok := target.(*UnsupportedMedia)
	return ok
}

type PingResponse struct {
	Res string `json:"res"`
}

type User struct {
	ID                   string    `json:"id"`
	Email                string    `json:"email"`
	FirstName            string    `json:"firstName"`
	LastName             string    `json:"lastName"`
	StorageLabel         string    `json:"storageLabel"`
	ExternalPath         string    `json:"externalPath"`
	ProfileImagePath     string    `json:"profileImagePath"`
	ShouldChangePassword bool      `json:"shouldChangePassword"`
	IsAdmin              bool      `json:"isAdmin"`
	CreatedAt            time.Time `json:"createdAt"`
	DeletedAt            time.Time `json:"deletedAt"`
	UpdatedAt            time.Time `json:"updatedAt"`
	OauthID              string    `json:"oauthId"`
}

type List[T comparable] struct {
	list []T
	lock sync.RWMutex
}

func (l *List[T]) Includes(v T) bool {
	l.lock.RLock()
	defer l.lock.RUnlock()
	for i := range l.list {
		if l.list[i] == v {
			return true
		}
	}
	return false
}

func (l *List[T]) Push(v T) {
	l.lock.Lock()
	defer l.lock.Unlock()
	l.list = append(l.list, v)
}

func (l *List[T]) MarshalJSON() ([]byte, error) {
	return nil, errors.New("MarshalJSON not implemented for List[T]")
}

func (l *List[T]) UnmarshalJSON(data []byte) error {
	l.lock.Lock()
	defer l.lock.Unlock()
	if l.list == nil {
		l.list = []T{}
	}
	return json.Unmarshal(data, &l.list)
}

type myBool bool

func (b myBool) String() string {
	if b {
		return "true"
	}
	return "false"
}

// immich Asset simplified
type Asset struct {
	ID               string            `json:"id"`
	DeviceAssetID    string            `json:"deviceAssetId"`
	OwnerID          string            `json:"ownerId"`
	DeviceID         string            `json:"deviceId"`
	Type             string            `json:"type"`
	OriginalPath     string            `json:"originalPath"`
	OriginalFileName string            `json:"originalFileName"`
	Resized          bool              `json:"resized"`
	Thumbhash        string            `json:"thumbhash"`
	FileCreatedAt    ImmichTime        `json:"fileCreatedAt"`
	FileModifiedAt   ImmichTime        `json:"fileModifiedAt"`
	UpdatedAt        ImmichTime        `json:"updatedAt"`
	IsFavorite       bool              `json:"isFavorite"`
	IsArchived       bool              `json:"isArchived"`
	IsTrashed        bool              `json:"isTrashed"`
	Duration         string            `json:"duration"`
	Rating           int               `json:"rating"`
	ExifInfo         ExifInfo          `json:"exifInfo"`
	LivePhotoVideoID string            `json:"livePhotoVideoId"`
	Checksum         string            `json:"checksum"`
	StackParentID    string            `json:"stackParentId"`
	Albums           []AlbumSimplified `json:"-"` // Albums that asset belong to
	Tags             []TagSimplified   `json:"tags"`
	// JustUploaded     bool              `json:"-"` // TO REMOVE
}

// NewAssetFromImmich creates an assets.Asset from an immich.Asset.
func (ia Asset) AsAsset() *assets.Asset {
	a := &assets.Asset{
		FileDate:         ia.FileModifiedAt.Time,
		Description:      ia.ExifInfo.Description,
		OriginalFileName: ia.OriginalFileName,
		ID:               ia.ID,
		CaptureDate:      ia.ExifInfo.DateTimeOriginal.Time,
		Trashed:          ia.IsTrashed,
		Archived:         ia.IsArchived,
		Favorite:         ia.IsFavorite,
		Rating:           ia.Rating,
		Latitude:         ia.ExifInfo.Latitude,
		Longitude:        ia.ExifInfo.Longitude,
		File:             fshelper.FSName(nil, ia.OriginalFileName),
	}
	a.FileSize = int(ia.ExifInfo.FileSizeInByte)
	for _, album := range ia.Albums {
		a.Albums = append(a.Albums, assets.Album{
			Title:       album.AlbumName,
			Description: album.Description,
		})
	}

	for _, tag := range ia.Tags {
		a.Tags = append(a.Tags, tag.AsTag())
	}
	return a
}

type ExifInfo struct {
	Make             string     `json:"make"`
	Model            string     `json:"model"`
	ExifImageWidth   int        `json:"exifImageWidth"`
	ExifImageHeight  int        `json:"exifImageHeight"`
	FileSizeInByte   int64      `json:"fileSizeInByte"`
	Orientation      string     `json:"orientation"`
	DateTimeOriginal ImmichTime `json:"dateTimeOriginal,omitempty"`
	// 	ModifyDate       time.Time `json:"modifyDate"`
	TimeZone string `json:"timeZone"`
	// LensModel        string    `json:"lensModel"`
	// 	FNumber          float64   `json:"fNumber"`
	// 	FocalLength      float64   `json:"focalLength"`
	// 	Iso              int       `json:"iso"`
	// 	ExposureTime     string    `json:"exposureTime"`
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	// 	City             string    `json:"city"`
	// 	State            string    `json:"state"`
	// 	Country          string    `json:"country"`
	Description string `json:"description"`
}

type ImmichTime struct {
	time.Time
}

// ImmichTime.UnmarshalJSON read time from the JSON string.
// The json provides a time UTC, but the server and the images dates are given in local time.
// The get the correct time into the struct, we capture the UTC time and return it in the local zone.
//
// workaround for: error at connection to immich server: cannot parse "+174510-04-28T00:49:44.000Z" as "2006" #28
// capture the error

func (t *ImmichTime) UnmarshalJSON(b []byte) error {
	var ts time.Time
	if len(b) < 3 {
		t.Time = time.Time{}
		return nil
	}
	b = b[1 : len(b)-1]
	ts, err := time.ParseInLocation("2006-01-02T15:04:05.000Z", string(b), time.UTC)
	if err != nil {
		t.Time = time.Time{}
		return nil
	}
	t.Time = ts.In(time.Local)
	return nil
}

func (t ImmichTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return json.Marshal("")
	}

	return json.Marshal(t.Time.Format("\"" + time.RFC3339 + "\""))
}
