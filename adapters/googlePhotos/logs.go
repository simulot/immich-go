package gp

import (
	"context"
	"encoding/csv"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/simulot/immich-go/helpers/gen"
	"github.com/simulot/immich-go/internal/fileevent"
)

// logMessage for the photo and the movie attached to the photo
func (to *Takeout) logMessage(ctx context.Context, code fileevent.Code, f fileevent.FileAndName, reason string) {
	t := "reason"
	if code == fileevent.Error {
		t = "error"
	}
	to.log.Record(ctx, code, f, t, reason)
}

func (to *Takeout) DebugLinkedFiles(w io.Writer) {
	csvWriter := csv.NewWriter(w)
	_ = csvWriter.Write([]string{"Dir", "Base", "Image", "Image Size", "Video", "Video Size"})

	slices.SortFunc(to.debugLinkedFiles, func(a, b linkedFiles) int {
		if cmp := strings.Compare(a.dir, b.dir); cmp != 0 {
			return cmp
		}
		return strings.Compare(a.base, b.base)
	})

	line := make([]string, 6)
	for _, k := range to.debugLinkedFiles {
		line[0] = k.dir
		line[1] = k.base
		if k.image != nil {
			line[2] = k.image.base
			line[3] = strconv.Itoa(k.image.length)
		} else {
			line[2] = ""
			line[3] = ""
		}
		if k.video != nil {
			line[4] = k.video.base
			line[5] = strconv.Itoa(k.video.length)
		} else {
			line[4] = ""
			line[5] = ""
		}
		_ = csvWriter.Write(line)
	}
	csvWriter.Flush()
}

func (to *Takeout) DebugFileTracker(w io.Writer) {
	csv := csv.NewWriter(w)
	_ = csv.Write([]string{"File", "Size", "Count", "Duplicated", "Uploaded", "Status", "Date", "Albums", "Paths"})

	keys := gen.MapKeys(to.fileTracker)

	slices.SortFunc(keys, trackerKeySortFunc)
	line := make([]string, 9)
	for _, k := range keys {
		track := to.fileTracker[k]
		line[0] = k.baseName
		line[1] = strconv.Itoa(int(k.size))     // Size
		line[2] = strconv.Itoa(track.count)     // Count
		line[3] = strconv.Itoa(track.count - 1) // Duplicated
		if track.status == fileevent.Uploaded {
			line[4] = "1" // Uploaded
		} else {
			line[4] = "0"
		}
		line[5] = track.status.String()
		if track.metadata != nil {
			line[6] = track.metadata.DateTaken.Format("2006-01-02 15:04:05 -0700") // Date
			line[7] = strings.Join(track.metadata.Collections, ",")                // Albums
		} else {
			line[6] = ""
			line[7] = ""
		}
		line[8] = strings.Join(track.paths, ",") // Paths
		_ = csv.Write(line)
	}
	csv.Flush()
}

/*
func (to *Takeout) DebugUploadedFiles(w io.Writer) {
	csv := csv.NewWriter(w)
	_ = csv.Write([]string{"File", "Size"})

	slices.SortFunc(to.debugUploadedFile, trackerKeySortFunc)
	line := make([]string, 2)
	for _, k := range to.debugUploadedFile {
		line[0] = k.baseName
		line[1] = strconv.Itoa(int(k.size))
		_ = csv.Write(line)
	}
	csv.Flush()
}
*/
