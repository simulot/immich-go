package manage

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/simulot/immich-go/immich"
)

// newTestClient creates an ImmichClient pointed at a test server.
func newTestClient(t *testing.T, handler http.Handler) *immich.ImmichClient {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client, err := immich.NewImmichClient(server.URL, "test-key")
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func TestResolvePeopleIDs(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/api/people") {
			resp := immich.PeopleResponseDto{
				People: []immich.PersonResponseDto{
					{ID: "person-1", Name: "Alice"},
					{ID: "person-2", Name: "Bob"},
					{ID: "person-3", Name: "Charlie"},
				},
				HasNextPage: false,
				Total:       3,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	})

	client := newTestClient(t, handler)
	ctx := context.Background()

	sel := PeopleSelector{
		Names: []string{"Alice", "Bob"},
		IDs:   []string{"direct-uuid-1"},
	}

	ids, err := resolvePeopleIDs(ctx, client, sel)
	if err != nil {
		t.Fatal(err)
	}

	if len(ids) != 3 {
		t.Fatalf("expected 3 IDs, got %d: %v", len(ids), ids)
	}

	expected := map[string]bool{"person-1": true, "person-2": true, "direct-uuid-1": true}
	for _, id := range ids {
		if !expected[id] {
			t.Errorf("unexpected ID %q", id)
		}
	}
}

// searchResponse builds the JSON body for the /search/metadata endpoint.
// nextPage should be "0" to indicate no more pages.
func searchResponse(w http.ResponseWriter, assetIDs []string, nextPage string) {
	type item struct {
		ID string `json:"id"`
	}
	items := make([]item, len(assetIDs))
	for i, id := range assetIDs {
		items[i] = item{ID: id}
	}
	resp := struct {
		Assets struct {
			Total    int    `json:"total"`
			Count    int    `json:"count"`
			Items    []item `json:"items"`
			NextPage string `json:"nextPage"`
		} `json:"assets"`
	}{}
	resp.Assets.Total = len(assetIDs)
	resp.Assets.Count = len(assetIDs)
	resp.Assets.Items = items
	resp.Assets.NextPage = nextPage
	json.NewEncoder(w).Encode(resp)
}

// peopleResponse writes a standard people response.
func peopleResponse(w http.ResponseWriter, people []immich.PersonResponseDto) {
	json.NewEncoder(w).Encode(immich.PeopleResponseDto{
		People:      people,
		HasNextPage: false,
		Total:       len(people),
	})
}

func TestResolvePeopleIDs_NameNotFound(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(immich.PeopleResponseDto{
			People:      []immich.PersonResponseDto{},
			HasNextPage: false,
			Total:       0,
		})
	})

	client := newTestClient(t, handler)
	ctx := context.Background()

	sel := PeopleSelector{Names: []string{"NonExistent"}}
	_, err := resolvePeopleIDs(ctx, client, sel)
	if err == nil {
		t.Fatal("expected error for unresolved name, got nil")
	}
}

