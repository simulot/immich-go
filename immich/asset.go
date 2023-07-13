package immich

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime/multipart"
	"net/textproto"
	"path/filepath"
	"strings"
	"time"
)

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
	files          []*formFile
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

func (ic *ImmichClient) AssetUpload(fsys fs.FS, file string) (AssetResponse, error) {
	var resp AssetResponse

	asset := Asset{
		DeviceID:   ic.DeviceUUID,
		IsFavorite: false,
		IsReadOnly: false,
	}
	defer asset.Close()

	s, err := fs.Stat(fsys, file)
	if err != nil {
		return resp, LocalFileError(err)
	}

	assetData, err := newFormFile("assetData", fsys, file, s)
	if err != nil {
		return resp, err
	}
	asset.files = append(asset.files, assetData)

	xmp := file + ".xmp"
	s, err = fs.Stat(fsys, xmp)

	if err != nil {
		xmp := file + ".XMP"
		s, err = fs.Stat(fsys, xmp)

	}
	if err == nil {
		sidecar, err := newFormFile("sidecarData", fsys, xmp, s)
		if err == nil {
			asset.files = append(asset.files, sidecar)
		}
	}

	body, w := io.Pipe()
	ctype := asset.WriteBody(w)

	sc := ic.newServerCall("AssetUpload").
		postRequest("/asset/upload", ctype, body).
		setAcceptJSON().
		callServer().
		decodeJSONResponse(&resp)
	return resp, sc.Err()
}

func (a *Asset) WriteBody(w io.WriteCloser) string {
	m := multipart.NewWriter(w)

	go func() {
		var err error

		// Set mime type field with mime of assetData
		for _, f := range a.files {
			if f.field == "assetData" {
				a.FileCreatedAt = f.mod
				a.FileModifiedAt = f.mod
				a.DeviceAssetID = fmt.Sprintf("%s-%d", f.name, f.size)
				a.FileExtension = strings.ToLower(filepath.Ext(f.name))
				a.AssetType = strings.ToUpper(strings.SplitN(f.mime, "/", 2)[0])
			}
		}

		err = errors.Join(err, m.WriteField("deviceAssetId", a.DeviceAssetID))
		err = errors.Join(err, m.WriteField("deviceId", a.DeviceID))
		err = errors.Join(err, m.WriteField("assetType", a.AssetType))
		err = errors.Join(err, m.WriteField("fileCreatedAt", a.FileCreatedAt.Format(time.RFC3339)))
		err = errors.Join(err, m.WriteField("fileModifiedAt", a.FileModifiedAt.Format(time.RFC3339)))
		err = errors.Join(err, m.WriteField("isFavorite", myBool(a.IsFavorite).String()))
		err = errors.Join(err, m.WriteField("fileExtension", a.FileExtension))
		err = errors.Join(err, m.WriteField("duration", formatDuration(a.Duration)))
		err = errors.Join(err, m.WriteField("isReadOnly", myBool(a.IsReadOnly).String()))
		if err != nil {
			return
		}
		for _, f := range a.files {
			h := textproto.MIMEHeader{}
			h.Set("Content-Disposition",
				fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
					escapeQuotes(f.field), escapeQuotes(filepath.Base(f.name))))
			h.Set("Content-Type", f.mime)
			part, err := m.CreatePart(h)
			if err != nil {
				return
			}
			if _, err = io.Copy(part, f.f); err != nil {
				return
			}
		}
	}()
	return "multipart/form-data; boundary=" + m.Boundary()
}

func (a *Asset) Close() error {
	for _, f := range a.files {
		f.Close()
	}
	return nil
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

type formFile struct {
	field   string
	name    string
	mime    string
	size    int64
	mod     time.Time
	headers textproto.MIMEHeader
	io.Reader
	buf *bytes.Buffer
	f   fs.File
}

func newFormFile(field string, fsys fs.FS, path string, info fs.FileInfo) (*formFile, error) {
	f, err := fsys.Open(path)
	if err != nil {
		return nil, err
	}

	ff := formFile{
		field: field,
		f:     f,
		name:  filepath.Base(path),
		size:  info.Size(),
		mod:   info.ModTime(),
	}

	ff.buf = bytes.NewBuffer(nil)
	_, err = io.CopyN(ff.buf, f, 4096)

	if err != nil && err != io.EOF {
		return nil, err
	}

	mtype, err := IsMimeSupported(ff.buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("file %q: %w", path, err)
	}
	ff.mime = mtype
	ff.Reader = io.MultiReader(ff.buf, f)

	return &ff, nil
}

func (ff *formFile) Close() error {
	ff.buf = nil
	return ff.f.Close()
}
