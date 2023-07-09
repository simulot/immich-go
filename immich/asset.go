package immich

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/textproto"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gabriel-vasile/mimetype"
)

type UnsupportedMedia error
type LocalFileError error

type Asset struct {
	DeviceAssetID  string        `json:"deviceAssetId"`
	DeviceID       string        `json:"deviceId"`
	AssetType      string        `json:"assetType"`
	FileCreatedAt  time.Time     `json:"fileCreatedAt"`
	FileModifiedAt time.Time     `json:"fileModifiedAt"`
	IsFavorite     bool          `json:"isFavorite"`
	FileExtension  string        `json:"fileExtension"`
	Duration       time.Duration `json:"duration"`
	IsReadOnly     bool          `json:"isReadOnly"`
	files          []*FormFile
}

type FormFile struct {
	field   string
	info    fs.FileInfo
	mime    string
	headers textproto.MIMEHeader
	r       io.Reader
	f       io.Closer
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

func (ic *ImmichClient) AssetUpload(file string) (AssetResponse, error) {
	var resp AssetResponse

	asset := Asset{
		DeviceID:   ic.DeviceUUID,
		IsFavorite: false,
		IsReadOnly: false,
	}
	defer asset.Close()

	s, err := os.Stat(file)
	if err != nil {
		return resp, LocalFileError(err)
	}

	assetData, err := newFormFile("assetData", file, s)
	if err != nil {
		return resp, err
	}
	asset.files = append(asset.files, assetData)

	xmp := file + ".xmp"
	s, err = os.Stat(xmp)

	if err != nil {
		xmp := file + ".XMP"
		s, err = os.Stat(xmp)

	}
	if err == nil {
		sidecar, err := newFormFile("sidecarData", xmp, s)
		if err == nil {
			asset.files = append(asset.files, sidecar)
		}
	}

	// set asset values based on the
	for _, f := range asset.files {
		if f.field == "assetData" {
			asset.FileCreatedAt = f.info.ModTime()
			asset.FileModifiedAt = f.info.ModTime()
			asset.DeviceAssetID = fmt.Sprintf("%s-%d", filepath.Base(f.info.Name()), f.info.Size())
			asset.FileExtension = strings.ToLower(filepath.Ext(f.info.Name()))
			asset.AssetType = strings.ToUpper(strings.SplitN(f.mime, "/", 2)[0])
		}
	}

	sc := ic.newServerCall("AssetUpload").postFormRequest("/asset/upload", asset).callServer().decodeJSONResponse(&resp)
	return resp, sc.Err()
}

func (a Asset) WriteMultiPart(w *multipart.Writer) error {
	var err error
	err = errors.Join(err, w.WriteField("deviceAssetId", a.DeviceAssetID))
	err = errors.Join(err, w.WriteField("deviceId", a.DeviceID))
	err = errors.Join(err, w.WriteField("assetType", a.AssetType))
	err = errors.Join(err, w.WriteField("fileCreatedAt", a.FileCreatedAt.Format(time.RFC3339)))
	err = errors.Join(err, w.WriteField("fileModifiedAt", a.FileModifiedAt.Format(time.RFC3339)))
	err = errors.Join(err, w.WriteField("isFavorite", myBool(a.IsFavorite).String()))
	err = errors.Join(err, w.WriteField("fileExtension", a.FileExtension))
	err = errors.Join(err, w.WriteField("duration", formatDuration(a.Duration)))
	err = errors.Join(err, w.WriteField("isReadOnly", myBool(a.IsReadOnly).String()))

	// Add files content
	for _, f := range a.files {
		fw, err := w.CreatePart(f.headers)
		if err != nil {
			return err
		}
		_, err = io.Copy(fw, f.r)
		if err != nil {
			return err
		}
	}
	return err
}

func (a Asset) Close() error {
	var err error
	for _, f := range a.files {
		err = errors.Join(f.f.Close())
	}
	return err
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

func newFormFile(field string, path string, info fs.FileInfo) (*FormFile, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	buf := bytes.NewBuffer([]byte{})
	_, err = io.CopyN(buf, f, 64*1024)
	if err != nil && err != io.EOF {
		return nil, err
	}

	mtype := mimetype.Detect(buf.Bytes())
	_, err = IsMimeSupported(mtype.String())
	if err != nil {
		return nil, UnsupportedMedia(fmt.Errorf("file type not supported: %s", filepath.Base(path)))
	}

	h := textproto.MIMEHeader{}
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			escapeQuotes(field), escapeQuotes(filepath.Base(path))))
	h.Set("Content-Type", mtype.String())

	ff := FormFile{
		field:   field,
		headers: h,
		info:    info,
		mime:    mtype.String(),
		r:       io.MultiReader(buf, f),
		f:       f,
	}

	return &ff, nil
}
