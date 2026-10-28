package immich_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
)

func TestSetAssetsVisibility(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotMethod, gotPath, gotBody = r.Method, r.URL.Path, strings.TrimSpace(string(b))
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := immich.NewImmichClient(server.URL, "test-key")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.SetAssetsVisibility(context.Background(), []string{"a1"}, assets.VisibilityLocked); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotMethod != http.MethodPut || gotPath != "/api/assets" {
		t.Errorf("got %s %s, want PUT /api/assets", gotMethod, gotPath)
	}
	// only ids and visibility: any other field would reset the asset's archived, favorite or location state
	if want := `{"ids":["a1"],"visibility":"locked"}`; gotBody != want {
		t.Errorf("body: got %s, want %s", gotBody, want)
	}
}

func TestSetAssetsVisibilityDryRun(t *testing.T) {
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	client, err := immich.NewImmichClient(server.URL, "test-key", immich.OptionDryRun(true))
	if err != nil {
		t.Fatal(err)
	}
	if err := client.SetAssetsVisibility(context.Background(), []string{"a1"}, assets.VisibilityLocked); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if called {
		t.Error("dry run must not call the server")
	}
}
