package xmpsidecar

import (
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
	"time"

	"github.com/clbanning/mxj/v2"
	"github.com/simulot/immich-go/internal/assets"
)

func ReadXMP(r io.Reader, md *assets.Metadata) error {
	// Read the XMP data from the reader and return an Asset
	m, err := mxj.NewMapXmlReader(r)
	if err != nil {
		return err
	}
	walk(m, md, "")
	return nil
}

func walk(m mxj.Map, md *assets.Metadata, path string) {
	for key, value := range m {
		switch v := value.(type) {
		case map[string]interface{}:
			walk(v, md, path+"/"+key)
		case []interface{}:
			listPath := path + "/" + key
			for i, item := range v {
				p := fmt.Sprintf("%s[%d]", listPath, i)
				if itemMap, ok := item.(map[string]interface{}); ok {
					walk(itemMap, md, p)
				} else if s, ok := item.(string); ok {
					filter(md, p, s)
				}
			}
		case string:
			filter(md, path+"/"+key, v)
		}
	}
}

// reIndex matches the index that walk appends to the elements of a list, such as the
// rdf:Description elements of the document or the rdf:li items of a tag list. The same property
// must be recognised whether it is the only element of its kind or one of several.
var reIndex = regexp.MustCompile(`\[\d+\]`)

func filter(md *assets.Metadata, p string, value string) {
	p = reIndex.ReplaceAllString(p, "")
	p = strings.TrimPrefix(p, "/xmpmeta/RDF/Description/")
	// debug 	fmt.Printf("%s: %s\n", p, value)
	switch p {
	case "DateTimeOriginal":
		if d, err := TimeStringToTime(value, time.UTC); err == nil {
			md.DateTaken = d
		}
	case "ImageDescription/Alt/li/#text":
		// An Alt list may repeat the text in several languages; the XMP specification
		// puts the x-default entry first, so the first item wins.
		if md.Description == "" {
			md.Description = value
		}
	case "Rating":
		md.Rating = StringToByte(value)
	case "TagsList/Seq/li":
		md.Tags = append(md.Tags,
			assets.Tag{
				Name:  path.Base(value),
				Value: value,
			})
	case "GPSLatitude":
		if f, err := GPTStringToFloat(value); err == nil {
			md.Latitude = f
		}
	case "GPSLongitude":
		if f, err := GPTStringToFloat(value); err == nil {
			md.Longitude = f
		}
	}
}