func TestSyncFlow_AddsOnlyMissingAssets(t *testing.T) {
	var mu sync.Mutex
	var addedAssets []string

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/api/people"):
			peopleResponse(w, []immich.PersonResponseDto{{ID: "person-alice", Name: "Alice"}})

		case r.Method == "GET" && r.URL.Path == "/api/albums" && r.URL.Query().Get("assetId") == "":
			json.NewEncoder(w).Encode([]immich.AlbumSimplified{
				{ID: "album-1", AlbumName: "Family", AssetIds: []string{"asset-1"}},
			})

		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/api/albums/album-1"):
			json.NewEncoder(w).Encode(immich.AlbumContent{
				ID:        "album-1",
				AlbumName: "Family",
				AssetIDs:  []string{"asset-1"},
			})

		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/api/search/metadata"):
			searchResponse(w, []string{"asset-1", "asset-2", "asset-3"}, "0")

		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/albums/") && strings.Contains(r.URL.Path, "/assets"):
			var body struct {
				IDS []string `json:"ids"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			mu.Lock()
			addedAssets = append(addedAssets, body.IDS...)
			mu.Unlock()
			results := make([]immich.UpdateAlbumResult, len(body.IDS))
			for i, id := range body.IDS {
				results[i] = immich.UpdateAlbumResult{ID: id, Success: true}
			}
			json.NewEncoder(w).Encode(results)

		default:
			http.NotFound(w, r)
		}
	})

	client := newTestClient(t, handler)
	ctx := context.Background()

	cfg := &PeopleAlbumSyncConfig{
		Albums: []AlbumMapping{
			{
				Album:  "Family",
				People: PeopleSelector{Names: []string{"Alice"}},
			},
		},
	}

	sc := &PeopleAlbumSyncCmd{}
	sc.client.Immich = client

	err := sc.run(ctx, slog.Default(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Should only add asset-2 and asset-3 (asset-1 already in album)
	// Note: GetFilteredAssetsFn runs 3 concurrent queries (one per visibility),
	// so we may get duplicates, if so use a set to deduplicate.
	addedSet := map[string]bool{}
	for _, id := range addedAssets {
		addedSet[id] = true
	}
	if addedSet["asset-1"] {
		t.Error("asset-1 should NOT have been added (already in album)")
	}
	if !addedSet["asset-2"] {
		t.Error("asset-2 should have been added")
	}
	if !addedSet["asset-3"] {
		t.Error("asset-3 should have been added")
	}
}

func TestSyncFlow_CreatesAlbumIfMissing(t *testing.T) {
	albumCreated := false

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/api/people"):
			peopleResponse(w, []immich.PersonResponseDto{{ID: "person-1", Name: "Alice"}})

		case r.Method == "GET" && r.URL.Path == "/api/albums" && r.URL.Query().Get("assetId") == "":
			json.NewEncoder(w).Encode([]immich.AlbumSimplified{})

		case r.Method == "POST" && r.URL.Path == "/api/albums":
			albumCreated = true
			json.NewEncoder(w).Encode(immich.AlbumSimplified{
				ID:        "new-album-id",
				AlbumName: "New Album",
			})

		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/api/albums/new-album-id"):
			json.NewEncoder(w).Encode(immich.AlbumContent{
				ID:        "new-album-id",
				AlbumName: "New Album",
				AssetIDs:  []string{},
			})

		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/api/search/metadata"):
			searchResponse(w, []string{"asset-1"}, "0")

		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/assets"):
			var body struct {
				IDS []string `json:"ids"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			results := make([]immich.UpdateAlbumResult, len(body.IDS))
			for i, id := range body.IDS {
				results[i] = immich.UpdateAlbumResult{ID: id, Success: true}
			}
			json.NewEncoder(w).Encode(results)

		default:
			http.NotFound(w, r)
		}
	})

	client := newTestClient(t, handler)
	ctx := context.Background()

	cfg := &PeopleAlbumSyncConfig{
		Albums: []AlbumMapping{
			{
				Album:  "New Album",
				People: PeopleSelector{Names: []string{"Alice"}},
			},
		},
	}

	sc := &PeopleAlbumSyncCmd{}
	sc.client.Immich = client

	err := sc.run(ctx, slog.Default(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	if !albumCreated {
		t.Error("expected album to be created")
	}
}

func TestSyncFlow_NeverCallsDestructiveAPIs(t *testing.T) {
	var mu sync.Mutex
	destructiveCalled := ""

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		if r.Method == "DELETE" {
			mu.Lock()
			destructiveCalled = "DELETE " + r.URL.Path
			mu.Unlock()
			http.Error(w, "should not be called", http.StatusInternalServerError)
			return
		}

		switch {
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/api/people"):
			peopleResponse(w, []immich.PersonResponseDto{{ID: "p1", Name: "Alice"}})
		case r.Method == "GET" && r.URL.Path == "/api/albums" && r.URL.Query().Get("assetId") == "":
			json.NewEncoder(w).Encode([]immich.AlbumSimplified{
				{ID: "a1", AlbumName: "Test", AssetIds: []string{}},
			})
		case r.Method == "GET" && strings.HasPrefix(r.URL.Path, "/api/albums/a1"):
			json.NewEncoder(w).Encode(immich.AlbumContent{ID: "a1", AlbumName: "Test", AssetIDs: []string{}})
		case r.Method == "POST" && strings.HasPrefix(r.URL.Path, "/api/search/metadata"):
			searchResponse(w, []string{"x1"}, "0")
		case r.Method == "PUT" && strings.Contains(r.URL.Path, "/assets"):
			var body struct {
				IDS []string `json:"ids"`
			}
			json.NewDecoder(r.Body).Decode(&body)
			results := []immich.UpdateAlbumResult{{ID: "x1", Success: true}}
			json.NewEncoder(w).Encode(results)
		default:
			http.NotFound(w, r)
		}
	})

	client := newTestClient(t, handler)
	ctx := context.Background()

	cfg := &PeopleAlbumSyncConfig{
		Albums: []AlbumMapping{{
			Album:  "Test",
			People: PeopleSelector{Names: []string{"Alice"}},
		}},
	}

	sc := &PeopleAlbumSyncCmd{}
	sc.client.Immich = client

	err := sc.run(ctx, slog.Default(), cfg)
	if err != nil {
		t.Fatal(err)
	}

	mu.Lock()
	defer mu.Unlock()
	if destructiveCalled != "" {
		t.Fatalf("destructive API call made: %s", destructiveCalled)
	}
}
