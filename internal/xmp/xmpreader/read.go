package xmpreader

import (
	"io"
	"strconv"
	"time"

	"github.com/clbanning/mxj/v2"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/xmp/convert"
)

func ReadXMP(a *assets.Asset, r io.Reader) error {
	// Read the XMP data from the reader and return an Asset
	m, err := mxj.NewMapXmlReader(r)
	if err != nil {
		return err
	}
	walk(m, a, "")
	return nil
}

func walk(m mxj.Map, a *assets.Asset, path string) {
	for key, value := range m {
		switch v := value.(type) {
		case map[string]interface{}:
			walk(v, a, path+"/"+key)
		case []interface{}:
			path = path + "/" + key
			for _, item := range v {
				if itemMap, ok := item.(map[string]interface{}); ok {
					walk(itemMap, a, path)
				} else {
					filter(a, path, item.(string))
				}
			}
		default:
			filter(a, path+"/"+key, value.(string))
		}
	}
}

func filter(a *assets.Asset, path string, value string) {
	// fmt.Printf("filter: %s, %s\n", path, value)
	var err error
	switch path {
	case "/xmpmeta/RDF/Description/TagsList/Seq/Li":
		// a.Tags = append(a.Tags, value)
	case "/xmpmeta/RDF/Description/description/Alt/li/#text":
		a.Title = value
	case "/xmpmeta/RDF/Description/GPSLatitude":
		a.Latitude, err = convert.GPTStringToFloat(value)
	case "/xmpmeta/RDF/Description/GPSLongitude":
		a.Longitude, err = convert.GPTStringToFloat(value)
	case "/xmpmeta/RDF/Description/DateTimeOriginal":
		a.CaptureDate, err = convert.TimeStringToTime(value, time.UTC)
	case "/xmpmeta/RDF/Description/Rating":
		a.Stars, err = strconv.Atoi(value)
	}
	_ = err
}
