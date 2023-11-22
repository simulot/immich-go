package gp

import (
	"bytes"
	"encoding/json"
	"errors"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/simulot/immich-go/immich/metadata"

	"github.com/psanford/memfs"
)

type inMemFS struct {
	*memfs.FS
	err error
}

func newInMemFS() *inMemFS {
	return &inMemFS{
		FS: memfs.New(),
	}
}

func (mfs *inMemFS) addFile(name string, content []byte) *inMemFS {
	if mfs.err != nil {
		return mfs
	}
	dir := path.Dir(name)
	mfs.err = errors.Join(mfs.err, mfs.MkdirAll(dir, 0o777))
	mfs.err = errors.Join(mfs.err, mfs.WriteFile(name, content, 0o777))
	return mfs
}

func (mfs *inMemFS) addImage(name string, len int) *inMemFS {
	b := make([]byte, len)
	for i := 0; i < len; i++ {
		b[i] = byte(i % 256)
	}
	mfs.addFile(name, b)
	return mfs
}

type jsonFn func(md *GoogleMetaData)

func takenTime(date string) func(md *GoogleMetaData) {
	return func(md *GoogleMetaData) {
		md.PhotoTakenTime.Timestamp = strconv.FormatInt(metadata.TakeTimeFromName(date).Unix(), 10)
	}
}

func (mfs *inMemFS) addJSONImage(name string, title string, modifiers ...jsonFn) *inMemFS {
	md := GoogleMetaData{
		Title: title,
	}
	md.PhotoTakenTime.Timestamp = strconv.FormatInt(time.Date(2023, 10, 23, 15, 0, 0, 0, time.Local).Unix(), 10)
	for _, f := range modifiers {
		f(&md)
	}
	content := bytes.NewBuffer(nil)
	enc := json.NewEncoder(content)
	enc.Encode(md)
	mfs.addFile(name, []byte(content.Bytes()))
	return mfs
}

func (mfs *inMemFS) addJSONAlbum(file string, albumName string) *inMemFS {
	return mfs.addFile(file, []byte(`{
		"title": "`+albumName+`",
		"description": "",
		"access": "",
		"date": {
		  "timestamp": "0",
		  "formatted": "1 janv. 1970, 00:00:00 UTC"
		},
		"location": "",
		"geoData": {
		  "latitude": 0.0,
		  "longitude": 0.0,
		  "altitude": 0.0,
		  "latitudeSpan": 0.0,
		  "longitudeSpan": 0.0
		}
	  }`))
}

type fileResult struct {
	name  string
	size  int
	title string
}

func sortFileResult(s []fileResult) []fileResult {
	sort.Slice(s, func(i, j int) bool {
		c := strings.Compare(s[i].name, s[j].name)
		switch {
		case c < 0:
			return true
		case c > 0:
			return false
		}
		return s[i].size < s[j].size
	})
	return s
}

func simpleYear() *inMemFS {
	return newInMemFS().
		addJSONImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144936660.jpg.json", "PXL_20230922_144936660.jpg").
		addImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144936660.jpg", 10).
		addJSONImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144956000.jpg.json", "PXL_20230922_144956000.jpg").
		addImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144956000.jpg", 20)
}

func simpleAlbum() *inMemFS {
	return newInMemFS().
		addJSONImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144936660.jpg.json", "PXL_20230922_144936660.jpg", takenTime("PXL_20230922_144936660")).
		addImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144936660.jpg", 10).
		addJSONImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144934440.jpg.json", "PXL_20230922_144934440.jpg", takenTime("PXL_20230922_144934440")).
		addImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144934440.jpg", 15).
		addJSONAlbum("TakeoutGoogle Photos/Album/anyname.json", "Album").
		addJSONImage("TakeoutGoogle Photos/Album/PXL_20230922_144936660.jpg.json", "PXL_20230922_144936660.jpg", takenTime("PXL_20230922_144936660")).
		addImage("TakeoutGoogle Photos/Album/PXL_20230922_144936660.jpg", 10).
		addJSONImage("TakeoutGoogle Photos/Photos from 2023/IMG_8172.jpg.json", "IMG_8172.jpg", takenTime("20230922102100")).
		addImage("TakeoutGoogle Photos/Photos from 2023/IMG_8172.jpg", 52).
		addJSONImage("TakeoutGoogle Photos/Album/IMG_8172.jpg.json", "IMG_8172.jpg", takenTime("20230922102100")).
		addImage("TakeoutGoogle Photos/Album/IMG_8172.jpg", 52).
		addJSONImage("TakeoutGoogle Photos/Photos from 2020/IMG_8172.jpg.json", "IMG_8172.jpg", takenTime("20200101103000")).
		addImage("TakeoutGoogle Photos/Photos from 2020/IMG_8172.jpg", 25)
}

func albumWithoutImage() *inMemFS {
	return newInMemFS().
		addJSONImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144936660.jpg.json", "PXL_20230922_144936660.jpg").
		addImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144936660.jpg", 10).
		addJSONImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144934440.jpg.json", "PXL_20230922_144934440.jpg").
		addImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144934440.jpg", 15).
		addJSONAlbum("TakeoutGoogle Photos/Album/anyname.json", "Album").
		addJSONImage("TakeoutGoogle Photos/Album/PXL_20230922_144936660.jpg.json", "PXL_20230922_144936660.jpg").
		addImage("TakeoutGoogle Photos/Album/PXL_20230922_144936660.jpg", 10).
		addJSONImage("TakeoutGoogle Photos/Album/PXL_20230922_144934440.jpg.json", "PXL_20230922_144934440.jpg")
}

