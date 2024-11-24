package xmpsidecar

import (
	"fmt"
	"io"
	"path"
	"regexp"
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
			path = path + "/" + key
			for i, item := range v {
				p := fmt.Sprintf("%s[%d]", path, i)
				if itemMap, ok := item.(map[string]interface{}); ok {
					walk(itemMap, md, p)
				} else {
					filter(md, p, item.(string))
				}
			}
		default:
			filter(md, path+"/"+key, value.(string))
		}
	}
}

var reDescription = regexp.MustCompile(`/xmpmeta/RDF/Description\[\d+\]/`)

func filter(md *assets.Metadata, p string, value string) {
	p = reDescription.ReplaceAllString(p, "")
	// debug 	fmt.Printf("%s: %s\n", p, value)
	switch p {
	case "DateTimeOriginal":
		if d, err := TimeStringToTime(value, time.UTC); err == nil {
			md.DateTaken = d
		}
	case "ImageDescription/Alt/li/#text":
		md.Description = value
	case "Rating":
		md.Rating = StringToByte(value)
	case "TagsList/Seq/li":
		md.Tags = append(md.Tags,
			assets.Tag{
				Name:  path.Base(value),
				Value: value,
			})
	case "/xmpmeta/RDF/Description/GPSLatitude":
		if f, err := GPTStringToFloat(value); err == nil {
			md.Latitude = f
		}
	case "/xmpmeta/RDF/Description/GPSLongitude":
		if f, err := GPTStringToFloat(value); err == nil {
			md.Longitude = f
		}
	}
}
