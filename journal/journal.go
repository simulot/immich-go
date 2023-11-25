package journal

import (
	"fmt"
	"immich-go/helpers/gen"
	"immich-go/logger"
	"io"
	"sort"
	"sync"
	"time"
)

type Journal struct {
	sync.RWMutex
	Files map[string]Entries
	log   logger.Logger
}

type Entries struct {
	terminated bool
	entries    []Entry
}

type Entry struct {
	ts      time.Time
	action  Action
	comment string
}

type Action string

const (
	// INFO             Action = "Information"
	SCANNED          Action = "Scanned"
	DISCARDED        Action = "Discarded"
	UPLOADED         Action = "Uploaded"
	UPGRADED         Action = "Server photo upgraded"
	ERROR            Action = "Error"
	LOCAL_DUPLICATE  Action = "Local duplicate"
	SERVER_DUPLICATE Action = "Server has photo"
	STACKED          Action = "Stacked"
	SERVER_BETTER    Action = "Server photo is better"
	ALBUM            Action = "Added to an album"
	LIVE_PHOTO       Action = "Live photo"
	FAILED_VIDEO     Action = "Failed video"
	NOT_SUPPORTED    Action = "File type not supported"
	METADATA         Action = "Metadata files"
)

func NewJournal(log logger.Logger) *Journal {
	return &Journal{
		Files: map[string]Entries{},
		log:   log,
	}
}

func (j *Journal) AddEntry(file string, action Action, comment string) {
	if j == nil {
		return
	}
	if j.log != nil {
		switch action {

		case ERROR:
			j.log.Error("%-40s: %s: %s", action, file, comment)
		case UPLOADED:
			j.log.OK("%-40s: %s: %s", action, file, comment)
		default:
			j.log.Info("%-40s: %s: %s", action, file, comment)
		}
	}
	j.Lock()
	defer j.Unlock()
	e := j.Files[file]

	switch action {
	case DISCARDED, UPGRADED, UPLOADED, LOCAL_DUPLICATE, SERVER_DUPLICATE, SERVER_BETTER, FAILED_VIDEO, NOT_SUPPORTED, METADATA, ERROR:
		if e.terminated {
			return
			// j.log.Error("%-40s: Already terminated %s: %s", action, file, comment)
		}
		e.terminated = true
	}
	e.entries = append(e.entries, Entry{ts: time.Now(), action: action, comment: comment})
	j.Files[file] = e
}

func (j *Journal) Report() {
	counts := map[Action]int{}
	terminated := 0

	for _, es := range j.Files {
		for _, e := range es.entries {
			counts[e.action]++
		}
		if es.terminated {
			terminated++
		}
	}
	j.log.OK("Upload report:")
	j.log.OK("%6d scanned files", len(j.Files))
	j.log.OK("%6d handled files", terminated)
	j.log.OK("%6d metadata files", counts[METADATA])
	j.log.OK("%6d uploaded files on the server", counts[UPLOADED])
	j.log.OK("%6d upgraded files on the server", counts[UPGRADED])
	j.log.OK("%6d duplicated files in the input", counts[LOCAL_DUPLICATE])
	j.log.OK("%6d files already on the server", counts[SERVER_DUPLICATE])

	j.log.OK("%6d discarded files because in folder failed videos", counts[FAILED_VIDEO])
	j.log.OK("%6d discarded files because of options", counts[DISCARDED])
	j.log.OK("%6d discarded files because server has a better image", counts[SERVER_BETTER])
	j.log.OK("%6d files type not supported", counts[NOT_SUPPORTED])
	j.log.OK("%6d errors", counts[ERROR])

}

func (j *Journal) WriteJournal(w io.Writer) {
	keys := gen.MapKeys(j.Files)
	sort.Strings(keys)
	for _, k := range keys {
		if !j.Files[k].terminated {
			fmt.Fprintln(w, "File:", k)
			for _, e := range j.Files[k].entries {
				fmt.Fprint(w, "\t", e.action)
				if len(e.comment) > 0 {
					fmt.Fprint(w, ", ", e.comment)
				}
				fmt.Fprintln(w)
			}
		}
	}
}
