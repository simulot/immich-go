package gp

import (
	"context"
	"encoding/csv"
	"io"
	"slices"
	"strconv"
	"strings"

	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fshelper"
)

// logMessage for the photo and the movie attached to the photo
func (toc *TakeoutCmd) logMessage(ctx context.Context, code fileevent.Code, file fshelper.FSAndName, reason string) {
	t := "reason"
	if code == fileevent.Error {
		t = "error"
	}
	toc.processor.RecordNonAsset(ctx, file, 0, code, t, reason)
}

func (toc *TakeoutCmd) DebugFileTracker(w io.Writer) {
	csv := csv.NewWriter(w)
	_ = csv.Write([]string{"File", "Size", "Count", "Duplicated", "Uploaded", "Status", "Date", "Albums", "Paths"})

	keys := toc.fileTracker.Keys()

	slices.SortFunc(keys, trackerKeySortFunc)
	line := make([]string, 9)
	for _, k := range keys {
		track, _ := toc.fileTracker.Load(k)
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
			albums := make([]string, 0, len(track.metadata.Albums))
			for _, a := range track.metadata.Albums {
				albums = append(albums, a.Title)
			}
			line[6] = track.metadata.DateTaken.Format("2006-01-02 15:04:05 -0700") // Date
			line[7] = strings.Join(albums, ",")                                    // Albums
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
