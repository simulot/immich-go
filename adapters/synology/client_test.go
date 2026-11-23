package synology

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

// newTestClient returns a client pointing at srv with a fake session and a
// logger that discards everything.
func newTestClient(t *testing.T, srv *httptest.Server) *Client {
	t.Helper()
	c, err := NewClient(srv.URL, "user", "pass",
		WithLogger(slog.New(slog.NewTextHandler(io.Discard, nil))))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	c.sid = "test-session"
	return c
}

func TestListTeamItemsUsesTeamAPI(t *testing.T) {
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		got = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"success":true,"data":{"list":[{"id":7,"filename":"photo.jpg","type":"photo"}]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	items, err := c.ListTeamItems(context.Background(), 0, 10, []string{"tag"})
	if err != nil {
		t.Fatalf("ListTeamItems: %v", err)
	}
	if got.Get(paramAPI) != apiTeamBrowseItem {
		t.Errorf("api = %q, want %q", got.Get(paramAPI), apiTeamBrowseItem)
	}
	if got.Get(paramVersion) != "1" {
		t.Errorf("version = %q, want 1", got.Get(paramVersion))
	}
	if got.Get(paramMethod) != methodList {
		t.Errorf("method = %q, want %q", got.Get(paramMethod), methodList)
	}
	if len(items) != 1 || items[0].ID != 7 {
		t.Errorf("unexpected items: %+v", items)
	}
}

func TestListItemsKeepsPersonalAPI(t *testing.T) {
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		got = r.PostForm
		w.Header().Set("Content-Type", "application/json")
		_, _ = io.WriteString(w, `{"success":true,"data":{"list":[]}}`)
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	if _, err := c.ListAllItems(context.Background(), 0, 10, nil); err != nil {
		t.Fatalf("ListAllItems: %v", err)
	}
	if got.Get(paramAPI) != apiBrowseItem {
		t.Errorf("api = %q, want %q", got.Get(paramAPI), apiBrowseItem)
	}
}

func TestDownloadTeamItemUsesTeamAPI(t *testing.T) {
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		got = r.PostForm
		w.Header().Set("Content-Type", "application/octet-stream")
		_, _ = io.WriteString(w, "file-content")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	rc, err := c.DownloadTeamItem(context.Background(), 42, "42_1")
	if err != nil {
		t.Fatalf("DownloadTeamItem: %v", err)
	}
	defer rc.Close()

	body, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(body) != "file-content" {
		t.Errorf("body = %q", body)
	}
	if got.Get(paramAPI) != apiTeamDownload {
		t.Errorf("api = %q, want %q", got.Get(paramAPI), apiTeamDownload)
	}
}

func TestDownloadLivePhotoUsesTeamAPI(t *testing.T) {
	var got url.Values
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.ParseForm()
		got = r.PostForm
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="download.zip"`)
		_, _ = io.WriteString(w, "zip-content")
	}))
	defer srv.Close()

	c := newTestClient(t, srv)
	rc, _, isZip, err := c.DownloadTeamLivePhoto(context.Background(), 42, "IMG_1.HEIC")
	if err != nil {
		t.Fatalf("DownloadTeamLivePhoto: %v", err)
	}
	defer rc.Close()

	if !isZip {
		t.Error("expected ZIP response to be detected")
	}
	if got.Get(paramAPI) != apiTeamDownload {
		t.Errorf("api = %q, want %q", got.Get(paramAPI), apiTeamDownload)
	}
}
