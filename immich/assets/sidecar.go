package assets

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"io/fs"
	"time"
)

// SideCarMetadata
type SideCarMetadata struct {
	FileName string
	OnFSsys  bool

	DateTake  *time.Time
	Latitude  *float64
	Longitude *float64
	Elevation *float64
}

func (sc *SideCarMetadata) Open(fsys fs.FS, name string) (io.ReadCloser, error) {
	if sc.OnFSsys {
		return fsys.Open(name)
	}

	b := bytes.NewBuffer(nil)
	xmp := XMPMetadata{}
	if sc.DateTake != nil {
		xmp.CreateRDF.CreateDate = sc.DateTake.Format("2006-01-02T15:04:05")
	}
	if sc.Latitude != nil {
		xmp.GPSRDF.GPSLatitude = fmt.Sprintf("%f", *sc.Latitude)
	}
	if sc.Longitude != nil {
		xmp.GPSRDF.GPSLongitude = fmt.Sprintf("%f", *sc.Longitude)
	}
	if sc.Elevation != nil {
		xmp.GPSRDF.GPSAltitude = fmt.Sprintf("%f", *sc.Elevation)
	}
	err := xml.NewEncoder(b).Encode(xmp)
	if err != nil {
		return nil, fmt.Errorf("can't generate XMP sidecar file: %w", err)
	}

	return io.NopCloser(b), nil

}

type XMPMetadata struct {
	XMLName   xml.Name `xml:"x:xmpmeta"`
	XmpTk     string   `xml:"x:xmptk,attr"`
	CreateRDF RDF      `xml:"rdf:RDF"`
	GPSRDF    RDF      `xml:"rdf:RDF"`
}

type RDF struct {
	About        string `xml:"rdf:Description>about,attr"`
	Xmp          string `xml:"xmlns:xmp,attr"`
	Exif         string `xml:"xmlns:exif,attr"`
	CreateDate   string `xml:"Description>exif:CreateDate,omitempty"`
	GPSLatitude  string `xml:"Description>exif:GPSLatitude,omitempty"`
	GPSLongitude string `xml:"Description>exif:GPSLongitude,omitempty"`
	GPSAltitude  string `xml:"Description>exif:GPSAltitude,omitempty"`
	GPSAltRef    string `xml:"Description>exif:GPSAltitudeRef,omitempty"`
}
