package immich

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/simulot/immich-go/browser"
)

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

func (ic *ImmichClient) AssetUpload(ctx context.Context, la *browser.LocalAssetFile) (AssetResponse, error) {
	var ar AssetResponse
	mtype := ic.TypeFromExt(path.Ext(la.FileName))
	switch mtype {
	case "video", "image":
	default:
		return ar, fmt.Errorf("type file not supported: %s", path.Ext(la.FileName))
	}

	f, err := la.Open()
	if err != nil {
		return ar, (err)
	}

	body, pw := io.Pipe()
	m := multipart.NewWriter(pw)

	go func() {
		defer func() {
			m.Close()
			pw.Close()
		}()
		var s fs.FileInfo
		s, err = f.Stat()
		if err != nil {
			return
		}
		assetType := strings.ToUpper(mtype)
		ext := path.Ext(la.Title)
		if strings.TrimSuffix(la.Title, ext) == "" {
			la.Title = "No Name" + ext // fix #88, #128
		}

		// disguise MP files as MP4
		if strings.ToLower(ext) == ".mp" {
			la.Title = strings.TrimSuffix(la.Title, ext) + ".MP4"
			ext = ".MP4"
		}

		err = m.WriteField("deviceAssetId", fmt.Sprintf("%s-%d", path.Base(la.Title), s.Size()))
		if err != nil {
			return
		}
		err = m.WriteField("deviceId", ic.DeviceUUID)
		if err != nil {
			return
		}
		err = m.WriteField("assetType", assetType)
		if err != nil {
			return
		}
		err = m.WriteField("fileCreatedAt", la.Metadata.DateTaken.Format(time.RFC3339))
		if err != nil {
			return
		}
		err = m.WriteField("fileModifiedAt", s.ModTime().Format(time.RFC3339))
		if err != nil {
			return
		}
		err = m.WriteField("isFavorite", myBool(la.Favorite).String())
		if err != nil {
			return
		}
		err = m.WriteField("fileExtension", ext)
		if err != nil {
			return
		}
		err = m.WriteField("duration", formatDuration(0))
		if err != nil {
			return
		}
		err = m.WriteField("isReadOnly", "false")
		if err != nil {
			return
		}
		if la.LivePhotoID != "" {
			err = m.WriteField("livePhotoVideoId", la.LivePhotoID)
			if err != nil {
				return
			}
		}

		// m.WriteField("isArchived", myBool(la.Archived).String()) // Not supported by the api
		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition",
			fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
				escapeQuotes("assetData"), escapeQuotes(path.Base(la.Title))))
		h.Set("Content-Type", mtype)

		var part io.Writer
		part, err = m.CreatePart(h)
		if err != nil {
			return
		}
		_, err = io.Copy(part, f)
		if err != nil {
			return
		}

		if la.SideCar.IsSet() {
			scName := path.Base(la.FileName) + ".xmp"
			h.Set("Content-Disposition",
				fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
					escapeQuotes("sidecarData"), escapeQuotes(scName)))
			h.Set("Content-Type", "application/xml")

			var part io.Writer
			part, err = m.CreatePart(h)
			if err != nil {
				return
			}
			err = la.SideCar.Write(part)
			if err != nil {
				return
			}
		} else if la.Metadata.IsSet() {
			scName := path.Base(la.FileName) + ".xmp"
			h.Set("Content-Disposition",
				fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
					escapeQuotes("sidecarData"), escapeQuotes(scName)))
			h.Set("Content-Type", "application/xml")

			var part io.Writer
			part, err = m.CreatePart(h)
			if err != nil {
				return
			}
			err = la.Metadata.Write(part)
			if err != nil {
				return
			}
		}
	}()

	errCall := ic.newServerCall(ctx, "AssetUpload").
		do(postRequest("/assets", m.FormDataContentType(), setAcceptJSON(), setBody(body)), responseJSON(&ar))

	err = errors.Join(err, errCall)
	return ar, err
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
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
	req := struct {
		Force bool     `json:"force"`
		IDs   []string `json:"ids"`
	}{
		IDs:   id,
		Force: forceDelete,
	}

	return ic.newServerCall(ctx, "DeleteAsset").do(deleteRequest("/assets", setJSONBody(&req)))
}

func (ic *ImmichClient) GetAssetByID(ctx context.Context, id string) (*Asset, error) {
	body := struct {
		WithExif  bool   `json:"withExif,omitempty"`
		IsVisible bool   `json:"isVisible,omitempty"`
		ID        string `json:"id"`
	}{WithExif: true, IsVisible: true, ID: id}
	r := Asset{}
	err := ic.newServerCall(ctx, "GetAssetByID").do(postRequest("/search/metadata", "application/json", setAcceptJSON(), setJSONBody(body)), responseJSON(&r))
	return &r, err
}

func (ic *ImmichClient) UpdateAssets(ctx context.Context, ids []string,
	isArchived bool, isFavorite bool,
	latitude float64, longitude float64,
	removeParent bool, stackParentID string,
) error {
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

func (ic *ImmichClient) UpdateAsset(ctx context.Context, id string, a *browser.LocalAssetFile) (*Asset, error) {
	type updAsset struct {
		IsArchived  bool    `json:"isArchived"`
		IsFavorite  bool    `json:"isFavorite"`
		Latitude    float64 `json:"latitude,omitempty"`
		Longitude   float64 `json:"longitude,omitempty"`
		Description string  `json:"description,omitempty"`
	}
	param := updAsset{
		IsArchived:  a.Archived,
		IsFavorite:  a.Favorite,
		Description: a.Metadata.Description,
		Latitude:    a.Metadata.Latitude,
		Longitude:   a.Metadata.Longitude,
	}
	r := Asset{}
	err := ic.newServerCall(ctx, "updateAsset").do(putRequest("/assets/"+id, setJSONBody(param)), responseJSON(&r))
	return &r, err
}

func (ic *ImmichClient) StackAssets(ctx context.Context, coverID string, ids []string) error {
	cover, err := ic.GetAssetByID(ctx, coverID)
	if err != nil {
		return err
	}

	return ic.UpdateAssets(ctx, ids, cover.IsArchived, cover.IsFavorite, cover.ExifInfo.Latitude, cover.ExifInfo.Longitude, false, coverID)
}
