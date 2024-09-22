// Package fileevent provides a mechanism to record and report events related to file processing.

package fileevent

/*
	TODO:
	- rename the package as journal
	- use a filenemame type that keeps the fsys and the name in that fsys

*/
import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/simulot/immich-go/internal/fshelper"
)

/*
	Collect all actions done on a given file
*/

type Code int

const (
	NotHandled            Code = iota
	DiscoveredImage            // = "Scanned image"
	DiscoveredVideo            // = "Scanned video"
	DiscoveredSidecar          // = "Scanned side car file"
	DiscoveredDiscarded        // = "Discarded"
	DiscoveredUnsupported      // = "File type not supported"

	AnalysisAssociatedMetadata
	AnalysisMissingAssociatedMetadata
	AnalysisLocalDuplicate

	UploadNotSelected
	UploadUpgraded        // = "Server's asset upgraded"
	UploadServerDuplicate // = "Server has photo"
	UploadServerBetter    // = "Server's asset is better"
	UploadAlbumCreated
	UploadAddToAlbum  // = "Added to an album"
	UploadServerError // = "Server error"

	Uploaded  // = "Uploaded"
	Stacked   // = "Stacked"
	LivePhoto // = "Live photo"
	Metadata  // = "Metadata files"
	INFO      // = "Info"
	Error
	MaxCode
)

var _code = map[Code]string{
	NotHandled:            "Not handled",
	DiscoveredImage:       "scanned image file",
	DiscoveredVideo:       "scanned video file",
	DiscoveredSidecar:     "scanned sidecar file",
	DiscoveredDiscarded:   "discarded file",
	DiscoveredUnsupported: "unsupported file",

	AnalysisAssociatedMetadata:        "associated metadata file",
	AnalysisMissingAssociatedMetadata: "missing associated metadata file",
	AnalysisLocalDuplicate:            "file duplicated in the input",

	UploadNotSelected:     "file not selected",
	UploadUpgraded:        "server's asset upgraded with the input",
	UploadAddToAlbum:      "added to an album",
	UploadServerDuplicate: "server has same asset",
	UploadServerBetter:    "server has a better asset",
	UploadAlbumCreated:    "album created/updated",
	UploadServerError:     "upload error",
	Uploaded:              "uploaded",

	Stacked:   "Stacked",
	LivePhoto: "Live photo",
	Metadata:  "Metadata files",
	INFO:      "Info",
	Error:     "error",
}

func (e Code) String() string {
	if s, ok := _code[e]; ok {
		return s
	}
	return fmt.Sprintf("unknown event code: %d", int(e))
}

type Recorder struct {
	lock   sync.RWMutex
	counts counts
	log    *slog.Logger
}

type FileAndName struct {
	fsys fs.FS
	name string
}

func AsFileAndName(fsys fs.FS, name string) FileAndName {
	return FileAndName{fsys: fsys, name: name}
}

func (fn FileAndName) Name() string {
	fsys := fn.fsys
	if fsys, ok := fsys.(fshelper.NameFS); ok {
		return fsys.Name() + ":" + fn.name
	}
	return fn.name
}

type counts []int64

func NewRecorder(l *slog.Logger, debug bool) *Recorder {
	r := &Recorder{
		counts: make([]int64, MaxCode),
		log:    l,
	}
	return r
}

func (r *Recorder) Record(ctx context.Context, code Code, file FileAndName, args ...any) {
	atomic.AddInt64(&r.counts[code], 1)
	if r.log != nil {
		level := slog.LevelInfo
		if file.name != "" {
			args = append([]any{"file", file.Name()}, args...)
		}
		for _, a := range args {
			if a == "error" {
				level = slog.LevelError
			}
		}
		r.log.Log(ctx, level, code.String(), args...)
	}
}

func (r *Recorder) SetLogger(l *slog.Logger) {
	r.log = l
}

func (r *Recorder) Report() {
	sb := strings.Builder{}

	sb.WriteString("\n")
	sb.WriteString("Input analysis:\n")
	sb.WriteString("---------------\n")
	for _, c := range []Code{
		DiscoveredImage,
		DiscoveredVideo,
		DiscoveredSidecar,
		DiscoveredDiscarded,
		DiscoveredUnsupported,
		AnalysisLocalDuplicate,
		AnalysisAssociatedMetadata,
		AnalysisMissingAssociatedMetadata,
	} {
		sb.WriteString(fmt.Sprintf("%-40s: %7d\n", c.String(), r.counts[c]))
	}

	sb.WriteString("\n")
	sb.WriteString("Uploading:\n")
	sb.WriteString("----------\n")
	for _, c := range []Code{
		Uploaded,
		UploadServerError,
		UploadNotSelected,
		UploadUpgraded,
		UploadServerDuplicate,
		UploadServerBetter,
	} {
		sb.WriteString(fmt.Sprintf("%-40s: %7d\n", c.String(), r.counts[c]))
	}

	r.log.Info(sb.String())
	fmt.Println(sb.String())
}

func (r *Recorder) GetCounts() []int64 {
	r.lock.Lock()
	defer r.lock.Unlock()
	counts := make([]int64, MaxCode)
	copy(counts, r.counts)
	return counts
}

func (r *Recorder) TotalAssets() int64 {
	return atomic.LoadInt64(&r.counts[DiscoveredImage]) + atomic.LoadInt64(&r.counts[DiscoveredVideo])
}

func (r *Recorder) TotalProcessedGP() int64 {
	return atomic.LoadInt64(&r.counts[AnalysisAssociatedMetadata]) +
		atomic.LoadInt64(&r.counts[AnalysisMissingAssociatedMetadata]) +
		atomic.LoadInt64(&r.counts[DiscoveredDiscarded])
}

func (r *Recorder) TotalProcessed(forcedMissingJSON bool) int64 {
	v := atomic.LoadInt64(&r.counts[Uploaded]) +
		atomic.LoadInt64(&r.counts[UploadServerError]) +
		atomic.LoadInt64(&r.counts[UploadNotSelected]) +
		atomic.LoadInt64(&r.counts[UploadUpgraded]) +
		atomic.LoadInt64(&r.counts[UploadServerDuplicate]) +
		atomic.LoadInt64(&r.counts[UploadServerBetter]) +
		atomic.LoadInt64(&r.counts[DiscoveredDiscarded]) +
		atomic.LoadInt64(&r.counts[AnalysisLocalDuplicate])
	if !forcedMissingJSON {
		v += atomic.LoadInt64(&r.counts[AnalysisMissingAssociatedMetadata])
	}
	return v
}

// IsEqualCounts checks if two slices of int64 have the same elements in the same order.
// Used for tests only
func IsEqualCounts(a, b []int64) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// PrepareCountsForTest takes an undefined  number of int arguments and returns a slice of int64
// Used for tests only

func NewCounts() *counts {
	c := counts(make([]int64, MaxCode))
	return &c
}

func (cnt *counts) Set(c Code, v int64) *counts {
	(*cnt)[c] = v
	return cnt
}

func (cnt *counts) Value() []int64 {
	return (*cnt)[:MaxCode]
}
