package fileevent

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/simulot/immich-go/browser"
	"github.com/simulot/immich-go/helpers/gen"
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
	UploadTagAssetError

	Uploaded  // = "Uploaded"
	Stacked   // = "Stacked"
	LivePhoto // = "Live photo"
	Metadata  // = "Metadata files"
	INFO      // = "Info"
	Error
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
	UploadServerError:     "upload error",
	UploadTagAssetError:   "tag asset error",
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
	lock       sync.RWMutex
	counts     []int64
	fileEvents map[string]map[Code]int
	log        *slog.Logger
	debug      bool
}

func NewRecorder(l *slog.Logger, debug bool) *Recorder {
	r := &Recorder{
		counts:     make([]int64, MaxCode),
		fileEvents: map[string]map[Code]int{},
		log:        l,
		debug:      debug,
	}
	return r
}

func (r *Recorder) Record(ctx context.Context, code Code, object any, file string, args ...any) {
	atomic.AddInt64(&r.counts[code], 1)
	if r.debug && file != "" {
		r.lock.Lock()
		events := r.fileEvents[file]
		if events == nil {
			events = map[Code]int{}
		}
		v := events[code] + 1
		events[code] = v
		r.fileEvents[file] = events
		r.lock.Unlock()
	}
	if r.log != nil {
		level := slog.LevelInfo
		if file != "" {
			args = append([]any{"file", file}, args...)
		}
		for _, a := range args {
			if a == "error" {
				level = slog.LevelError
			}
		}
		r.log.Log(ctx, level, code.String(), args...)
	}
	if a, ok := object.(*browser.LocalAssetFile); ok && a.LivePhoto != nil {
		arg2 := []any{}
		for i := 0; i < len(args); i++ {
			if args[i] == "file" {
				i += 1
				continue
			}
			arg2 = append(arg2, args[i])
		}
		r.Record(ctx, code, a.LivePhoto, a.LivePhoto.FileName, arg2...)
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

func (r *Recorder) WriteFileCounts(w io.Writer) error {
	reportCodes := []Code{
		-1,
		DiscoveredImage,
		DiscoveredVideo,
		AnalysisAssociatedMetadata,
		DiscoveredDiscarded,
		DiscoveredUnsupported,
		AnalysisLocalDuplicate,
		UploadNotSelected,
		UploadUpgraded,
		UploadServerBetter,
		UploadServerDuplicate,
		Uploaded,
	}
	fmt.Fprint(w, "File,")
	for _, c := range reportCodes {
		if c >= 0 {
			fmt.Fprint(w, strings.Replace(c.String(), " ", "_", -1)+",")
		} else {
			fmt.Fprint(w, "check,")
		}
	}
	fmt.Fprintln(w)
	keys := gen.MapKeys(r.fileEvents)
	sort.Strings(keys)
	for _, f := range keys {
		fmt.Fprint(w, "\"", f, "\",")
		e := r.fileEvents[f]
		check := 0
		for _, c := range reportCodes {
			if c >= 0 {
				check += e[c]
			}
		}
		for _, c := range reportCodes {
			if c >= 0 {
				fmt.Fprint(w, e[c], ",")
			} else {
				fmt.Fprint(w, check, ",")
			}
		}
		fmt.Fprintln(w)
	}
	return nil
}

func (r *Recorder) TotalAssets() int64 {
	return atomic.LoadInt64(
		&r.counts[DiscoveredImage],
	) + atomic.LoadInt64(
		&r.counts[DiscoveredVideo],
	)
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
