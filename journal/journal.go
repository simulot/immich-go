package journal

import (
	"strings"

	"github.com/simulot/immich-go/logger"
)

type Journal struct {
	counts map[Action]int
	log    logger.Logger
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
)

func NewJournal(log logger.Logger) *Journal {
	return &Journal{
		// files:  map[string]Entries{},
		log:    log,
		counts: map[Action]int{},
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
		case UPLOADED, SCANNED_IMAGE, SCANNED_VIDEO:
			j.log.OK("%-25s: %s: %s", action, file, c)
		default:
			j.log.Info("%-25s: %s: %s", action, file, c)
		}
	}
	j.counts[action] = j.counts[action] + 1
}
func (j *Journal) Report() {

	checkFiles := j.counts[SCANNED_IMAGE] + j.counts[SCANNED_VIDEO] + j.counts[METADATA] + j.counts[UNSUPPORTED] + j.counts[FAILED_VIDEO] + j.counts[ERROR] + j.counts[DISCARDED]

	j.log.OK("Scan of the sources:")
	j.log.OK("%6d files in the input", j.counts[DISCOVERED_FILE])
	j.log.OK("--------------------------------------------------------")
	j.log.OK("%6d photos", j.counts[SCANNED_IMAGE])
	j.log.OK("%6d videos", j.counts[SCANNED_VIDEO])
	j.log.OK("%6d metadata files", j.counts[METADATA])
	j.log.OK("%6d discarded files", j.counts[DISCARDED])
	j.log.OK("%6d files having a type not supported", j.counts[UNSUPPORTED])
	j.log.OK("%6d discarded files because in folder failed videos", j.counts[FAILED_VIDEO])
	j.log.OK("%6d errors", j.counts[ERROR])
	j.log.OK("%6d total (difference %d)", checkFiles, j.counts[DISCOVERED_FILE]-checkFiles)
	j.log.OK("--------------------------------------------------------")

	j.log.OK("%6d files with metadata", j.counts[ASSOCIATED_META])
	j.log.OK("%6d discarded files because duplicated in the input", j.counts[LOCAL_DUPLICATE])
	j.log.OK("%6d files already on the server", j.counts[SERVER_DUPLICATE])
	j.log.OK("%6d uploaded files on the server", j.counts[UPLOADED])
	j.log.OK("%6d upgraded files on the server", j.counts[UPGRADED])
	j.log.OK("%6d discarded files because server has a better image", j.counts[SERVER_BETTER])

}
