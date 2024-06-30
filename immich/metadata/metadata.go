package metadata

import (
	"encoding/xml"
	"fmt"
	"io"
	"strings"
	"time"
)

type Metadata struct {
	Description string
	DateTaken   time.Time
	Latitude    float64
	Longitude   float64
	Altitude    float64
}

func (m Metadata) IsSet() bool {
	return m.Description != "" || !m.DateTaken.IsZero() || m.Latitude != 0 || m.Longitude != 0
}

func (m Metadata) Write(w io.Writer) error {
	_, err := io.WriteString(w, header)
	if err != nil {
		return err
	}
	if m.Description != "" {
		_, err = io.WriteString(w, descriptionHeader)
		if err != nil {
			return err
		}
		err = xml.EscapeText(w, []byte(m.Description))
		if err != nil {
			return err
		}
		_, err = io.WriteString(w, descriptionFooter)
		if err != nil {
			return err
		}
	}

	writeExifBlock := !m.DateTaken.IsZero() || m.Latitude != 0 || m.Longitude != 0
	if writeExifBlock {
		_, err = io.WriteString(w, exifHeader)
		if err != nil {
			return err
		}
		if !m.DateTaken.IsZero() {
			_, err := fmt.Fprintf(w, exifDateTimeOriginal, m.DateTaken.UTC().Format("2006-01-02T15:04:05Z"))
			if err != nil {
				return err
			}
		}
		if m.Latitude != 0 || m.Longitude != 0 {
			_, err = fmt.Fprintf(w, exifGPSLatitude, m.Latitude)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(w, exifGPSLongitude, m.Longitude)
			if err != nil {
				return err
			}
		}
		_, err = io.WriteString(w, exifFooter)
		if err != nil {
			return err
		}
	}
	_, err = io.WriteString(w, footer)
	return err
}

func (m Metadata) String() string {
	s := strings.Builder{}
	_ = m.Write(&s)
	return s.String()
}

const (
	header = `<?xpacket begin='?' id='W5M0MpCehiHzreSzNTczkc9d'?>
<x:xmpmeta xmlns:x='adobe:ns:meta/' x:xmptk='Image::ExifTool 12.40'>
<rdf:RDF xmlns:rdf='http://www.w3.org/1999/02/22-rdf-syntax-ns#'>
`
	descriptionHeader = ` <rdf:Description rdf:about=''
  xmlns:dc='http://purl.org/dc/elements/1.1/'>
  <dc:description>
   <rdf:Alt>
    <rdf:li xml:lang='x-default'>`

	descriptionFooter = `</rdf:li>
   </rdf:Alt>
  </dc:description>
 </rdf:Description>
`

	exifHeader = ` <rdf:Description rdf:about=''
  xmlns:exif='http://ns.adobe.com/exif/1.0/'>
  <exif:ExifVersion>0220</exif:ExifVersion>`

	exifDateTimeOriginal = `  <exif:DateTimeOriginal>%s</exif:DateTimeOriginal>
`
	exifGPSAltitude = `  <exif:GPSAltitude>0</exif:GPSAltitude>
`
	exifGPSLatitude = `  <exif:GPSLatitude>%f</exif:GPSLatitude>
`
	exifGPSLongitude = `  <exif:GPSLongitude>%f</exif:GPSLongitude>
`
	exifFooter = `  <exif:GPSVersionID>2.3.0.0</exif:GPSVersionID>
 </rdf:Description>
`
	footer = `</rdf:RDF>
</x:xmpmeta>
<?xpacket end='w'?>`
)
