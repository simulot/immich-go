package immich

import (
	"context"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"testing"
	"testing/fstest"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/stretchr/testify/require"
)

func TestAssetUploadCorrectsJPEGWithHEICSuffix(t *testing.T) {
	t.Parallel()

	jpeg := []byte{0xff, 0xd8, 0xff, 0xe0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 0xff, 0xd9}
	type uploadRequest struct {
		name   string
		data   []byte
		fields map[string]string
		err    error
	}
	received := make(chan uploadRequest, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		result := uploadRequest{fields: make(map[string]string)}
		defer func() { received <- result }()
		if r.URL.Path != "/api/assets" {
			result.err = fmt.Errorf("unexpected request path %q", r.URL.Path)
			http.Error(w, result.err.Error(), http.StatusNotFound)
			return
		}

		reader, err := r.MultipartReader()
		if err != nil {
			result.err = err
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		for {
			part, err := reader.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				result.err = err
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			value, err := io.ReadAll(part)
			if err != nil {
				result.err = err
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if part.FormName() == "assetData" {
				result.name = part.FileName()
				result.data = value
			} else {
				result.fields[part.FormName()] = string(value)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"id":"asset-id","status":"created"}`)
	}))
	t.Cleanup(server.Close)

	client, err := NewImmichClient(server.URL, "test-key")
	require.NoError(t, err)
	client.supportedMediaTypes = filetypes.DefaultSupportedMedia

	fileSystem := fstest.MapFS{
		"photo.HEIC": &fstest.MapFile{Data: jpeg, Mode: fs.ModePerm},
	}
	asset := &assets.Asset{
		File:             fshelper.FSName(fileSystem, "photo.HEIC"),
		FileSize:         len(jpeg),
		OriginalFileName: "photo.HEIC",
	}

	_, err = client.AssetUpload(context.Background(), asset)
	require.NoError(t, err)
	uploaded := <-received
	require.NoError(t, uploaded.err)
	require.Equal(t, "photo.jpg", uploaded.name)
	require.Equal(t, ".jpg", uploaded.fields["fileExtension"])
	require.Equal(t, "photo.jpg-13", uploaded.fields["deviceAssetId"])
	require.Equal(t, jpeg, uploaded.data)
}
