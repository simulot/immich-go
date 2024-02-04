package metadata

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"text/template"
	"time"
)

// SideCar
type SideCar struct {
	FileName string
	OnFSsys  bool

	DateTaken time.Time
	Latitude  float64
	Longitude float64
	Elevation float64
}

func (sc *SideCar) Open(fsys fs.FS, name string) (io.ReadCloser, error) {
	if sc.OnFSsys {
		return fsys.Open(name)
	}

	b := bytes.NewBuffer(nil)
	err := sidecarTemplate.Execute(b, sc)
	if err != nil {
		return nil, fmt.Errorf("can't generate XMP sidecar file: %w", err)
	}

	return io.NopCloser(b), nil
}

func (sc *SideCar) Bytes() ([]byte, error) {
	b := bytes.NewBuffer(nil)
	err := sidecarTemplate.Execute(b, sc)
	if err != nil {
		return nil, fmt.Errorf("can't generate XMP sidecar file: %w", err)
	}
	return b.Bytes(), nil
}

var sidecarTemplate = template.Must(template.New("xmp").Parse(`<x:xmpmeta xmlns:x='adobe:ns:meta/' x:xmptk='Image::ExifTool 12.56'>
<rdf:RDF xmlns:rdf='http://www.w3.org/1999/02/22-rdf-syntax-ns#'>
 <rdf:Description rdf:about=''
  xmlns:exif='http://ns.adobe.com/exif/1.0/'>
  <exif:ExifVersion>0232</exif:ExifVersion>
  <exif:DateTimeOriginal>{{((.DateTaken).Local).Format "2006-01-02T15:04:05"}}</exif:DateTimeOriginal>
  <exif:GPSAltitude>{{.Elevation}}</exif:GPSAltitude>
  <exif:GPSLatitude>{{.Latitude}}</exif:GPSLatitude>
  <exif:GPSLongitude>{{.Longitude}}</exif:GPSLongitude>  
  <exif:GPSTimeStamp>{{((.DateTaken).UTC).Format "2006-01-02T15:04:05+0000"}}</exif:GPSTimeStamp>
 </rdf:Description>
</rdf:RDF>
</x:xmpmeta>`))
