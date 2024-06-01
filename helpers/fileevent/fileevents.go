package fileevent

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
)

/*
	Collect all actions done on a given file
*/

type Code int

const (
	DiscoveredImage       Code = iota // = "Scanned image"
	DiscoveredVideo                   // = "Scanned video"
	DiscoveredSidecar                 // = "Scanned side car file"
	DiscoveredDiscarded               // = "Discarded"
	DiscoveredUnsupported             // = "File type not supported"

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

	Error

	Uploaded    // = "Uploaded"
	Stacked     // = "Stacked"
	LivePhoto   // = "Live photo"
	FailedVideo // = "Failed video"
	Metadata    // = "Metadata files"
	INFO        // = "Info"
	MaxCode
)

var _code = map[Code]string{
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
	UploadServerError:     "server error",
	Uploaded:              "uploaded",

	Error: "error",

	Stacked:     "Stacked",
	LivePhoto:   "Live photo",
	FailedVideo: "Failed video",
	Metadata:    "Metadata files",
	INFO:        "Info",
}

func (e Code) String() string {
	if s, ok := _code[e]; ok {
		return s
	}
	return fmt.Sprintf("unknow event code: %d", int(e))
}

type event struct {
	Code   Code
	Object any
	File   string
	Args   []any
}

type Recorder struct {
	lock   sync.RWMutex
	counts []int64
	events map[Code][]event
	log    *slog.Logger
}

func NewRecorder(l *slog.Logger) *Recorder {
	r := &Recorder{
		counts: make([]int64, MaxCode),
		events: map[Code][]event{},
		log:    l,
	}
	return r
}

func (r *Recorder) Record(ctx context.Context, code Code, object any, file string, args ...any) {
	atomic.AddInt64(&r.counts[code], 1)
	switch code { // nolint:gocritic
	case DiscoveredDiscarded:
		r.lock.Lock()
		r.events[code] = append(r.events[code], event{Code: code, Object: object, File: file, Args: args})
		r.lock.Unlock()
	}
	if r.log != nil {
		if file != "" {
			args = append([]any{"file", file}, args...)
		}
		r.log.Log(ctx, slog.LevelInfo, code.String(), args...)
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
