// Package assetmatch provides shared asset deduplication logic
// used by both the upload and sync commands.
package assetmatch

import (
	"fmt"
	"time"
)

// AdviceCode represents the outcome of matching a local asset against server assets.
type AdviceCode int

const (
	IDontKnow        AdviceCode = iota
	SmallerOnServer             // server has a smaller version
	BetterOnServer              // server has a better (larger) version
	SameOnServer                // identical asset on server
	NotOnServer                 // asset not found on server
	AlreadyProcessed            // already processed in this session (upload-specific)
	ForceUpload                 // forced upload (--overwrite)
)

func (a AdviceCode) String() string {
	switch a {
	case IDontKnow:
		return "IDontKnow"
	case SmallerOnServer:
		return "SmallerOnServer"
	case BetterOnServer:
		return "BetterOnServer"
	case SameOnServer:
		return "SameOnServer"
	case NotOnServer:
		return "NotOnServer"
	case AlreadyProcessed:
		return "AlreadyProcessed"
	case ForceUpload:
		return "ForceUpload"
	}
	return fmt.Sprintf("advice(%d)", a)
}

// ServerAsset is a minimal representation of an asset on the server,
// carrying just enough data for matching decisions.
type ServerAsset struct {
	ID          string
	Checksum    string
	Filename    string
	CaptureDate time.Time
	Size        int64
}

// Advice is the result of matching a local asset against the server index.
type Advice struct {
	Code        AdviceCode
	Message     string
	ServerAsset *ServerAsset // nil when NotOnServer
}

// Index indexes server assets for efficient matching.
type Index struct {
	byChecksum map[string]*ServerAsset
	byName     map[string][]*ServerAsset
}

// NewIndex creates an empty asset index.
func NewIndex() *Index {
	return &Index{
		byChecksum: make(map[string]*ServerAsset),
		byName:     make(map[string][]*ServerAsset),
	}
}

// Add adds a server asset to the index.
func (idx *Index) Add(a ServerAsset) {
	sa := &a
	if sa.Checksum != "" {
		idx.byChecksum[sa.Checksum] = sa
	}
	if sa.Filename != "" {
		idx.byName[sa.Filename] = append(idx.byName[sa.Filename], sa)
	}
}

// Len returns the number of indexed assets (by checksum).
func (idx *Index) Len() int {
	return len(idx.byChecksum)
}

// LookupChecksum returns the server asset with the given checksum, if any.
func (idx *Index) LookupChecksum(checksum string) (*ServerAsset, bool) {
	sa, ok := idx.byChecksum[checksum]
	return sa, ok
}

// Match checks whether a local asset (identified by checksum, filename,
// captureDate, and size) already exists on the server.
//
// Algorithm: first try exact checksum match (SHA1), then fall back to
// filename + capture date (±5s tolerance) + size comparison.
func (idx *Index) Match(checksum, filename string, captureDate time.Time, size int64) Advice {
	// Phase 1: exact checksum match
	if sa, ok := idx.byChecksum[checksum]; ok {
		return Advice{
			Code:        SameOnServer,
			Message:     fmt.Sprintf("asset with same checksum exists on server (id=%s)", sa.ID),
			ServerAsset: sa,
		}
	}

	// Phase 2: name + date + size heuristic
	candidates, ok := idx.byName[filename]
	if ok {
		for _, sa := range candidates {
			cmpDate := CompareDate(captureDate, sa.CaptureDate)
			cmpSize := size - sa.Size

			switch {
			case cmpDate == 0 && cmpSize == 0:
				return Advice{
					Code:        SameOnServer,
					Message:     fmt.Sprintf("asset with same name %q, date, and size exists on server", filename),
					ServerAsset: sa,
				}
			case cmpDate == 0 && cmpSize > 0:
				return Advice{
					Code:        SmallerOnServer,
					Message:     fmt.Sprintf("asset with same name %q and date but smaller size on server (%d < %d)", filename, sa.Size, size),
					ServerAsset: sa,
				}
			case cmpDate == 0 && cmpSize < 0:
				return Advice{
					Code:        BetterOnServer,
					Message:     fmt.Sprintf("asset with same name %q and date but larger size on server (%d > %d)", filename, sa.Size, size),
					ServerAsset: sa,
				}
			}
		}
	}

	return Advice{
		Code:    NotOnServer,
		Message: "asset not found on server",
	}
}

// CompareDate compares two dates with a ±5 second tolerance.
// Returns 0 if within tolerance, -1 if d1 is earlier, +1 if d1 is later.
func CompareDate(d1, d2 time.Time) int {
	diff := d1.Sub(d2)
	switch {
	case diff < -5*time.Second:
		return -1
	case diff >= 5*time.Second:
		return +1
	}
	return 0
}
