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
	Files map[string][]Entry
	log   logger.Logger
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
)

func NewJournal(log logger.Logger) *Journal {
	return &Journal{
		Files: map[string][]Entry{},
		log:   log,
	}
}

func (j *Journal) AddEntry(file string, action Action, comment string) {
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
	if j == nil {
		return
	}
	j.Lock()
	defer j.Unlock()
	j.Files[file] = append(j.Files[file], Entry{ts: time.Now(), action: action, comment: comment})
}

func (j *Journal) Report() {
	counts := map[Action]int{}

	for _, es := range j.Files {
		for _, e := range es {
			counts[e.action]++
		}
	}
	j.log.OK("Upload report:")
	j.log.OK("%6d errors", counts[ERROR])
	j.log.OK("%6d files scanned", counts[SCANNED])
	j.log.OK("%6d files discarded because in folder failed videos", counts[FAILED_VIDEO])
	j.log.OK("%6d files discarded because of options", counts[DISCARDED])
	j.log.OK("%6d files discarded because server has a better image", counts[SERVER_BETTER])
	j.log.OK("%6d files duplicated locally", counts[LOCAL_DUPLICATE])
	j.log.OK("%6d files already on the server", counts[SERVER_DUPLICATE])

	j.log.OK("%6d files uploaded on the server", counts[UPLOADED])
	j.log.OK("%6d files upgraded on the server", counts[UPGRADED])

}

func (j *Journal) WriteJournal(w io.Writer) {
	keys := gen.MapKeys(j.Files)
	sort.Strings(keys)
	for _, k := range keys {
		fmt.Fprintln(w, "File:", k)
		for _, e := range j.Files[k] {
			fmt.Fprint(w, "\t", e.action)
			if len(e.comment) > 0 {
				fmt.Fprint(w, ", ", e.comment)
			}
			fmt.Fprintln(w)
		}
	}

}
