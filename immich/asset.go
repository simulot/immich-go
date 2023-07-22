package immich

import (
	"bytes"
	"fmt"
	"immich-go/immich/assets"
	"io"
	"math"
	"mime/multipart"
	"net/textproto"
	"net/url"
	"path"
	"path/filepath"
	"strings"
	"time"
)

// immich Asset simplified
type Asset struct {
	ID            string `json:"id"`
	DeviceAssetID string `json:"deviceAssetId"`
	// OwnerID          string `json:"ownerId"`
	DeviceID         string `json:"deviceId"`
	Type             string `json:"type"`
	OriginalPath     string `json:"originalPath"`
	OriginalFileName string `json:"originalFileName"`
	// Resized          bool      `json:"resized"`
	// Thumbhash        string    `json:"thumbhash"`
	FileCreatedAt time.Time `json:"fileCreatedAt"`
	// FileModifiedAt time.Time `json:"fileModifiedAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	// IsFavorite     bool      `json:"isFavorite"`
	// IsArchived     bool      `json:"isArchived"`
	// Duration       string    `json:"duration"`
	ExifInfo struct {
		// 	Make             string    `json:"make"`
		// 	Model            string    `json:"model"`
		ExifImageWidth  int `json:"exifImageWidth"`
		ExifImageHeight int `json:"exifImageHeight"`
		FileSizeInByte  int `json:"fileSizeInByte"`
		// 	Orientation      string    `json:"orientation"`
		DateTimeOriginal time.Time `json:"dateTimeOriginal"`
		// 	ModifyDate       time.Time `json:"modifyDate"`
		// 	TimeZone         string    `json:"timeZone"`
		// 	LensModel        string    `json:"lensModel"`
		// 	FNumber          float64   `json:"fNumber"`
		// 	FocalLength      float64   `json:"focalLength"`
		// 	Iso              int       `json:"iso"`
		// 	ExposureTime     string    `json:"exposureTime"`
		// 	Latitude         float64   `json:"latitude"`
		// 	Longitude        float64   `json:"longitude"`
		// 	City             string    `json:"city"`
		// 	State            string    `json:"state"`
		// 	Country          string    `json:"country"`
		// 	Description      string    `json:"description"`
	} `json:"exifInfo"`
	// LivePhotoVideoID any    `json:"livePhotoVideoId"`
	// Tags             []any  `json:"tags"`
	Checksum string `json:"checksum"`
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

func (ic *ImmichClient) AssetUpload(la *assets.LocalAssetFile) (AssetResponse, error) {
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
			// f.Close()
			m.Close()
			pw.Close()
		}()
		s, err := f.Stat()
		if err != nil {
			return
		}
		assetType := strings.ToUpper(strings.Split(mtype, "/")[0])

		m.WriteField("deviceAssetId", fmt.Sprintf("%s-%d", path.Base(la.FileName), s.Size()))
		m.WriteField("deviceId", ic.DeviceUUID)
		m.WriteField("assetType", assetType)
		m.WriteField("fileCreatedAt", s.ModTime().Format(time.RFC3339))
		m.WriteField("fileModifiedAt", s.ModTime().Format(time.RFC3339))
		m.WriteField("isFavorite", "false")
		m.WriteField("fileExtension", path.Ext(la.FileName))
		m.WriteField("duration", formatDuration(0))
		m.WriteField("isReadOnly", "false")
		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition",
			fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
				escapeQuotes("assetData"), escapeQuotes(path.Base(la.FileName))))
		h.Set("Content-Type", mtype)

		part, err := m.CreatePart(h)
		if err != nil {
			return
		}
		_, err = io.Copy(part, io.MultiReader(b4k, f))
		if err != nil {
			return
		}
	}()

	err = ic.newServerCall("AssetUpload").
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

func (ic *ImmichClient) GetAllAssets(opt *GetAssetOptions) (*AssetIndex, error) {
	r := AssetIndex{}
	err := ic.newServerCall("GetAllAssets").do(get("/asset", setUrlValues(opt.Values()), setAcceptJSON()), responseJSON(&r.assets))
	if err != nil {
		return nil, err
	}
	r.ReIndex()
	return &r, nil

}

type AssetIndex struct {
	assets []*Asset
	byHash map[string][]*Asset
	byName map[string][]*Asset
	byID   map[string]*Asset
}

func (ai *AssetIndex) ReIndex() {
	ai.byHash = map[string][]*Asset{}
	ai.byName = map[string][]*Asset{}
	ai.byID = map[string]*Asset{}

	for _, a := range ai.assets {
		l := ai.byHash[a.Checksum]
		l = append(l, a)
		ai.byHash[a.Checksum] = l

		n := a.OriginalFileName
		l = ai.byName[n]
		l = append(l, a)
		ai.byName[n] = l
		ai.byID[a.DeviceAssetID] = a
	}
}

func (ai *AssetIndex) Len() int {
	return len(ai.assets)
}

func (ai *AssetIndex) AddLocalAsset(la *assets.LocalAssetFile) {
	sa := &Asset{
		ID:               fmt.Sprintf("%s-%s", path.Base(la.FileName), la.Size()),
		OriginalFileName: path.Base(la.FileName),
	}
	ai.assets = append(ai.assets, sa)
	ai.byID[sa.ID] = sa
	l := ai.byName[sa.OriginalFileName]
	l = append(l, sa)
	ai.byName[sa.OriginalFileName] = l
}

