package logger

type UpLdAction int

const (
	UpldDiscoveredFile     UpLdAction = iota // "File"
	UpldScannedImage                         // "Scanned image"
	UpldScannedVideo                         // "Scanned video"
	UpldDiscarded                            // "Discarded"
	UpldUploaded                             // "Uploaded"
	UpldUpgraded                             // "Server's asset upgraded"
	UpldERROR                                // "Error"
	UpldLocalDuplicate                       // "Local duplicate"
	UpldServerDuplicate                      // "Server has photo"
	UpldStacked                              // "Stacked"
	UpldServerBetter                         // "Server's asset is better"
	UpldAlbum                                // "Added to an album"
	UpldMetadata                             // "Metadata files"
	UpldAssociatedMetadata                   // "Associated with metadata"
	UpldINFO                                 // "Info"
	UpldNotSelected                          // "Not selected because of options"
	UpldServerError                          // "Server error"
	UpldReceived                             // "Asset received from the server",
	UpldStack                                // "Stack assets"
	UpldCreateAlbum                          // "Create/Update album"
	UpldDeleteServerAssets                   //"Delete server's assets"
)

var _uploadActionStrings = map[UpLdAction]string{
	UpldDiscoveredFile:     "File discovered",
	UpldScannedImage:       "Scanned image",
	UpldScannedVideo:       "Scanned video",
	UpldDiscarded:          "Discarded",
	UpldUploaded:           "Uploaded",
	UpldUpgraded:           "Server's asset upgraded",
	UpldERROR:              "Error",
	UpldLocalDuplicate:     "Local duplicate",
	UpldServerDuplicate:    "Server has photo",
	UpldStacked:            "Stacked",
	UpldServerBetter:       "Server's asset is better",
	UpldAlbum:              "Added to an album",
	UpldMetadata:           "Metadata files",
	UpldAssociatedMetadata: "Associated with metadata",
	UpldINFO:               "Info",
	UpldNotSelected:        "Not selected because of options",
	UpldServerError:        "Server error",
	UpldReceived:           "Asset received from the server",
	UpldStack:              "Stack assets",
	UpldCreateAlbum:        "Create/Update album",
	UpldDeleteServerAssets: "Delete server's assets",
}

func (m UpLdAction) String() string {
	return _uploadActionStrings[m]
}
