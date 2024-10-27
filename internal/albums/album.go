package albums

type Album struct {
	Title               string  // either the directory base name, or metadata
	Description         string  // As found in the metadata
	Latitude, Longitude float64 // As found in the metadata
}
