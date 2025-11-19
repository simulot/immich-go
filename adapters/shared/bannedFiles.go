package shared

var DefaultBannedFiles = []string{
	`@eaDir/`,
	`@__thumb/`,          // QNAP
	`SYNOFILE_THUMB_*.*`, // SYNOLOGY
	`Lightroom Catalog/`, // LR
	`thumbnails/`,        // Android photo
	`.DS_Store`,          // macOS Finder metadata
	`/._*`,               // MacOS resource files
	`.Spotlight-V100/`,   // macOS system index
	`.photostructure/`,   // PhotoStructure
	`Recently Deleted/`,  // ICloud recently deleted

}
