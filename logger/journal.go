package logger

import (
	"strings"
	"sync"
)

type Journal struct {
	mut    sync.Mutex
	counts map[Action]int
	Logger
}

type Action string

const (
	DISCOVERED_FILE  Action = "File"
	SCANNED_IMAGE    Action = "Scanned image"
	SCANNED_VIDEO    Action = "Scanned video"
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
	ASSOCIATED_META  Action = "Associated with metadata"
	INFO             Action = "Info"
	NOT_SELECTED     Action = "Not selected because options"
	SERVER_ERROR     Action = "Server error"
)

func NewJournal(log Logger) *Journal {
	return &Journal{
		// files:  map[string]Entries{},
		Logger: log,
		counts: map[Action]int{},
	}
}

func (j *Journal) AddEntry(file string, action Action, comment ...string) {
	if j == nil {
		return
	}
	c := strings.Join(comment, ", ")
	if j.Logger != nil {
		switch action {
		case ERROR, SERVER_ERROR:
			j.Logger.Error("%-25s: %s: %s", action, file, c)
		case DISCOVERED_FILE:
			j.Logger.Debug("%-25s: %s: %s", action, file, c)
		case UPLOADED:
			j.Logger.OK("%-25s: %s: %s", action, file, c)
		default:
			j.Logger.Info("%-25s: %s: %s", action, file, c)
		}
	}
	j.mut.Lock()
	j.counts[action]++
	if action == UPGRADED {
		j.counts[UPLOADED]--
	}
	j.mut.Unlock()
}

func (j *Journal) Report() {
	checkFiles := j.counts[SCANNED_IMAGE] + j.counts[SCANNED_VIDEO] + j.counts[METADATA] + j.counts[UNSUPPORTED] + j.counts[FAILED_VIDEO] + j.counts[DISCARDED]
	handledFiles := j.counts[NOT_SELECTED] + j.counts[LOCAL_DUPLICATE] + j.counts[SERVER_DUPLICATE] + j.counts[SERVER_BETTER] + j.counts[UPLOADED] + j.counts[UPGRADED] + j.counts[SERVER_ERROR]
	j.Logger.OK("Scan of the sources:")
	j.Logger.OK("%6d files in the input", j.counts[DISCOVERED_FILE])
	j.Logger.OK("--------------------------------------------------------")
	j.Logger.OK("%6d photos", j.counts[SCANNED_IMAGE])
	j.Logger.OK("%6d videos", j.counts[SCANNED_VIDEO])
	j.Logger.OK("%6d metadata files", j.counts[METADATA])
	j.Logger.OK("%6d files with metadata", j.counts[ASSOCIATED_META])
	j.Logger.OK("%6d discarded files", j.counts[DISCARDED])
	j.Logger.OK("%6d files having a type not supported", j.counts[UNSUPPORTED])
	j.Logger.OK("%6d discarded files because in folder failed videos", j.counts[FAILED_VIDEO])

	j.Logger.OK("%6d input total (difference %d)", checkFiles, j.counts[DISCOVERED_FILE]-checkFiles)
	j.Logger.OK("--------------------------------------------------------")

	j.Logger.OK("%6d uploaded files on the server", j.counts[UPLOADED])
	j.Logger.OK("%6d upgraded files on the server", j.counts[UPGRADED])
	j.Logger.OK("%6d files already on the server", j.counts[SERVER_DUPLICATE])
	j.Logger.OK("%6d discarded files because of options", j.counts[NOT_SELECTED])
	j.Logger.OK("%6d discarded files because duplicated in the input", j.counts[LOCAL_DUPLICATE])
	j.Logger.OK("%6d discarded files because server has a better image", j.counts[SERVER_BETTER])
	j.Logger.OK("%6d errors when uploading", j.counts[SERVER_ERROR])

	j.Logger.OK("%6d handled total (difference %d)", handledFiles, j.counts[SCANNED_IMAGE]+j.counts[SCANNED_VIDEO]-handledFiles)
}
