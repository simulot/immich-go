package immich

import (
	"context"
	"io/fs"
	"os"
	"time"
)

type Logger interface {
	Info(string)
	Error(string)
}

type stopFn func()

type assetBrowse interface {
	browse(context.Context, indexerOptions, []fs.FS) (chan *localFile, stopFn)
}

type AssetUploader struct {
	options indexerOptions
}

type localFile struct {
	srcFS     fs.FS
	file      fs.File
	name      string
	dateTaken *time.Time
	tempFile  *os.File
}

// IndexerOptionsFn is a function to change Indexer parameters
type IndexerOptionsFn func(*indexerOptions)

// Indexer options
type indexerOptions struct {
	log           Logger
	createAlbums  bool      // CLI flag fore album creation
	dateRange     DateRange // Accepted range of capture date
	fileterAlbums []string  // List of albums we want
	uploadInAlbum string    // Upload assets in this album
}
