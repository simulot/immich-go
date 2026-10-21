package immich

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

// The calls that read one asset refuse an empty ID without calling the server: GET /assets/,
// GET /albums?assetId= and a search with no ID ask for something else.
func TestEmptyAssetIDRefused(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected call %s %s", r.Method, r.URL)
		http.Error(w, "unexpected", http.StatusBadRequest)
	}))
	defer srv.Close()
	ic, err := NewImmichClient(srv.URL, "test-key")
	if err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := ic.GetAssetInfo(ctx, ""); err == nil {
		t.Error("GetAssetInfo with no ID: no error")
	}
	if _, err := ic.GetAssetAlbums(ctx, ""); err == nil {
		t.Error("GetAssetAlbums with no ID: no error")
	}
	if in, err := ic.IsInStack(ctx, ""); err == nil || in {
		t.Errorf("IsInStack with no ID: %v, %v; want false and an error", in, err)
	}
}

// A failed search is no answer: IsInStack says false with the error, never "in a stack".
func TestIsInStackFailedSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"message": "failed on purpose"}`, http.StatusInternalServerError)
	}))
	defer srv.Close()
	ic, err := NewImmichClient(srv.URL, "test-key")
	if err != nil {
		t.Fatal(err)
	}
	in, err := ic.IsInStack(context.Background(), "aaaaaaaa-0000-4000-8000-000000000001")
	if err == nil || in {
		t.Errorf("IsInStack after a failed search: %v, %v; want false and an error", in, err)
	}
}
