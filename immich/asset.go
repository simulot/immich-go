package immich

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/simulot/immich-go/internal/assets"
)

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
	TimeFormat = "2006-01-02T15:04:05Z"
)

func (ic *ImmichClient) AssetUpload(ctx context.Context, la *assets.Asset) (AssetResponse, error) {
	if ic.dryRun {
		return AssetResponse{
			ID:     uuid.NewString(),
			Status: UploadCreated,
		}, nil
	}
	var ar AssetResponse
	ext := path.Ext(la.OriginalFileName)
	if strings.TrimSuffix(la.OriginalFileName, ext) == "" {
		la.OriginalFileName = "No Name" + ext // fix #88, #128
	}

	if strings.ToUpper(ext) == ".MP" {
		ext = ".MP4" // #405
		la.OriginalFileName = la.OriginalFileName + ".MP4"
	}
	mtype := ic.TypeFromExt(ext)
	switch mtype {
	case "video", "image":
	default:
		return ar, fmt.Errorf("type file not supported: %s", path.Ext(la.OriginalFileName))
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

		err = m.WriteField("deviceAssetId", fmt.Sprintf("%s-%d", path.Base(la.OriginalFileName), s.Size()))
		if err != nil {
			return
		}
		err = m.WriteField("deviceId", ic.DeviceUUID)
		if err != nil {
			return
		}
		err = m.WriteField("assetType", mtype)
		if err != nil {
			return
		}

		if !la.CaptureDate.IsZero() {
			err = m.WriteField("fileCreatedAt", la.CaptureDate.Format(TimeFormat))
		} else {
			err = m.WriteField("fileCreatedAt", s.ModTime().Format(TimeFormat))
		}
		if err != nil {
			return
		}
		err = m.WriteField("fileModifiedAt", s.ModTime().Format(TimeFormat))
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
		err := m.WriteField("isArchived", myBool(la.Archived).String())
		if err != nil {
			return
		}

		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition",
			fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
				escapeQuotes("assetData"), escapeQuotes(path.Base(la.OriginalFileName))))
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

		if la.FromSideCar != nil && strings.ToLower(la.FromSideCar.File.Name()) == ".xmp" {
			scName := path.Base(la.OriginalFileName) + ".xmp"
			h.Set("Content-Disposition",
				fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
					escapeQuotes("sidecarData"), escapeQuotes(scName)))
			h.Set("Content-Type", "application/xml")

			var part io.Writer
			part, err = m.CreatePart(h)
			if err != nil {
				return
			}
			defer f.Close()
			f, err = la.FromSideCar.File.Open()
			if err != nil {
				return
			}

			_, err = io.Copy(part, f)
			if err != nil {
				return
			}
		}
	}()

	var callValues map[string]string
	if ic.apiTraceWriter != nil {
		callValues = map[string]string{
			ctxAssetName: la.File.Name(),
		}
		if la.FromSideCar != nil {
			callValues[ctxSideCarName] = la.FromSideCar.File.Name()
		}
	}

	errCall := ic.newServerCall(ctx, "AssetUpload").
		do(postRequest("/assets", m.FormDataContentType(), setContextValue(callValues), setAcceptJSON(), setBody(body)), responseJSON(&ar))

	err = errors.Join(err, errCall)
	return ar, err
}

const (
	ctxCallValues    = "call-values"
	ctxAssetName     = "asset file name"
	ctxSideCarName   = "side car file name"
	ctxLiveVideoName = "live video name"
)

func setContextValue(kv map[string]string) serverRequestOption {
	return func(sc *serverCall, req *http.Request) error {
		if sc.err != nil || kv == nil {
			return nil
		}
		sc.ctx = context.WithValue(sc.ctx, ctxCallValues, kv)
		return nil
	}
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
	IsArchived       bool      `json:"isArchived"`
	IsFavorite       bool      `json:"isFavorite"`
	Latitude         float64   `json:"latitude,omitempty"`
	Longitude        float64   `json:"longitude,omitempty"`
	Description      string    `json:"description,omitempty"`
	Rating           int       `json:"rating,omitempty"`
	LivePhotoVideoID string    `json:"livePhotoVideoId,omitempty"`
	DateTimeOriginal time.Time `json:"dateTimeOriginal,omitempty"`
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
