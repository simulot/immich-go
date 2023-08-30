package immich

import (
	"bytes"
	"context"
	"fmt"
	"immich-go/immich/assets"
	"io"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"path"
	"strings"
	"time"
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
	FileCreatedAt    time.Time         `json:"fileCreatedAt"`
	FileModifiedAt   time.Time         `json:"fileModifiedAt"`
	UpdatedAt        time.Time         `json:"updatedAt"`
	IsFavorite       bool              `json:"isFavorite"`
	IsArchived       bool              `json:"isArchived"`
	Duration         string            `json:"duration"`
	ExifInfo         ExifInfo          `json:"exifInfo"`
	LivePhotoVideoID any               `json:"livePhotoVideoId"`
	Tags             []any             `json:"tags"`
	Checksum         string            `json:"checksum"`
	JustUploaded     bool              `json:"-"`
	Albums           []AlbumSimplified `json:"-"` // Albums that asset belong to
}

type ExifInfo struct {
	Make             string     `json:"make"`
	Model            string     `json:"model"`
	ExifImageWidth   int        `json:"exifImageWidth"`
	ExifImageHeight  int        `json:"exifImageHeight"`
	FileSizeInByte   int        `json:"fileSizeInByte"`
	Orientation      string     `json:"orientation"`
	DateTimeOriginal *time.Time `json:"dateTimeOriginal"`
	// 	ModifyDate       time.Time `json:"modifyDate"`
	TimeZone string `json:"timeZone"`
	// LensModel        string    `json:"lensModel"`
	// 	FNumber          float64   `json:"fNumber"`
	// 	FocalLength      float64   `json:"focalLength"`
	// 	Iso              int       `json:"iso"`
	// 	ExposureTime     string    `json:"exposureTime"`
	Latitude  *float64 `json:"latitude,omitempty"`
	Longitude *float64 `json:"longitude,omitempty"`
	// 	City             string    `json:"city"`
	// 	State            string    `json:"state"`
	// 	Country          string    `json:"country"`
	// 	Description      string    `json:"description"`
}

type AssetResponse struct {
	ID        string `json:"id"`
	Duplicate bool   `json:"duplicate"`
}

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

func (ic *ImmichClient) AssetUpload(ctx context.Context, la *assets.LocalAssetFile) (AssetResponse, error) {
	var ar AssetResponse

	// Check the mime type with the first 4k of the file
	b4k := bytes.NewBuffer(nil)
	f, err := la.Open()
	if err != nil {
		return ar, (err)
	}

	_, err = io.CopyN(b4k, f, 16*1024)
	if err != nil && err != io.EOF {
		return ar, (err)
	}

	mtype, err := GetMimeType(b4k.Bytes())
	if err != nil {
		return ar, err
	}

	body, pw := io.Pipe()
	m := multipart.NewWriter(pw)

	go func() {
		defer func() {
			m.Close()
			pw.Close()
		}()
		s, err := f.Stat()
		if err != nil {
			return
		}
		assetType := strings.ToUpper(strings.Split(mtype, "/")[0])

		m.WriteField("deviceAssetId", fmt.Sprintf("%s-%d", path.Base(la.Title), s.Size()))
		m.WriteField("deviceId", ic.DeviceUUID)
		m.WriteField("assetType", assetType)
		m.WriteField("fileCreatedAt", la.DateTaken.Format(time.RFC3339))
		m.WriteField("fileModifiedAt", s.ModTime().Format(time.RFC3339))
		m.WriteField("isFavorite", "false")
		m.WriteField("fileExtension", path.Ext(la.FileName))
		m.WriteField("duration", formatDuration(0))
		m.WriteField("isReadOnly", "false")
		// m.WriteField("isArchived", myBool(la.Archived).String()) // Not supported by the api
		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition",
			fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
				escapeQuotes("assetData"), escapeQuotes(path.Base(la.Title))))
		h.Set("Content-Type", mtype)

		part, err := m.CreatePart(h)
		if err != nil {
			return
		}
		_, err = io.Copy(part, io.MultiReader(b4k, f))
		if err != nil {
			return
		}

		if la.SideCar != nil {
			h.Set("Content-Disposition",
				fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
					escapeQuotes("sidecarData"), escapeQuotes(path.Base(la.SideCar.FileName))))
			h.Set("Content-Type", "application/xml")

			part, err := m.CreatePart(h)
			if err != nil {
				return
			}
			sc, err := la.SideCar.Open(la.FSys, la.SideCar.FileName)
			if err != nil {
				return
			}
			defer sc.Close()
			_, err = io.Copy(part, sc)
			if err != nil {
				return
			}
		}
	}()

	err = ic.newServerCall(ctx, "AssetUpload").
		do(post("/asset/upload", m.FormDataContentType(), setAcceptJSON(), setBody(body)), responseJSON(&ar))

	return ar, err

}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

type GetAssetOptions struct {
	UserId        string
	IsFavorite    bool
	IsArchived    bool
	WithoutThumbs bool
	Skip          string
}

func (o *GetAssetOptions) Values() url.Values {
	if o == nil {
		return nil
	}
	v := url.Values{}
	v.Add("userId", o.UserId)
	v.Add("isFavorite", myBool(o.IsFavorite).String())
	v.Add("isArchived", myBool(o.IsArchived).String())
	v.Add("withoutThumbs", myBool(o.WithoutThumbs).String())
	v.Add("skip", o.Skip)
	return v
}

func (ic *ImmichClient) GetAllAssets(ctx context.Context, opt *GetAssetOptions) ([]*Asset, error) {
	var r []*Asset

	err := ic.newServerCall(ctx, "GetAllAssets").do(get("/asset", setUrlValues(opt.Values()), setAcceptJSON()), responseJSON(&r))
	return r, err

}

type deleteResponse []struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (ic *ImmichClient) DeleteAssets(ctx context.Context, id []string) (*deleteResponse, error) {
	req := struct {
		IDs []string `json:"ids"`
	}{
		IDs: id,
	}

	resp := deleteResponse{}

	err := ic.newServerCall(ctx, "DeleteAsset").do(delete("/asset", setAcceptJSON(), setJSONBody(req)), responseJSON(&resp))
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

func (ic *ImmichClient) GetAssetByID(ctx context.Context, id string) (*Asset, error) {
	r := Asset{}
	err := ic.newServerCall(ctx, "GetAssetByID").do(get("/asset/assetById/"+id, setAcceptJSON()), responseJSON(&r))
	return &r, err
}

func (ic *ImmichClient) UpdateAsset(ctx context.Context, a *Asset) (*Asset, error) {
	r := Asset{}
	err := ic.newServerCall(ctx, "updateAsset").
		do(
			put("/asset/"+a.ID,
				setJSONBody(a),
				setAcceptJSON(),
			),
			responseJSON(&r))
	return &r, err
}
