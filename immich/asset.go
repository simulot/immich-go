package immich

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper"
)

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
	LibraryID        string            `json:"libraryId,omitempty"`
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
		FileSize:         int(ia.ExifInfo.FileSizeInByte),
		Checksum:         ia.Checksum,
	}
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
	Make             string         `json:"make"`
	Model            string         `json:"model"`
	ExifImageWidth   int            `json:"exifImageWidth"`
	ExifImageHeight  int            `json:"exifImageHeight"`
	FileSizeInByte   int64          `json:"fileSizeInByte"`
	Orientation      string         `json:"orientation"`
	DateTimeOriginal ImmichExifTime `json:"dateTimeOriginal,omitempty"`
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

type AssetResponse struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

const (
	UploadCreated   = "created"
	UploadReplaced  = "replaced"
	UploadDuplicate = "duplicate"
)

func formatDuration(duration time.Duration) string {
	hours := duration / time.Hour
	duration -= hours * time.Hour

	minutes := duration / time.Minute
	duration -= minutes * time.Minute

	seconds := duration / time.Second
	duration -= seconds * time.Second

	milliseconds := duration / time.Millisecond

	return fmt.Sprintf("%02d:%02d:%02d.%06d", hours, minutes, seconds, milliseconds)
}

const (
	StatusCreated   = "created"
	StatusReplaced  = "replaced"
	StatusDuplicate = "duplicate"
)

func (ic *ImmichClient) AssetUpload(ctx context.Context, la *assets.Asset) (AssetResponse, error) {
	return ic.uploadAsset(ctx, la, EndPointAssetUpload, "")
}

func (ic *ImmichClient) ReplaceAsset(ctx context.Context, ID string, la *assets.Asset) (AssetResponse, error) {
	return ic.uploadAsset(ctx, la, EndPointAssetReplace, ID)
}

type GetAssetOptions struct {
	UserID        string
	IsFavorite    bool
	IsArchived    bool
	WithoutThumbs bool
	Skip          string
}

func (o *GetAssetOptions) Values() url.Values {
	if o == nil {
		return url.Values{}
	}
	v := url.Values{}
	v.Add("userId", o.UserID)
	v.Add("isFavorite", myBool(o.IsFavorite).String())
	v.Add("isArchived", myBool(o.IsArchived).String())
	v.Add("withoutThumbs", myBool(o.WithoutThumbs).String())
	v.Add("skip", o.Skip)
	return v
}

func (ic *ImmichClient) DeleteAssets(ctx context.Context, id []string, forceDelete bool) error {
	if ic.dryRun {
		return nil
	}
	req := struct {
		Force bool     `json:"force"`
		IDs   []string `json:"ids"`
	}{
		IDs:   id,
		Force: forceDelete,
	}

	return ic.newServerCall(ctx, "DeleteAsset").do(deleteRequest("/assets", setJSONBody(&req)))
}

func (ic *ImmichClient) GetAssetInfo(ctx context.Context, id string) (*Asset, error) {
	r := Asset{}
	err := ic.newServerCall(ctx, "GetAssetInfo").do(getRequest("/assets/"+id, setAcceptJSON()), responseJSON(&r))
	return &r, err
}

func (ic *ImmichClient) UpdateAssets(ctx context.Context, ids []string,
	isArchived bool, isFavorite bool,
	latitude float64, longitude float64,
	removeParent bool, stackParentID string,
) error {
	if ic.dryRun {
		return nil
	}
	type updAssets struct {
		IDs           []string `json:"ids"`
		IsArchived    bool     `json:"isArchived"`
		IsFavorite    bool     `json:"isFavorite"`
		Latitude      float64  `json:"latitude"`
		Longitude     float64  `json:"longitude"`
		RemoveParent  bool     `json:"removeParent"`
		StackParentID string   `json:"stackParentId,omitempty"`
	}

	param := updAssets{
		IDs:           ids,
		IsArchived:    isArchived,
		IsFavorite:    isFavorite,
		Latitude:      latitude,
		Longitude:     longitude,
		RemoveParent:  removeParent,
		StackParentID: stackParentID,
	}
	return ic.newServerCall(ctx, "updateAssets").do(putRequest("/assets", setJSONBody(param)))
}

// UpdAssetField is used to update asset with fields given in the struct fields
type UpdAssetField struct {
	IsArchived       bool      `json:"isArchived,omitempty"`
	IsFavorite       bool      `json:"isFavorite,omitempty"`
	Latitude         float64   `json:"latitude,omitempty"`
	Longitude        float64   `json:"longitude,omitempty"`
	Description      string    `json:"description,omitempty"`
	Rating           int       `json:"rating,omitempty"`
	DateTimeOriginal time.Time `json:"dateTimeOriginal,omitempty"`
}

// MarshalJSON customizes the JSON marshaling for the UpdAssetField struct.
// If either Latitude or Longitude is non-zero, it includes them in the JSON output.
// Otherwise, it omits them by using the alias type.
func (u UpdAssetField) MarshalJSON() ([]byte, error) {
	// withGPS is a struct that always includes Latitude and Longitude in the JSON output.
	type withGPS struct {
		IsArchived       bool      `json:"isArchived,omitempty"`
		IsFavorite       bool      `json:"isFavorite,omitempty"`
		Latitude         float64   `json:"latitude"`
		Longitude        float64   `json:"longitude"`
		Description      string    `json:"description,omitempty"`
		Rating           int       `json:"rating,omitempty"`
		DateTimeOriginal time.Time `json:"dateTimeOriginal,omitempty"`
	}

	// alias is used to omit Latitude and Longitude when they are zero.
	type alias UpdAssetField

	// Check if Latitude or Longitude is non-zero, and use withGPS if true.
	if u.Latitude != 0 || u.Longitude != 0 {
		return json.Marshal(withGPS(u))
	}

	// Otherwise, use alias to omit Latitude and Longitude.
	return json.Marshal(alias(u))
}

func (ic *ImmichClient) UpdateAsset(ctx context.Context, id string, param UpdAssetField) (*Asset, error) {
	if ic.dryRun {
		return nil, nil
	}
	r := Asset{}
	err := ic.newServerCall(ctx, "updateAsset").do(putRequest("/assets/"+id, setJSONBody(param)), responseJSON(&r))
	return &r, err
}

func (ic *ImmichClient) DownloadAsset(ctx context.Context, id string) (io.ReadCloser, error) {
	var rc io.ReadCloser

	err := ic.newServerCall(ctx, "DownloadAsset").do(getRequest(fmt.Sprintf("/assets/%s/original", id), setOctetStream()), responseOctetStream(&rc))
	return rc, err
}