func namesWithNumbers() *inMemFS {
	return newInMemFS().
		addJSONImage("TakeoutGoogle Photos/Photos from 2009/IMG_3479.JPG.json", "IMG_3479.JPG").
		addImage("TakeoutGoogle Photos/Photos from 2009/IMG_3479.JPG", 10).
		addJSONImage("TakeoutGoogle Photos/Photos from 2009/IMG_3479.JPG(1).json", "IMG_3479.JPG").
		addImage("TakeoutGoogle Photos/Photos from 2009/IMG_3479(1).JPG", 12).
		addJSONImage("TakeoutGoogle Photos/Photos from 2009/IMG_3479.JPG(2).json", "IMG_3479.JPG").
		addImage("TakeoutGoogle Photos/Photos from 2009/IMG_3479(2).JPG", 15)
}

func namesTruncated() *inMemFS {
	return newInMemFS().
		addJSONImage("TakeoutGoogle Photos/Photos from 2023/😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋.json", "😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋😛😝😜🤪🤨🧐🤓😎🥸🤩🥳😏😒😞😔😟😕🙁☹️😣😖😫😩🥺😢😭😤😠😡🤬🤯😳🥵🥶.jpg").
		addImage("TakeoutGoogle Photos/Photos from 2023/😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋😛.jpg", 10).
		addJSONImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGIN.json", "PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGINAL.jpg").
		addImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGINA.jpg", 40).
		addJSONImage("TakeoutGoogle Photos/Photos from 2023/05yqt21kruxwwlhhgrwrdyb6chhwszi9bqmzu16w0 2.jp.json", "05yqt21kruxwwlhhgrwrdyb6chhwszi9bqmzu16w0 2.jpg").
		addImage("TakeoutGoogle Photos/Photos from 2023/05yqt21kruxwwlhhgrwrdyb6chhwszi9bqmzu16w0 2.jpg", 25)
}

func imagesEditedJSON() *inMemFS {
	return newInMemFS().
		addJSONImage("TakeoutGoogle Photos/Photos from 2023/PXL_20220405_090123740.PORTRAIT.jpg.json", "PXL_20220405_090123740.PORTRAIT.jpg").
		addImage("TakeoutGoogle Photos/Photos from 2023/PXL_20220405_090123740.PORTRAIT.jpg", 41).
		addImage("TakeoutGoogle Photos/Photos from 2023/PXL_20220405_090123740.PORTRAIT-modifié.jpg", 21).
		addImage("TakeoutGoogle Photos/Photos from 2023/PXL_20220405_090200110.PORTRAIT-modifié.jpg", 12)
}

func titlesWithForbiddenChars() *inMemFS {
	return newInMemFS().
		addJSONImage("TakeoutGoogle Photos/Photos from 2012/27_06_12 - 1.mov.json", "27/06/12 - 1.mov").
		addImage("TakeoutGoogle Photos/Photos from 2012/27_06_12 - 1.mov", 52)
	// addJSONImage("TakeoutGoogle Photos/Photos from 2012/27_06_12 - 1.json", "27/06/12 - 1").
	// addImage("TakeoutGoogle Photos/Photos from 2012/27_06_12 - 1.jpg", 24)
}

func namesIssue39() *inMemFS {
	return newInMemFS().
		addJSONAlbum("TakeoutGoogle Photos/Album/anyname.json", "Album").
		addJSONImage("TakeoutGoogle Photos/Album/Backyard_ceremony_wedding_photography_xxxxxxx_.json", "Backyard_ceremony_wedding_photography_xxxxxxx_magnoliastudios-371.jpg", takenTime("20200101")).
		addImage("TakeoutGoogle Photos/Album/Backyard_ceremony_wedding_photography_xxxxxxx_m.jpg", 1).
		addImage("TakeoutGoogle Photos/Photos from 2020/Backyard_ceremony_wedding_photography_xxxxxxx_m.jpg", 1).
		addJSONImage("TakeoutGoogle Photos/Album/Backyard_ceremony_wedding_photography_xxxxxxx_(1).json", "Backyard_ceremony_wedding_photography_xxxxxxx_magnoliastudios-181.jpg", takenTime("20200101")).
		addImage("TakeoutGoogle Photos/Album/Backyard_ceremony_wedding_photography_xxxxxxx_m(1).jpg", 181).
		addImage("TakeoutGoogle Photos/Photos from 2020/Backyard_ceremony_wedding_photography_xxxxxxx_m(1).jpg", 181).
		addJSONImage("TakeoutGoogle Photos/Album/Backyard_ceremony_wedding_photography_xxxxxxx_(494).json", "Backyard_ceremony_wedding_photography_markham_magnoliastudios-19.jpg", takenTime("20200101")).
		addImage("TakeoutGoogle Photos/Photos from 2020/Backyard_ceremony_wedding_photography_xxxxxxx_m(494).jpg", 494)
}
