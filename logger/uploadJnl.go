package logger

type UpLdAction int

const (
	UpldDiscoveredFile  UpLdAction = iota // "File"
	UpldScannedImage                      // "Scanned image"
	UpldScannedVideo                      // "Scanned video"
	UpldDiscarded                         // "Discarded"
	UpldUploaded                          // "Uploaded"
	UpldUpgraded                          // "Server's asset upgraded"
	UpldERROR                             // "Error"
	UpldLocalDuplicate                    // "Local duplicate"
	UpldServerDuplicate                   // "Server has photo"
	UpldStacked                           // "Stacked"
	UpldServerBetter                      // "Server's asset is better"
	UpldAlbum                             // "Added to an album"
	// UpldLivePhoto                            // "Live photo"
	// UpldFailedVideo                          // "Failed video"
	// UpldUnsupported                          // "File type not supported"
	UpldMetadata           // "Metadata files"
	UpldAssociatedMetadata // "Associated with metadata"
	UpldINFO               // "Info"
	// UpldNotSelected                          // "Not selected because of options"
	UpldServerError // "Server error"
	UpldReceived    // "Asset received from the server",
)

var _uploadActionStrings = map[UpLdAction]string{
	UpldDiscoveredFile:  "File discovered",
	UpldScannedImage:    "Scanned image",
	UpldScannedVideo:    "Scanned video",
	UpldDiscarded:       "Discarded",
	UpldUploaded:        "Uploaded",
	UpldUpgraded:        "Server's asset upgraded",
	UpldERROR:           "Error",
	UpldLocalDuplicate:  "Local duplicate",
	UpldServerDuplicate: "Server has photo",
	UpldStacked:         "Stacked",
	UpldServerBetter:    "Server's asset is better",
	UpldAlbum:           "Added to an album",
	// UpldLivePhoto:          "Live photo",
	// UpldFailedVideo:        "Failed video",
	// UpldUnsupported:        "File type not supported",
	UpldMetadata:           "Metadata files",
	UpldAssociatedMetadata: "Associated with metadata",
	UpldINFO:               "Info",
	// UpldNotSelected:        "Not selected because of options",
	UpldServerError: "Server error",
	UpldReceived:    "Asset received from the server",
}

func (m UpLdAction) String() string {
	return _uploadActionStrings[m]
}
