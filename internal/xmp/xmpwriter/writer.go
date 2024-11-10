package xmpwriter

import (
	"encoding/xml"
	"io"

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
			{Name: xml.Name{Local: "xmlns:immichgo"}, Value: "http://ns.immich-go.com/immich-go/1.0/"},
		},
		Xmptk: application.GetVersion(),
	}
}

// <rdf:RDF xmlns:rdf='http://www.w3.org/1999/02/22-rdf-syntax-ns#'>
type RDF struct {
	XMLName      xml.Name `xml:"rdf:RDF"`
	Descriptions []Descriptioner
}

type RDFDescription struct {
	XMLName  xml.Name        `xml:"rdf:Description"`
	About    string          `xml:"rdf:about,attr"`
	Sections []Descriptioner `xml:",any"`
}

type Descriptioner interface {
	isDescription() bool
}

/*
type DCDescription struct {
	XMLName     xml.Name `xml:"rdf:Description"`
	Description string   `xml:"dc:description>rdf:Alt>rdf:Li"`
}

func (DCDescription) isDescription() bool { return true }

type TIFFDescription struct {
	XMLName     xml.Name `xml:"rdf:Description"`
	Description string   `xml:"tiff:ImageDescription>rdf:Alt>rdf:Li"`
}

func (TIFFDescription) isDescription() bool { return true }

type TagList struct {
	XMLName xml.Name `xml:"rdf:Description"`
	Seq     Seq      `xml:"digikam:TagsList>rdf:Seq"`
}

func (TagList) isDescription() bool { return true }
*/

type ImmichGoProperties struct {
	XMLName          xml.Name      `xml:"rdf:Description"`
	Title            string        `xml:"immichgo:ImmichGoProperties>immichgo:title,omitempty"`
	DateTimeOriginal string        `xml:"immichgo:ImmichGoProperties>immichgo:DateTimeOriginal,omitempty"`
	Trashed          string        `xml:"immichgo:ImmichGoProperties>immichgo:trashed,omitempty"`
	Archived         string        `xml:"immichgo:ImmichGoProperties>immichgo:archived,omitempty"`
	FromPartner      string        `xml:"immichgo:ImmichGoProperties>immichgo:fromPartner,omitempty"`
	Favorite         string        `xml:"immichgo:ImmichGoProperties>immichgo:favorite,omitempty"`
	Rating           string        `xml:"immichgo:ImmichGoProperties>immichgo:rating,omitempty"`
	Latitude         string        `xml:"immichgo:ImmichGoProperties>immichgo:latitude,omitempty"`
	Longitude        string        `xml:"immichgo:ImmichGoProperties>immichgo:longitude,omitempty"`
	Albums           *ImmichAlbums `xml:"immichgo:ImmichGoProperties>immichgo:albums,omitempty"`
	Tags             *ImmichTags   `xml:"immichgo:ImmichGoProperties>immichgo:tags,omitempty"`
}

func (ImmichGoProperties) isDescription() bool { return true }

type ImmichTags struct {
	Tags Bag
}

type ImmichTag struct {
	XMLName xml.Name `xml:"rdf:Li"`
	Name    string   `xml:"immichgo:tag>immichgo:name,omitempty"`
	Value   string   `xml:"immichgo:tag>immichgo:value,omitempty"`
}

func (ImmichTag) isLi() bool { return true }

type ImmichAlbums struct {
	Albums Bag
}

type ImmichAlbum struct {
	XMLName     xml.Name `xml:"rdf:Li"`
	Title       string   `xml:"immichgo:album>immichgo:title,omitempty"`
	Description string   `xml:"immichgo:album>immichgo:description,omitempty"`
	Latitude    string   `xml:"immichgo:album>immichgo:latitude,omitempty"`
	Longitude   string   `xml:"immichgo:album>immichgo:longitude,omitempty"`
}

func (ImmichAlbum) isLi() bool { return true }

type Bag struct {
	XMLName xml.Name `xml:"rdf:Bag,omitempty"`
	Li      []lier
}

type Seq struct {
	XMLName xml.Name `xml:"rdf:Seq"`
	Li      []lier
}

type lier interface {
	isLi() bool
}

type Li struct {
	XMLName xml.Name `xml:"rdf:li,omitempty"`
	Lang    string   `xml:"xml:lang,attr,omitempty"`
	Value   string   `xml:",chardata"`
}

func (Li) isLi() bool { return true }

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
	prop := ImmichGoProperties{
		Title:       a.Title,
		Trashed:     convert.BoolToString(a.Trashed),
		Archived:    convert.BoolToString(a.Archived),
		FromPartner: convert.BoolToString(a.FromPartner),
		Favorite:    convert.BoolToString(a.Favorite),
		Rating:      convert.IntToString(a.Rating),
	}
	if !a.CaptureDate.IsZero() {
		prop.DateTimeOriginal = convert.TimeToString(a.CaptureDate)
	}
	if a.Latitude != 0 || a.Longitude != 0 {
		prop.Latitude = convert.GPSFloatToString(a.Latitude, true)
		prop.Longitude = convert.GPSFloatToString(a.Longitude, false)
	}

	if len(a.Albums) > 0 {
		prop.Albums = &ImmichAlbums{
			Albums: Bag{
				Li: []lier{},
			},
		}
		for _, album := range a.Albums {
			al := ImmichAlbum{
				Title:       album.Title,
				Description: album.Description,
			}
			if album.Latitude != 0 || album.Longitude != 0 {
				al.Latitude = convert.GPSFloatToString(album.Latitude, true)
				al.Longitude = convert.GPSFloatToString(album.Longitude, false)
			}
			prop.Albums.Albums.Li = append(prop.Albums.Albums.Li, al)
		}
	}
	if len(a.Tags) > 0 {
		prop.Tags = &ImmichTags{
			Tags: Bag{
				Li: []lier{},
			},
		}
		for _, tag := range a.Tags {
			prop.Tags.Tags.Li = append(prop.Tags.Tags.Li, ImmichTag{Name: tag.Name, Value: tag.Value})
		}
	}

	xmp := NewXMP()
	xmp.RDF.Descriptions = append(xmp.RDF.Descriptions, prop)
	return xmp.encode(w)
}