type deleteResponse []struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}

func (ic *ImmichClient) DeleteAsset(id []string) (*deleteResponse, error) {
	req := struct {
		IDs []string `json:"ids"`
	}{
		IDs: id,
	}

	resp := deleteResponse{}

	err := ic.newServerCall("DeleteAsset").do(delete("/asset", setAcceptJSON(), setJSONBody(req)), responseJSON(&resp))
	if err != nil {
		return nil, err
	}
	return &resp, nil
}

// - - go:generate stringer -type=AdviceCode
type AdviceCode int

func (a AdviceCode) String() string {
	switch a {
	case IDontKnow:
		return "IDontKnow"
	// case SameNameOnServerButNotSure:
	// 	return "SameNameOnServerButNotSure"
	case SmallerOnServer:
		return "SmallerOnServer"
	case BetterOnServer:
		return "BetterOnServer"
	case SameOnServer:
		return "SameOnServer"
	case NotOnServer:
		return "NotOnServer"
	}
	return fmt.Sprintf("advice(%d)", a)
}

const (
	IDontKnow AdviceCode = iota
	// SameNameOnServerButNotSure
	SmallerOnServer
	BetterOnServer
	SameOnServer
	NotOnServer
)

type Advice struct {
	Advice      AdviceCode
	Message     string
	ServerAsset *Asset
	LocalAsset  *assets.LocalAssetFile
}

func formatBytes(s int) string {
	suffixes := []string{"B", "KB", "MB", "GB"}
	bytes := float64(s)
	base := 1024.0
	if bytes < base {
		return fmt.Sprintf("%.0f %s", bytes, suffixes[0])
	}
	exp := int64(0)
	for bytes >= base && exp < int64(len(suffixes)-1) {
		bytes /= base
		exp++
	}
	roundedSize := math.Round(bytes*10) / 10
	return fmt.Sprintf("%.1f %s", roundedSize, suffixes[exp])
}

func adviceIDontKnow(la *assets.LocalAssetFile) *Advice {
	return &Advice{
		Advice:     IDontKnow,
		Message:    fmt.Sprintf("Can't decide what to do with %q. Check this vile yourself", la.FileName),
		LocalAsset: la,
	}
}

func adviceSameOnServer(sa *Asset) *Advice {
	return &Advice{
		Advice:      SameOnServer,
		Message:     fmt.Sprintf("An asset with the same name:%q, date:%q and size:%s exists on the server. No need to upload.", sa.OriginalFileName, sa.ExifInfo.DateTimeOriginal.Format(time.DateTime), formatBytes(sa.ExifInfo.FileSizeInByte)),
		ServerAsset: sa,
	}
}
func adviceSmallerOnServer(sa *Asset) *Advice {
	return &Advice{
		Advice:      SmallerOnServer,
		Message:     fmt.Sprintf("An asset with the same name:%q and date:%q but with smaller size:%s exists on the server. Replace it.", sa.OriginalFileName, sa.ExifInfo.DateTimeOriginal.Format(time.DateTime), formatBytes(sa.ExifInfo.FileSizeInByte)),
		ServerAsset: sa,
	}
}
func adviceBetterOnServer(sa *Asset) *Advice {
	return &Advice{
		Advice:      BetterOnServer,
		Message:     fmt.Sprintf("An asset with the same name:%q and date:%q but with bigger size:%s exists on the server. No need to upload.", sa.OriginalFileName, sa.ExifInfo.DateTimeOriginal.Format(time.DateTime), formatBytes(sa.ExifInfo.FileSizeInByte)),
		ServerAsset: sa,
	}
}
func adviceNotOnServer(sa *Asset) *Advice {
	return &Advice{
		Advice:      NotOnServer,
		Message:     "This a new asset, upload it.",
		ServerAsset: sa,
	}
}

// ShouldUpload check if the server has this asset
//
// The server may have different assets with the same name. This happens with photos produced by digital cameras.
// The server may have the asset, but in lower resolution. Compare the taken date and resolution
//
//

func (ai *AssetIndex) ShouldUpload(la *assets.LocalAssetFile) (*Advice, error) {

	ID := fmt.Sprintf("%s-%d", path.Base(la.FileName), la.Size())

	sa := ai.byID[ID]
	if sa != nil {
		// the same ID exist on the server
		return adviceSameOnServer(sa), nil
	}

	var l []*Asset
	var n string

	// check all files with the same name

	// n = strings.TrimSuffix(path.Base(la.Name), path.Ext(la.Name))
	n = filepath.Base(path.Base(la.FileName))
	l = ai.byName[n]
	if len(l) == 0 {
		n = strings.TrimSuffix(n, filepath.Ext(n))
		l = ai.byName[n]
	}

	if len(l) > 0 {
		dateTaken, err := la.DateTaken()
		size := int(la.Size())
		if err != nil {
			return adviceIDontKnow(la), nil

		}
		for _, sa = range l {
			compareDate := dateTaken.Compare(sa.ExifInfo.DateTimeOriginal)
			compareSize := size - sa.ExifInfo.FileSizeInByte

			switch {
			case compareDate == 0 && compareSize == 0:
				return adviceSameOnServer(sa), nil
			case compareDate == 0 && compareSize > 0:
				return adviceSmallerOnServer(sa), nil
			case compareDate == 0 && compareSize < 0:
				return adviceBetterOnServer(sa), nil
			}
		}
	}
	return adviceNotOnServer(nil), nil
}
