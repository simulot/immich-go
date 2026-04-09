package snapchat

import (
	"io/fs"
	"sort"
	"time"

	"github.com/simulot/immich-go/internal/assets"
)

const (
	memoriesHistoryJSON = "json/memories_history.json"
	mergedAssetName     = "merged"
)

type fileRef struct {
	fsys    fs.FS
	name    string
	base    string
	size    int64
	modTime time.Time
}

type catalogEntry struct {
	sid      string
	main     *fileRef
	overlay  *fileRef
	metadata *assets.Metadata
}

type catalog struct {
	entries       map[string]*catalogEntry
	metadataCount int
}

func newCatalog() *catalog {
	return &catalog{entries: map[string]*catalogEntry{}}
}

func (c *catalog) getOrCreate(sid string) *catalogEntry {
	e, ok := c.entries[sid]
	if ok {
		return e
	}
	e = &catalogEntry{sid: sid}
	c.entries[sid] = e
	return e
}

func (c *catalog) setMain(sid string, ref *fileRef) {
	e := c.getOrCreate(sid)
	if e.main == nil {
		e.main = ref
		return
	}
	if ref.size > e.main.size {
		e.main = ref
	}
}

func (c *catalog) setOverlay(sid string, ref *fileRef) {
	e := c.getOrCreate(sid)
	if e.overlay == nil {
		e.overlay = ref
		return
	}
	if ref.size > e.overlay.size {
		e.overlay = ref
	}
}

func (c *catalog) addMetadata(m map[string]*assets.Metadata) {
	for sid, md := range m {
		e := c.getOrCreate(sid)
		e.metadata = md
	}
	c.metadataCount += len(m)
}

func (c *catalog) sortedEntries() []*catalogEntry {
	out := make([]*catalogEntry, 0, len(c.entries))
	for _, e := range c.entries {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].sid < out[j].sid
	})
	return out
}

type mediaFile struct {
	sid      string
	fsys     fs.FS
	name     string
	base     string
	size     int64
	modTime  time.Time
	main     *fileRef
	overlay  *fileRef
	metadata *assets.Metadata
}
