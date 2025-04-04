package immichfs

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/simulot/immich-go/immich"
)

func newTestImmichServer(_ *testing.T) *immich.ImmichClient { //nolint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/users/me":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"id":"1","email":"test@email.com"}`)) // nolint
		case "/api/server/media-types":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"image":[".jpg",".png"],"video":[".mp4"]}`)) // nolint
		case "/api/server/ping":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"res":"pong"}`)) // nolint

		case "/api/assets/test-asset-id":
			w.WriteHeader(http.StatusOK)
			// w.Write([]byte(`{"id":"test-asset-id","name":"test-asset","type":"image","size":1024}`)) // nolint
			w.Write([]byte(asssetinfo)) // nolint

		case "/api/assets/test-asset-id/original":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`original asset content`)) // nolint
		}
	}))
	client, _ := immich.NewImmichClient(server.URL, "test-key")
	return client
}

func TestImmichfs(t *testing.T) {
	ctx := context.Background()
	client := newTestImmichServer(t)
	ifs := NewImmichFS(ctx, "testclient", client)

	file, err := ifs.Open("test-asset-id")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	buf := bytes.NewBuffer(nil)
	_, err = io.Copy(buf, file)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if buf.String() != "original asset content" {
		t.Fatalf("expected 'original asset content', got %v", buf.String())
	}
}

var asssetinfo = `{
    "id": "test-asset-id",
    "deviceAssetId": "2cdcf6af-d13c-4080-a59e-353818a8cf3a.jpg-120645",
    "ownerId": "df17ccde-c94a-4b48-bb51-cf977115a722",
    "owner": {
        "id": "df17ccde-c94a-4b48-bb51-cf977115a722",
        "email": "demo@immich.app",
        "name": "demo",
        "profileImagePath": "",
        "avatarColor": "yellow",
        "profileChangedAt": "2024-10-15T19:33:53.081Z"
    },
    "deviceId": "gl65",
    "libraryId": null,
    "type": "IMAGE",
    "originalPath": "upload/upload/df17ccde-c94a-4b48-bb51-cf977115a722/b8/b1/b8b1cbe1-bd97-4725-962c-62f7cf68b427.jpg",
    "originalFileName": "test asset.jpg",
    "originalMimeType": "image/jpeg",
    "thumbhash": "nPgNDYR5aIiDd4iAh4iHi/yNhwgn",
    "fileCreatedAt": "2024-07-07T13:31:46.000Z",
    "fileModifiedAt": "2024-07-07T13:31:46.000Z",
    "localDateTime": "2024-07-07T13:31:46.000Z",
    "updatedAt": "2024-11-07T18:57:19.277Z",
    "isFavorite": false,
    "isArchived": false,
    "isTrashed": false,
    "duration": "0:00:00.00000",
    "exifInfo": {
        "make": null,
        "model": null,
        "exifImageWidth": 1600,
        "exifImageHeight": 1200,
        "fileSizeInByte": 120645,
        "orientation": null,
        "dateTimeOriginal": "2024-07-07T13:31:46.000Z",
        "modifyDate": "2024-07-07T13:31:46.000Z",
        "timeZone": null,
        "lensModel": null,
        "fNumber": null,
        "focalLength": null,
        "iso": null,
        "exposureTime": null,
        "latitude": null,
        "longitude": null,
        "city": null,
        "state": null,
        "country": null,
        "description": "",
        "projectionType": null,
        "rating": null
    },
    "livePhotoVideoId": null,
    "tags": [
        {
            "id": "a694d813-a787-4694-b012-60896e5cb5fd",
            "parentId": "85cdba4b-3614-4fac-bf32-d5276de6ab72",
            "name": "outdoors",
            "value": "activities/outdoors",
            "createdAt": "2024-11-07T18:57:08.281Z",
            "updatedAt": "2024-11-08T18:11:28.816Z"
        }
    ],
    "people": [
        {
            "id": "9d9ca38f-48e1-4e46-9ddf-a12a500a7284",
            "name": "",
            "birthDate": null,
            "thumbnailPath": "upload/thumbs/df17ccde-c94a-4b48-bb51-cf977115a722/9d/9c/9d9ca38f-48e1-4e46-9ddf-a12a500a7284.jpeg",
            "isHidden": false,
            "updatedAt": "2024-11-07T18:21:26.716Z",
            "faces": [
                {
                    "id": "fe231f45-0069-4ab8-8653-f2b20075957b",
                    "imageHeight": 1200,
                    "imageWidth": 1600,
                    "boundingBoxX1": 822,
                    "boundingBoxX2": 1043,
                    "boundingBoxY1": 240,
                    "boundingBoxY2": 548,
                    "sourceType": "machine-learning"
                }
            ]
        }
    ],
    "unassignedFaces": [],
    "checksum": "3A/J/6HAHpyOpSQirnn/9NSpWwA=",
    "stack": null,
    "isOffline": false,
    "hasMetadata": true,
    "duplicateId": null,
    "resized": true
}`
