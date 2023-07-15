package immich

import (
	"bytes"
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
	var ar AssetResponse

	// Check the mime type with the first 4k of the file
	b4k := bytes.NewBuffer(nil)
	f, err := fsys.Open(file)
	if err != nil {
		return ar, LocalFileError(err)
	}

	_, err = io.CopyN(b4k, f, 4096)
	if err != nil && err != io.EOF {
		return ar, LocalFileError(err)
	}

	mtype, err := IsMimeSupported(b4k.Bytes())
	if err != nil {
		return ar, err
	}

	body, pw := io.Pipe()
	m := multipart.NewWriter(pw)

	go func() {
		defer func() {
			f.Close()
			m.Close()
			pw.Close()
		}()
		s, err := f.Stat()
		if err != nil {
			return
		}
		assetType := strings.ToUpper(strings.Split(mtype, "/")[0])

		m.WriteField("deviceAssetId", fmt.Sprintf("%s-%d", filepath.Base(file), s.Size()))
		m.WriteField("deviceId", ic.DeviceUUID)
		m.WriteField("assetType", assetType)
		m.WriteField("fileCreatedAt", s.ModTime().Format(time.RFC3339))
		m.WriteField("fileModifiedAt", s.ModTime().Format(time.RFC3339))
		m.WriteField("isFavorite", "false")
		m.WriteField("fileExtension", filepath.Ext(file))
		m.WriteField("duration", formatDuration(0))
		m.WriteField("isReadOnly", "false")
		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition",
			fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
				escapeQuotes("assetData"), escapeQuotes(filepath.Base(file))))
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

/*
func (a *Asset) Read(b []byte) (int, error) {
	return a.r.Read(b)

}

func (a *Asset) makeBody() (string, io.ReadCloser, error) {
	r, w := io.Pipe()
	m := multipart.NewWriter(w)

	go func() {
		var err error
		defer func() {
			fmt.Println("End of makeBody,", err)
		}()

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
			var n int64
			fmt.Println("File:", f.name, "Size:", f.size, "Written:", n)
			h := textproto.MIMEHeader{}
			h.Set("Content-Disposition",
				fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
					escapeQuotes(f.field), escapeQuotes(filepath.Base(f.name))))
			h.Set("Content-Type", f.mime)
			part, err := m.CreatePart(h)
			if err != nil {
				return
			}
			n, err = io.Copy(part, f.f)
			if err != nil {
				return
			}
			fmt.Println("File:", f.name, "Size:", f.size, "Written:", n)
			err = f.Close()
			if err != nil {
				return
			}
		}

	}()
	return m.FormDataContentType(), r, nil
}

func (a *Asset) Close() error {
	for _, f := range a.files {
		f.Close()
	}
	return a.r.Close()
}


type formFile struct {
	field string
	name  string
	mime  string
	size  int64
	mod   time.Time
	io.Reader
	// bytes4K []byte
	// buf *bytes.Buffer
	f fs.File
}

// var bytes4K = sync.Pool{
// 	New: func() any {
// 		return make([]byte, 0, 4096)
// 	},
// }

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

	// ff.bytes4K = bytes4K.Get().([]byte)

	// ff.buf = bytes.NewBuffer(ff.bytes4K)
	buf := bytes.NewBuffer(nil)
	_, err = io.CopyN(buf, f, 4096)

	if err != nil && err != io.EOF {
		return nil, err
	}

	mtype, err := IsMimeSupported(buf.Bytes())
	if err != nil {
		return nil, err
	}
	ff.mime = mtype
	go func(){
		ff.Reader = io.MultiReader(buf, f)

	}

	return &ff, nil
}

func (ff *formFile) Close() error {
	// bytes4K.Put(ff.bytes4K)
	// ff.buf = nil
	fmt.Println("File:", ff.name, " is closed")
	return ff.f.Close()
}
*/
