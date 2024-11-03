package xmpwriter

import (
	"encoding/xml"
	"io"
	"strconv"

	"github.com/simulot/immich-go/commands/application"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/xmp/convert"
)

// <x:xmpmeta xmlns:x='adobe:ns:meta/' x:xmptk='Image::ExifTool 12.97'>
type XmpMeta struct {
	XMLName xml.Name   `xml:"x:xmpmeta"`
	NS      []xml.Attr `xml:",any,attr"`
	Xmptk   string     `xml:"x:xmptk,attr"`
	RDF     RDF
}

func NewXMP() *XmpMeta {
	return &XmpMeta{
		XMLName: xml.Name{Space: "adobe:ns:meta/", Local: "xmpmeta"},
		NS: []xml.Attr{
			{Name: xml.Name{Local: "xmlns:x"}, Value: "adobe:ns:meta/"},
			{Name: xml.Name{Local: "xmlns:rdf"}, Value: "http://www.w3.org/1999/02/22-rdf-syntax-ns#"},
			{Name: xml.Name{Local: "xmlns:dc"}, Value: "http://purl.org/dc/elements/1.1/"},
			{Name: xml.Name{Local: "xmlns:exif"}, Value: "http://ns.adobe.com/exif/1.0/"},
			{Name: xml.Name{Local: "xmlns:xmp"}, Value: "http://ns.adobe.com/xap/1.0/"},
			{Name: xml.Name{Local: "xmlns:tiff"}, Value: "http://ns.adobe.com/tiff/1.0/"},
			{Name: xml.Name{Local: "xmlns:digikam"}, Value: "http://www.digikam.org/ns/1.0/"},
		},
		Xmptk: application.GetVersion(),
	}
}

// <rdf:RDF xmlns:rdf='http://www.w3.org/1999/02/22-rdf-syntax-ns#'>
type RDF struct {
	XMLName      xml.Name `xml:"rdf:RDF"`
	Descriptions []Description
}

type Description struct {
	XMLName          xml.Name  `xml:"rdf:Description"`
	About            string    `xml:"rdf:about,attr"`
	DescriptionA     *Alt      `xml:"dc:description>rdf:Alt,omitempty"`
	DescriptionB     *Alt      `xml:"tiff:ImageDescription>rdf:Alt,omitempty"`
	TagsList         *TagList  `xml:"digikam:TagsList,omitempty"`
	DateTimeOriginal string    `xml:"exif:DateTimeOriginal,omitempty"`
	GPSLatitude      string    `xml:"exif:GPSLatitude,omitempty"`
	GPSLongitude     string    `xml:"exif:GPSLongitude,omitempty"`
	Rating           string    `xml:"xmp:Rating,omitempty"`
	Albums           *Relation `xml:"dc:relation>rdf:Bag,omitempty"`
}

type Alt struct {
	Items []Li
}

type Seq struct {
	XMLName xml.Name `xml:"rdf:Seq"`
	Li      []Li
}

type Relation struct {
	// XMLName xml.Name `xml:"dc:relation"`
	Li []Li
}

type TagList struct {
	Seq Seq
}

type Li struct {
	XMLName xml.Name `xml:"rdf:li"`
	Lang    string   `xml:"xml:lang,attr,omitempty"`
	Value   string   `xml:",chardata"`
}

func (xmp *XmpMeta) addImageDescription(description string) {
	xmp.RDF.Descriptions = append(xmp.RDF.Descriptions, Description{
		DescriptionA: &Alt{
			Items: []Li{
				{Lang: "x-default", Value: description},
			},
		},
		DescriptionB: &Alt{
			Items: []Li{
				{Lang: "x-default", Value: description},
			},
		},
	})
}

func (xmp *XmpMeta) addRating(rating int) {
	if rating < 0 || rating > 5 {
		return
	}

	xmp.RDF.Descriptions = append(xmp.RDF.Descriptions, Description{
		Rating: strconv.Itoa(rating),
	})
}

func (xmp *XmpMeta) addTag(tags []string) {
	if len(tags) == 0 {
		return
	}

	seq := Seq{}

	for _, tag := range tags {
		seq.Li = append(seq.Li, Li{Value: tag})
	}

	xmp.RDF.Descriptions = append(xmp.RDF.Descriptions, Description{
		TagsList: &TagList{
			Seq: seq,
		},
	})
}

func (xmp *XmpMeta) addExif(dateTimeOriginal string, gpsLatitude, gpsLongitude string) {
	emit := false

	d := Description{}

	if dateTimeOriginal != "" {
		d.DateTimeOriginal = dateTimeOriginal
		emit = true
	}
	if gpsLatitude != "" && gpsLongitude != "" {
		d.GPSLatitude = gpsLatitude
		d.GPSLongitude = gpsLongitude
		emit = true
	}

	if emit {
		xmp.RDF.Descriptions = append(xmp.RDF.Descriptions, d)
	}
}

func (xmp *XmpMeta) addAlbums(albums []assets.Album) {
	if len(albums) == 0 {
		return
	}
	d := Description{
		Albums: &Relation{
			Li: []Li{},
		},
	}
	for _, album := range albums {
		d.Albums.Li = append(d.Albums.Li, Li{Value: album.Title})
	}
	xmp.RDF.Descriptions = append(xmp.RDF.Descriptions, d)
}

func (xmp *XmpMeta) encode(w io.Writer) error {
	_, err := io.WriteString(w, "<?xpacket begin='\xEF\xBB\xBF' id='W5M0MpCehiHzreSzNTczkc9d'?>\n")
	if err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	err = enc.Encode(xmp)
	if err != nil {
		return err
	}
	_, err = io.WriteString(w, "<?xpacket end='w'?>\n")
	return err
}

func WriteXMP(a *assets.Asset, w io.Writer) error {
	xmp := NewXMP()
	if a.Title != "" {
		xmp.addImageDescription(a.Title)
	}
	if a.Stars != 0 {
		xmp.addRating(a.Stars)
	}

	if len(a.Albums) > 0 {
		xmp.addAlbums(a.Albums)
	}

	// xmp.AddTag(a.Tags)

	// gps data
	captureDate := ""
	if !a.CaptureDate.IsZero() {
		captureDate = convert.TimeToString(a.CaptureDate)
	}
	latitude, longitude := "", ""
	if a.Latitude != 0 || a.Longitude != 0 {
		latitude = convert.GPSFloatToString(a.Latitude, true)
		longitude = convert.GPSFloatToString(a.Longitude, false)
	}
	if latitude != "" || longitude != "" || captureDate != "" {
		xmp.addExif(captureDate, latitude, longitude)
	}
	return xmp.encode(w)
}
