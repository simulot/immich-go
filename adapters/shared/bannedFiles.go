package shared

var DefaultBannedFiles = []string{
	`@eaDir/`,
	`@__thumb/`,          // QNAP
	`SYNOFILE_THUMB_*.*`, // SYNOLOGY
	`Lightroom Catalog/`, // LR
	`thumbnails/`,        // Android photo
	`.DS_Store/`,         // Mac OS custom attributes
	`/._*`,               // MacOS resource files
	`.photostructure/`,   // PhotoStructure
	`Recently Deleted/`,  // ICloud recently deleted

}
