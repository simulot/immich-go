package bulktags

import (
	"context"
	"io"
	"log/slog"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/simulot/immich-go/immich"
	"github.com/stretchr/testify/assert"
)

type MockImmichTagInterface struct {
	sync.Mutex
	GetAllTagsFunc    func(ctx context.Context) ([]immich.TagSimplified, error)
	UpsertTagsFunc    func(ctx context.Context, tags []string) ([]immich.TagSimplified, error)
	TagAssetsFunc     func(ctx context.Context, tagID string, assetIDs []string) ([]immich.TagAssetsResponse, error)
	BulkTagAssetsFunc func(ctx context.Context, tagIDs []string, assetIDs []string) (struct {
		Count int `json:"count"`
	}, error)
	assets         map[string][]string
	idsUpsertCount map[string]int
}

func (m *MockImmichTagInterface) GetAllTags(ctx context.Context) ([]immich.TagSimplified, error) {
	return m.GetAllTagsFunc(ctx)
}

func (m *MockImmichTagInterface) UpsertTags(ctx context.Context, tags []string) ([]immich.TagSimplified, error) {
	return m.UpsertTagsFunc(ctx, tags)
}

func (m *MockImmichTagInterface) TagAssets(ctx context.Context, tagID string, assetIDs []string) ([]immich.TagAssetsResponse, error) {
	return m.TagAssetsFunc(ctx, tagID, assetIDs)
}

func (m *MockImmichTagInterface) BulkTagAssets(ctx context.Context, tagIDs []string, assetIDs []string) (struct {
	Count int `json:"count"`
}, error,
) {
	return m.BulkTagAssetsFunc(ctx, tagIDs, assetIDs)
}

func mustBool[T any, B bool](t T, _ B) T {
	return t
}

func TestBulkTagManager_AddTag(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockImmichTagInterface{
		UpsertTagsFunc: func(ctx context.Context, tags []string) ([]immich.TagSimplified, error) {
			r := make([]immich.TagSimplified, len(tags))
			for i, tag := range tags {
				r[i] = immich.TagSimplified{ID: tag + "ID", Name: tag, Value: tag}
			}
			return r, nil
		},
		BulkTagAssetsFunc: func(ctx context.Context, tagIDs []string, assetIDs []string) (struct {
			Count int `json:"count"`
		}, error,
		) {
			return struct {
				Count int `json:"count"`
			}{Count: 1}, nil
		},
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bm := NewBulkTagManager(ctx, mockClient, logger)

	bm.AddTag("tag1", "asset1")
	bm.AddTag("tag2", "asset2")

	time.Sleep(100 * time.Millisecond) // Give some time for the goroutine to process

	assert.Equal(t, []string{"asset1"}, mustBool(bm.tags.Load("tag1")))
	assert.Equal(t, []string{"asset2"}, mustBool(bm.tags.Load("tag2")))

	bm.Close()
}

func TestBulkTagManager_Tag1000AssetsWith5Tags(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockImmichTagInterface{
		assets:         make(map[string][]string),
		idsUpsertCount: make(map[string]int),
	}

	mockClient.UpsertTagsFunc = func(ctx context.Context, tags []string) ([]immich.TagSimplified, error) {
		mockClient.Lock()
		defer mockClient.Unlock()

		r := make([]immich.TagSimplified, len(tags))
		for i, tag := range tags {
			ID := tag + "ID"
			mockClient.idsUpsertCount[ID]++
			r[i] = immich.TagSimplified{ID: ID, Name: tag, Value: tag}
		}
		return r, nil
	}

	mockClient.BulkTagAssetsFunc = func(ctx context.Context, tagIDs []string, assetIDs []string) (struct {
		Count int `json:"count"`
	}, error,
	) {
		mockClient.Lock()
		defer mockClient.Unlock()
		for _, tagID := range tagIDs {
			for _, assetID := range assetIDs {
				l := mockClient.assets[tagID]
				for i := range l {
					if l[i] == assetID {
						t.Fatalf("asset %s already tagged with tag %s", assetID, tagID)
					}
				}
				l = append(l, assetID)
				mockClient.assets[tagID] = l
			}
		}
		return struct {
			Count int `json:"count"`
		}{Count: len(assetIDs)}, nil
	}

	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bm := NewBulkTagManager(ctx, mockClient, logger)

	const n = 1000
	for i := 1; i <= n; i++ {
		for j := 1; j <= 5; j++ {
			if i%j == 0 {
				bm.AddTag("tag"+strconv.Itoa(j), "asset"+strconv.Itoa(i))
			}
		}
	}
	bm.Close()

	// Upsert is called just once per tag
	for _, v := range mockClient.idsUpsertCount {
		assert.Equal(t, 1, v)
	}
	// Check that each asset is tagged with the correct tags
	assert.Equal(t, n, len(mockClient.assets["tag1ID"]))
	assert.Equal(t, int(n/2), len(mockClient.assets["tag2ID"]))
	assert.Equal(t, int(n/3), len(mockClient.assets["tag3ID"]))
	assert.Equal(t, int(n/4), len(mockClient.assets["tag4ID"]))
	assert.Equal(t, int(n/5), len(mockClient.assets["tag5ID"]))
}

func TestBulkTagManager_NoTagsSubmitted(t *testing.T) {
	ctx := context.Background()
	mockClient := &MockImmichTagInterface{
		UpsertTagsFunc: func(ctx context.Context, tags []string) ([]immich.TagSimplified, error) {
			return []immich.TagSimplified{}, nil
		},
		BulkTagAssetsFunc: func(ctx context.Context, tagIDs []string, assetIDs []string) (struct {
			Count int `json:"count"`
		}, error,
		) {
			return struct {
				Count int `json:"count"`
			}{Count: 0}, nil
		},
		assets: make(map[string][]string),
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	bm := NewBulkTagManager(ctx, mockClient, logger)

	bm.Close()

	assert.Empty(t, mockClient.assets)
}
