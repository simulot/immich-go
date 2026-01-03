package xmpsidecar

import (
	"html"
	"io"
	"text/template"

	"github.com/simulot/immich-go/internal/assets"
)

// xmpData holds the prepared data for XMP template rendering
type xmpData struct {
	Description  string
	DateTaken    string
	Rating       int
	GPSLatitude  string
	GPSLongitude string
	TagsList     []string
}

func Write(md *assets.Metadata, w io.Writer) error {
	data := prepareXMPData(md)
	return xmpTemplate.Execute(w, data)
}

func prepareXMPData(md *assets.Metadata) xmpData {
	data := xmpData{}

	if md.Description != "" {
		data.Description = html.EscapeString(md.Description)
	}

	if !md.DateTaken.IsZero() {
		data.DateTaken = TimeToString(md.DateTaken)
	}

	// Rating - if Favorited and no explicit rating, set to 5
	rating := int(md.Rating)
	if md.Favorited && rating == 0 {
		rating = 5
	}
	data.Rating = rating

	if md.Latitude != 0 {
		data.GPSLatitude = GPSFloatToString(md.Latitude, true)
	}
	if md.Longitude != 0 {
		data.GPSLongitude = GPSFloatToString(md.Longitude, false)
	}

	// Combine tags and albums into TagsList
	// Tags are stored as-is, albums are prefixed with "Albums/"
	for _, tag := range md.Tags {
		if tag.Value != "" {
			data.TagsList = append(data.TagsList, html.EscapeString(tag.Value))
		}
	}
	for _, album := range md.Albums {
		if album.Title != "" {
			data.TagsList = append(data.TagsList, "Albums/"+html.EscapeString(album.Title))
		}
	}

	return data
}

// hasExifBlock returns true if any EXIF fields are present
func (d xmpData) HasExifBlock() bool {
	return d.DateTaken != "" || d.GPSLatitude != "" || d.GPSLongitude != ""
}

// HasTagsList returns true if there are any tags or albums
func (d xmpData) HasTagsList() bool {
	return len(d.TagsList) > 0
}

// HasDescription returns true if description is set
func (d xmpData) HasDescription() bool {
	return d.Description != ""
}

// HasRating returns true if rating is set (> 0)
func (d xmpData) HasRating() bool {
	return d.Rating > 0
}

const xmpTemplateText = `<?xpacket begin='' id='W5M0MpCehiHzreSzNTczkc9d'?>
<x:xmpmeta xmlns:x='adobe:ns:meta/' x:xmptk='immich-go'>
<rdf:RDF xmlns:rdf='http://www.w3.org/1999/02/22-rdf-syntax-ns#'>
{{if .HasDescription}}
 <rdf:Description rdf:about=''
  xmlns:dc='http://purl.org/dc/elements/1.1/'>
  <dc:description>
   <rdf:Alt>
    <rdf:li xml:lang='x-default'>{{.Description}}</rdf:li>
   </rdf:Alt>
  </dc:description>
 </rdf:Description>

 <rdf:Description rdf:about=''
  xmlns:tiff='http://ns.adobe.com/tiff/1.0/'>
  <tiff:ImageDescription>
   <rdf:Alt>
    <rdf:li xml:lang='x-default'>{{.Description}}</rdf:li>
   </rdf:Alt>
  </tiff:ImageDescription>
 </rdf:Description>
{{end}}{{if .HasTagsList}}
 <rdf:Description rdf:about=''
  xmlns:digiKam='http://www.digikam.org/ns/1.0/'>
  <digiKam:TagsList>
   <rdf:Seq>
{{range .TagsList}}    <rdf:li>{{.}}</rdf:li>
{{end}}   </rdf:Seq>
  </digiKam:TagsList>
 </rdf:Description>
{{end}}{{if .HasExifBlock}}
 <rdf:Description rdf:about=''
  xmlns:exif='http://ns.adobe.com/exif/1.0/'>
{{if .DateTaken}}  <exif:DateTimeOriginal>{{.DateTaken}}</exif:DateTimeOriginal>
{{end}}{{if .GPSLatitude}}  <exif:GPSLatitude>{{.GPSLatitude}}</exif:GPSLatitude>
{{end}}{{if .GPSLongitude}}  <exif:GPSLongitude>{{.GPSLongitude}}</exif:GPSLongitude>
{{end}} </rdf:Description>
{{end}}{{if .HasRating}}
 <rdf:Description rdf:about=''
  xmlns:xmp='http://ns.adobe.com/xap/1.0/'>
  <xmp:Rating>{{.Rating}}</xmp:Rating>
 </rdf:Description>
{{end}}</rdf:RDF>
</x:xmpmeta>
<?xpacket end='w'?>
`

var xmpTemplate = template.Must(template.New("xmp").Parse(xmpTemplateText))
