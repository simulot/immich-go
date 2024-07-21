package fakefs

import (
	"fmt"
	"io"
	"strings"
	"time"
)

const albumTemplate = `{
  "title": "%s",
  "description": "",
  "access": "",
  "date": {
    "timestamp": "0",
    "formatted": "1 janv. 1970, 00:00:00 UTC"
  },
  "geoData": {
    "latitude": 0.0,
    "longitude": 0.0,
    "altitude": 0.0,
    "latitudeSpan": 0.0,
    "longitudeSpan": 0.0
  }
}`

func fakeAlbumData(name string) (io.Reader, int64) {
	t := fmt.Sprintf(albumTemplate, name)
	return strings.NewReader(t), int64(len(t))
}

const pictureTemplate = `{
  "title": "%[1]s",
  "description": "",
  "imageViews": "50",
  "creationTime": {
    "timestamp": "%[2]d"
  },
  "photoTakenTime": {
    "timestamp": "%[2]d"
  },
  "geoData": {
    "latitude": 48.0,
    "longitude": 1.0,
    "altitude": 102.86,
    "latitudeSpan": 0.0,
    "longitudeSpan": 0.0
  },
  "geoDataExif": {
    "latitude": 48.0,
    "longitude": 1.0,
    "altitude": 102.86,
    "latitudeSpan": 0.0,
    "longitudeSpan": 0.0
  },
  "url": "https://photos.google.com/photo/AF1QipMZVTuUYj4K1jaN5vy6mkflX6yiWLQO2GDXSNKl",
  "googlePhotosOrigin": {
    "webUpload": {
      "computerUpload": {
      }
    }
  }
}`

func fakePhotoData(name string, captureDate time.Time) (io.Reader, int64) {
	t := fmt.Sprintf(pictureTemplate, name, captureDate.Unix())
	return strings.NewReader(t), int64(len(t))
}

const fakeJSONTemplate = `{
  "Nothing": ""
}`

func fakeJSON() (io.Reader, int64) {
	t := fakeJSONTemplate
	return strings.NewReader(t), int64(len(t))
}
