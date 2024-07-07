package browser

type LocalAlbum struct {
	Path                string  // As found in the files
	Title               string  // either the directory base name, or metadata
	Description         string  // As found in the metadata
	Latitude, Longitude float64 // As found in the metadata
}
