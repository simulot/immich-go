package immich_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/filetypes"
)

/*
baseline

goos: linux
goarch: amd64
pkg: github.com/simulot/immich-go/immich
cpu: Intel(R) Core(TM) i7-10750H CPU @ 2.60GHz
Benchmark_IsExtensionPrefix-12    	 4096238	       297.3 ns/op	       3 B/op	       1 allocs/op
PASS
ok  	github.com/simulot/immich-go/immich	1.518s

goos: linux
goarch: amd64
pkg: github.com/simulot/immich-go/immich
cpu: Intel(R) Core(TM) i7-10750H CPU @ 2.60GHz
Benchmark_IsExtensionPrefix-12    	16536936	        72.85 ns/op	       3 B/op	       1 allocs/op
PASS
ok  	github.com/simulot/immich-go/immich	1.283s
*/
func Benchmark_IsExtensionPrefix(b *testing.B) {
	sm := filetypes.DefaultSupportedMedia
	sm.IsExtensionPrefix(".JP")
	for i := 0; i < b.N; i++ {
		sm.IsExtensionPrefix(".JP")
	}
}

func TestPingServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"res":"pong"}`))
	}))
	defer server.Close()

	client, _ := immich.NewImmichClient(server.URL, "test-key")
	err := client.PingServer(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestValidateConnection(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/users/me" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"id":"1","email":"test@example.com"}`))
		} else if r.URL.Path == "/api/server/media-types" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"image":[".jpg",".png"],"video":[".mp4"]}`))
		}
	}))
	defer server.Close()

	client, _ := immich.NewImmichClient(server.URL, "test-key")
	user, err := client.ValidateConnection(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if user.ID != "1" {
		t.Errorf("expected user ID to be '1', got %v", user.ID)
	}
	if user.Email != "test@example.com" {
		t.Errorf("expected user email to be 'test@example.com', got %v", user.Email)
	}
}

func TestGetServerStatistics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"photos":100,"videos":50,"usage":1024}`))
	}))
	defer server.Close()

	client, _ := immich.NewImmichClient(server.URL, "test-key")
	stats, err := client.GetServerStatistics(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stats.Photos != 100 {
		t.Errorf("expected photos to be 100, got %v", stats.Photos)
	}
	if stats.Videos != 50 {
		t.Errorf("expected videos to be 50, got %v", stats.Videos)
	}
	if stats.Usage != 1024 {
		t.Errorf("expected usage to be 1024, got %v", stats.Usage)
	}
}

func TestGetAssetStatistics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"images":200,"videos":100,"total":300}`))
	}))
	defer server.Close()

	client, _ := immich.NewImmichClient(server.URL, "test-key")
	stats, err := client.GetAssetStatistics(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if stats.Images != 200 {
		t.Errorf("expected images to be 200, got %v", stats.Images)
	}
	if stats.Videos != 100 {
		t.Errorf("expected videos to be 100, got %v", stats.Videos)
	}
	if stats.Total != 300 {
		t.Errorf("expected total to be 300, got %v", stats.Total)
	}
}

func TestGetSupportedMediaTypes(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"image":[".jpg",".png"],"video":[".mp4"]}`))
	}))
	defer server.Close()

	client, _ := immich.NewImmichClient(server.URL, "test-key")
	mediaTypes, err := client.GetSupportedMediaTypes(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if mediaTypes[".jpg"] != filetypes.TypeImage {
		t.Errorf("expected .jpg to be %v, got %v", filetypes.TypeImage, mediaTypes[".jpg"])
	}
	if mediaTypes[".png"] != filetypes.TypeImage {
		t.Errorf("expected .png to be %v, got %v", filetypes.TypeImage, mediaTypes[".png"])
	}
	if mediaTypes[".mp4"] != filetypes.TypeVideo {
		t.Errorf("expected .mp4 to be %v, got %v", filetypes.TypeVideo, mediaTypes[".mp4"])
	}
}

func TestDownloadAsset(t *testing.T) {
	expectedContent := "dummy content"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(expectedContent))
	}))
	defer server.Close()

	client, _ := immich.NewImmichClient(server.URL, "test-key")
	rc, err := client.DownloadAsset(context.Background(), "test-asset-id")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	defer rc.Close()

	content, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("expected no error reading content, got %v", err)
	}
	if string(content) != expectedContent {
		t.Errorf("expected content to be %v, got %v", expectedContent, string(content))
	}
}
