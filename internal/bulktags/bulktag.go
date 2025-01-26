package bulktags

import (
	"context"
	"log/slog"
	"sync"

	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/gen/syncmap"
)

const bulkBatchSize = 100

type BulkTagManager struct {
	ctx     context.Context
	client  immich.ImmichTagInterface
	logger  *slog.Logger
	tags    *syncmap.SyncMap[string, []string] // map of tag value to assets
	tagsID  *syncmap.SyncMap[string, string]   // map of tag value to ID
	tagChan chan struct {
		tag     string
		assetID string
	}
	done chan struct{}
	wg   sync.WaitGroup
}

func NewBulkTagManager(ctx context.Context, client immich.ImmichTagInterface, logger *slog.Logger) *BulkTagManager {
	bm := &BulkTagManager{
		ctx:    ctx,
		client: client,
		logger: logger,
		tags:   syncmap.New[string, []string](),
		tagsID: syncmap.New[string, string](),
		tagChan: make(chan struct {
			tag     string
			assetID string
		}),
		done: make(chan struct{}),
	}

	go bm.tagWorker()
	return bm
}

func (m *BulkTagManager) AddTag(tag string, assetID string) {
	if len(assetID) == 0 || len(tag) == 0 {
		return
	}
	m.tagChan <- struct{ tag, assetID string }{tag, assetID}
}

func (m *BulkTagManager) Close() {
	close(m.tagChan)
	<-m.done
}

// tagWorker is a goroutine that listens for tags+ids to be added to the BulkTagManager.
// it protects the tags map from concurrent access.
func (m *BulkTagManager) tagWorker() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case t, ok := <-m.tagChan:
			if !ok {
				m.flush()
				return
			}
			ids, _ := m.tags.Load(t.tag)
			ids = append(ids, t.assetID)
			m.tags.Store(t.tag, ids)
			if len(ids) >= bulkBatchSize {
				m.flushTag(t.tag)
			}
		}
	}
}

func (m *BulkTagManager) flushTag(tag string) {
	ids, ok := m.tags.Swap(tag, []string{})
	if !ok {
		return
	}

	ID, ok := m.tagsID.Load(tag)
	if !ok {
		tags, err := m.client.UpsertTags(m.ctx, []string{tag})
		if err != nil {
			m.logger.Error("Error upserting tag", "Tag", tag, "error", err)
			return
		}
		if len(tags) == 0 || tags[0].ID == "" {
			m.logger.Error("Error upserting tag", "Tag", tag, "error", "no tag ID returned")
			return
		}
		ID = tags[0].ID
		m.tagsID.Store(tag, ID)
	}
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		_, err := m.client.BulkTagAssets(m.ctx, []string{ID}, ids)
		if err != nil {
			m.logger.Error("Error tagging assets with tag", "Tag", tag, "error", err)
		}
	}()
}

func (m *BulkTagManager) flush() {
	for _, tag := range m.tags.Keys() {
		m.flushTag(tag)
	}
	m.wg.Wait()
	close(m.done)
}
