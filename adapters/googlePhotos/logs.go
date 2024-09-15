package gp

import (
	"context"
	"encoding/csv"
	"io"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/simulot/immich-go/helpers/gen"
	"github.com/simulot/immich-go/internal/fileevent"
)

// logMessage for the photo and the movie attached to the photo
func (to *Takeout) logMessage(ctx context.Context, code fileevent.Code, file string, reason string) {
	t := "reason"
	if code == fileevent.Error {
		t = "error"
	}
	to.log.Record(ctx, code, nil, file, t, reason)
}

func (to *Takeout) DebugDuplicates(w io.Writer) {
	csv := csv.NewWriter(w)
	_ = csv.Write([]string{"File", "Size", "Count"})
	dups := gen.MapKeys(to.duplicates)

	slices.SortFunc(dups, tackerKeySortFunc)

	for _, k := range dups {
		_ = csv.Write([]string{k.baseName, strconv.Itoa(int(k.size)), strconv.Itoa(to.duplicates[k])})
	}
	csv.Flush()
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
	_ = csv.Write([]string{"File", "Size", "Path"})

	keys := gen.MapKeys(to.fileTracker)

	slices.SortFunc(keys, tackerKeySortFunc)
	for _, k := range keys {
		line := make([]string, 3)
		line[0] = k.baseName
		line[1] = strconv.Itoa(int(k.size))
		paths := to.fileTracker[k]
		sort.Strings(paths)
		for _, p := range paths {
			line[2] = p
			_ = csv.Write(line)
		}
	}
	csv.Flush()
}
