package journal

import (
	"strings"
	"sync"
)

type Journal struct {
	mut    sync.Mutex
	counts map[Action]int
	Log    Logger
}

type Action string

const (
	DiscoveredFile     Action = "File"
	ScannedImage       Action = "Scanned image"
	ScannedVideo       Action = "Scanned video"
	Discarded          Action = "Discarded"
	Uploaded           Action = "Uploaded"
	Upgraded           Action = "Server's asset upgraded"
	ERROR              Action = "Error"
	LocalDuplicate     Action = "Local duplicate"
	ServerDuplicate    Action = "Server has photo"
	Stacked            Action = "Stacked"
	ServerBetter       Action = "Server's asset is better"
	Album              Action = "Added to an album"
	LivePhoto          Action = "Live photo"
	FailedVideo        Action = "Failed video"
	Unsupported        Action = "File type not supported"
	Metadata           Action = "Metadata files"
	AssociatedMetadata Action = "Associated with metadata"
	INFO               Action = "Info"
	NotSelected        Action = "Not selected because options"
	ServerError        Action = "Server error"
)

func NewJournal(log Logger) *Journal {
	return &Journal{
		// files:  map[string]Entries{},
		Log:    log,
		counts: map[Action]int{},
	}
}

func (j *Journal) AddEntry(file string, action Action, comment ...string) {
	if j == nil {
		return
	}
	c := strings.Join(comment, ", ")
	if j.Log != nil {
		switch action {
		case ERROR, ServerError:
			j.Log.Error("%-25s: %s: %s", action, file, c)
		case DiscoveredFile:
			j.Log.Debug("%-25s: %s: %s", action, file, c)
		case Uploaded:
			j.Log.OK("%-25s: %s: %s", action, file, c)
		default:
			j.Log.Info("%-25s: %s: %s", action, file, c)
		}
	}
	j.mut.Lock()
	j.counts[action]++
	if action == Upgraded {
		j.counts[Uploaded]--
	}
	j.mut.Unlock()
}

func (j *Journal) Report() {
	checkFiles := j.counts[ScannedImage] + j.counts[ScannedVideo] + j.counts[Metadata] + j.counts[Unsupported] + j.counts[FailedVideo] + j.counts[Discarded]
	handledFiles := j.counts[NotSelected] + j.counts[LocalDuplicate] + j.counts[ServerDuplicate] + j.counts[ServerBetter] + j.counts[Uploaded] + j.counts[Upgraded] + j.counts[ServerError]
	j.Log.OK("Scan of the sources:")
	j.Log.OK("%6d files in the input", j.counts[DiscoveredFile])
	j.Log.OK("--------------------------------------------------------")
	j.Log.OK("%6d photos", j.counts[ScannedImage])
	j.Log.OK("%6d videos", j.counts[ScannedVideo])
	j.Log.OK("%6d metadata files", j.counts[Metadata])
	j.Log.OK("%6d files with metadata", j.counts[AssociatedMetadata])
	j.Log.OK("%6d discarded files", j.counts[Discarded])
	j.Log.OK("%6d files having a type not supported", j.counts[Unsupported])
	j.Log.OK("%6d discarded files because in folder failed videos", j.counts[FailedVideo])

	j.Log.OK("%6d input total (difference %d)", checkFiles, j.counts[DiscoveredFile]-checkFiles)
	j.Log.OK("--------------------------------------------------------")

	j.Log.OK("%6d uploaded files on the server", j.counts[Uploaded])
	j.Log.OK("%6d upgraded files on the server", j.counts[Upgraded])
	j.Log.OK("%6d files already on the server", j.counts[ServerDuplicate])
	j.Log.OK("%6d discarded files because of options", j.counts[NotSelected])
	j.Log.OK("%6d discarded files because duplicated in the input", j.counts[LocalDuplicate])
	j.Log.OK("%6d discarded files because server has a better image", j.counts[ServerBetter])
	j.Log.OK("%6d errors when uploading", j.counts[ServerError])

	j.Log.OK("%6d handled total (difference %d)", handledFiles, j.counts[ScannedImage]+j.counts[ScannedVideo]-handledFiles)
}
