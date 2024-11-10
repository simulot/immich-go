package xmpreader

import (
	"fmt"
	"io"
	"regexp"
	"strconv"
	"strings"
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
			for i, item := range v {
				p := fmt.Sprintf("%s[%d]", path, i)
				if itemMap, ok := item.(map[string]interface{}); ok {
					walk(itemMap, a, p)
				} else {
					filter(a, p, item.(string))
				}
			}
		default:
			filter(a, path+"/"+key, value.(string))
		}
	}
}

var (
	reAlbum = regexp.MustCompile(`/xmpmeta/RDF/Description/ImmichGoProperties/albums/Bag/Li(\[(\d+)\])?(.*)`)
	reTag   = regexp.MustCompile(`/xmpmeta/RDF/Description/ImmichGoProperties/tags/Bag/Li(\[(\d+)\])?(.*)`)
)

func filter(a *assets.Asset, path string, value string) {
	// debug	fmt.Printf("%s: %s\n", path, value)
	switch {
	case path == "/xmpmeta/RDF/Description/ImmichGoProperties/title":
		a.Title = value
	case path == "/xmpmeta/RDF/Description/ImmichGoProperties/favorite":
		a.Favorite = convert.StringToBool(value)
	case path == "/xmpmeta/RDF/Description/ImmichGoProperties/rating":
		a.Rating = convert.StringToInt(value)
	case path == "/xmpmeta/RDF/Description/ImmichGoProperties/trashed":
		a.Trashed = convert.StringToBool(value)
	case path == "/xmpmeta/RDF/Description/ImmichGoProperties/archived":
		a.Archived = convert.StringToBool(value)
	case path == "/xmpmeta/RDF/Description/ImmichGoProperties/fromPartner":
		a.FromPartner = convert.StringToBool(value)
	case path == "/xmpmeta/RDF/Description/ImmichGoProperties/latitude":
		if f, err := convert.GPTStringToFloat(value); err == nil {
			a.Latitude = f
		}
	case path == "/xmpmeta/RDF/Description/ImmichGoProperties/longitude":
		if f, err := convert.GPTStringToFloat(value); err == nil {
			a.Longitude = f
		}
	case path == "/xmpmeta/RDF/Description/ImmichGoProperties/DateTimeOriginal":
		if d, err := convert.TimeStringToTime(value, time.UTC); err == nil {
			a.CaptureDate = d
		}
	case strings.HasPrefix(path, "/xmpmeta/RDF/Description/ImmichGoProperties/albums/Bag/Li"):
		// Extract the index and the remaining pat
		matches := reAlbum.FindStringSubmatch(path)
		if len(matches) == 4 {
			index, _ := strconv.Atoi(matches[2])
			remainingPath := matches[3]
			if len(a.Albums) <= index {
				a.Albums = append(a.Albums, make([]assets.Album, index-len(a.Albums)+1)...)
			}
			switch remainingPath {
			case "/album/title":
				a.Albums[index].Title = value
			case "/album/description":
				a.Albums[index].Description = value
			case "/album/latitude":
				if f, err := convert.GPTStringToFloat(value); err == nil {
					a.Albums[index].Latitude = f
				}
			case "/album/longitude":
				if f, err := convert.GPTStringToFloat(value); err == nil {
					a.Albums[index].Longitude = f
				}
			}
		}
	case strings.HasPrefix(path, "/xmpmeta/RDF/Description/ImmichGoProperties/tags/Bag/Li"):
		// Extract the index and the remaining path
		matches := reTag.FindStringSubmatch(path)
		if len(matches) == 4 {
			index, _ := strconv.Atoi(matches[2])
			remainingPath := matches[3]
			if len(a.Tags) <= index {
				a.Tags = append(a.Tags, make([]assets.Tag, index-len(a.Tags)+1)...)
			}
			switch remainingPath {
			case "/tag/name":
				a.Tags[index].Name = value
			case "/tag/value":
				a.Tags[index].Value = value
			}
		}
	}
}
