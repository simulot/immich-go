package journal

import (
	"immich-go/helpers/gen"
	"immich-go/logger"
	"slices"
	"sort"
	"strings"
	"sync"
	"time"
)

type Journal struct {
	sync.RWMutex
	files  map[string]Entries
	counts map[Action]int
	log    logger.Logger
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
	UPGRADED         Action = "Server's asset upgraded"
	ERROR            Action = "Error"
	LOCAL_DUPLICATE  Action = "Local duplicate"
	SERVER_DUPLICATE Action = "Server has photo"
	STACKED          Action = "Stacked"
	SERVER_BETTER    Action = "Server's asset is better"
	ALBUM            Action = "Added to an album"
	LIVE_PHOTO       Action = "Live photo"
	FAILED_VIDEO     Action = "Failed video"
	UNSUPPORTED      Action = "File type not supported"
	METADATA         Action = "Metadata files"
	UNHANDLED        Action = "File unhandled"
	HANDLED          Action = "File handled"
	INFO             Action = "Info"
)

func NewJournal(log logger.Logger) *Journal {
	return &Journal{
		files: map[string]Entries{},
		log:   log,
	}
}

func (j *Journal) AddEntry(file string, action Action, comment ...string) {
	if j == nil {
		return
	}
	c := strings.Join(comment, ", ")
	if j.log != nil {
		switch action {
		case ERROR:
			j.log.Error("%-25s: %s: %s", action, file, c)
		case UPLOADED:
			j.log.OK("%-25s: %s: %s", action, file, c)
		default:
			j.log.Info("%-25s: %s: %s", action, file, c)
		}
	}
	j.Lock()
	defer j.Unlock()
	e := j.files[file]

	switch action {
	case DISCARDED, UPGRADED, UPLOADED, LOCAL_DUPLICATE, SERVER_DUPLICATE, SERVER_BETTER, FAILED_VIDEO, UNSUPPORTED, METADATA, ERROR:
		if e.terminated {
			return
		}
		e.terminated = true
	}
	e.entries = append(e.entries, Entry{ts: time.Now(), action: action, comment: c})
	j.files[file] = e
}

func (j *Journal) Counters() map[Action]int {
	counts := map[Action]int{}
	terminated := 0

	for _, es := range j.files {
		for _, e := range es.entries {
			counts[e.action]++
		}
		if es.terminated {
			terminated++
		}
	}
	counts[HANDLED] = terminated
	counts[UNHANDLED] = len(j.files) - terminated
	return counts
}

func (j *Journal) Report() {
	counts := j.Counters()

	j.log.OK("Upload report:")
	j.log.OK("%6d scanned files", len(j.files))
	j.log.OK("%6d handled files", counts[HANDLED])
	j.log.OK("%6d metadata files", counts[METADATA])
	j.log.OK("%6d uploaded files on the server", counts[UPLOADED])
	j.log.OK("%6d upgraded files on the server", counts[UPGRADED])
	j.log.OK("%6d duplicated files in the input", counts[LOCAL_DUPLICATE])
	j.log.OK("%6d files already on the server", counts[SERVER_DUPLICATE])

	j.log.OK("%6d discarded files because in folder failed videos", counts[FAILED_VIDEO])
	j.log.OK("%6d discarded files because of options", counts[DISCARDED])
	j.log.OK("%6d discarded files because server has a better image", counts[SERVER_BETTER])
	j.log.OK("%6d files type not supported", counts[UNSUPPORTED])
	j.log.OK("%6d errors", counts[ERROR])
	j.log.OK("%6d files without metadata file", counts[UNHANDLED])

}

func (j *Journal) WriteJournal(events ...Action) {
	keys := gen.MapKeys(j.files)
	writeUnhandled := slices.Contains(events, UNHANDLED)
	sort.Strings(keys)
	for _, k := range keys {
		es := j.files[k]
		printFile := true
		mustTerminate := false
		for _, e := range es.entries {
			if slices.Contains(events, e.action) || (writeUnhandled && !es.terminated) {
				mustTerminate = true
				if printFile {
					j.log.OK("File: %s", k)
					printFile = false
				}
			}
			if slices.Contains(events, e.action) {
				j.log.MessageContinue(logger.OK, "\t%s", e.action)
				if len(e.comment) > 0 {
					j.log.MessageContinue(logger.OK, ", %s", e.comment)
				}
			}
		}
		if writeUnhandled && !es.terminated {
			j.log.MessageContinue(logger.OK, "\t%s, missing JSON", UNHANDLED)
		}
		if mustTerminate {
			j.log.MessageTerminate(logger.OK, "")
		}
	}
}
