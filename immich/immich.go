package immich

import (
	"context"
	"encoding/json"
	"io"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/filetypes"
)

var _ ImmichInterface = (*ImmichClient)(nil)

// ImmichInterface is an interface that implements the minimal immich client set of features for uploading
// interface used to mock up the client
type ImmichInterface interface {
	ImmichAssetInterface
	ImmichClientInterface
	ImmichAlbumInterface
	ImmichTagInterface
	ImmichStackInterface
	ImmichJobInterface
}

type ImmichAssetInterface interface {
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
}

type ImmichClientInterface interface {
	SetEndPoint(string)
	EnableAppTrace(w io.Writer)
	SetDeviceUUID(string)
	PingServer(ctx context.Context) error
	ValidateConnection(ctx context.Context) (User, error)
	GetServerStatistics(ctx context.Context) (ServerStatistics, error)
	GetAssetStatistics(ctx context.Context) (UserStatistics, error)
	SupportedMedia() filetypes.SupportedMedia
}

type ImmichAlbumInterface interface {
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
}
type ImmichTagInterface interface {
	GetAllTags(ctx context.Context) ([]TagSimplified, error)
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
	// CreateStack create a stack with the given assets, the 1st asset is the cover, return the stack ID
	CreateStack(ctx context.Context, ids []string) (string, error)
}

type ImmichJobInterface interface {
	GetJobs(ctx context.Context) (map[string]Job, error)
	SendJobCommand(
		ctx context.Context,
		jobID JobID,
		command JobCommand,
		force bool,
	) (SendJobCommandResponse, error)
	CreateJob(ctx context.Context, name JobName) error
}

type myBool bool

func (b myBool) String() string {
	if b {
		return "true"
	}
	return "false"
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

type ImmichExifTime struct {
	time.Time
}

// ImmichTime.UnmarshalJSON read time from the JSON string.
// The json provides a time UTC, but the server and the images dates are given in local time.
// The get the correct time into the struct, we capture the UTC time and return it in the local zone.
//
// workaround for: error at connection to immich server: cannot parse "+174510-04-28T00:49:44.000Z" as "2006" #28
// capture the error

func (t *ImmichExifTime) UnmarshalJSON(b []byte) error {
	var ts time.Time
	if len(b) < 3 {
		t.Time = time.Time{}
		return nil
	}
	b = b[1 : len(b)-1]
	var err error
	var pattern string
	str := string(b)

	switch len(b) {
	case 29:
		pattern = "2006-01-02T15:04:05.000+00:00"
	case 25:
		pattern = "2006-01-02T15:04:05+00:00"
	}

	if pattern != "" {
		ts, err = time.ParseInLocation(pattern, str, time.UTC)
		if err != nil {
			t.Time = time.Time{}
			return nil
		}
	}

	t.Time = ts.In(time.Local)
	return nil
}

func (t ImmichExifTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return json.Marshal("")
	}

	return json.Marshal(t.Time.Format("\"" + time.RFC3339 + "\""))
}
