package gp

import (
	"bytes"
	"encoding/json"
	"errors"
	"io/fs"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/psanford/memfs"
	"github.com/simulot/immich-go/internal/fakefs"
	"github.com/simulot/immich-go/internal/filenames"
)

type inMemFS struct {
	*memfs.FS
	name string
	err  error
}

func newInMemFS(name string) *inMemFS { // nolint: unparam
	return &inMemFS{
		name: name,
		FS:   memfs.New(),
	}
}

func (mfs inMemFS) Name() string {
	return mfs.name
}

func (mfs *inMemFS) FSs() []fs.FS {
	return []fs.FS{mfs}
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

func (mfs *inMemFS) addFile2(name string) *inMemFS { // nolint: unused
	if mfs.err != nil {
		return mfs
	}
	dir := path.Dir(name)
	mfs.err = errors.Join(mfs.err, mfs.MkdirAll(dir, 0o777))
	mfs.err = errors.Join(mfs.err, mfs.WriteFile(name, []byte(name), 0o777))
	return mfs
}

func (mfs *inMemFS) addImage(name string, length int) *inMemFS {
	b := make([]byte, length)
	for i := 0; i < length; i++ {
		b[i] = byte(i % 256)
	}
	mfs.addFile(name, b)
	return mfs
}

type jsonFn func(md *GoogleMetaData)

func takenTime(date string) func(md *GoogleMetaData) {
	return func(md *GoogleMetaData) {
		md.PhotoTakenTime.Timestamp = strconv.FormatInt(filenames.TakeTimeFromName(date, time.UTC).Unix(), 10)
	}
}

func (mfs *inMemFS) addJSONImage(name string, title string, modifiers ...jsonFn) *inMemFS {
	md := GoogleMetaData{
		Title:          title,
		URLPresent:     true,
		PhotoTakenTime: &googTimeObject{},
	}
	md.PhotoTakenTime.Timestamp = strconv.FormatInt(time.Date(2023, 10, 23, 15, 0, 0, 0, time.Local).Unix(), 10)
	for _, f := range modifiers {
		f(&md)
	}
	content := bytes.NewBuffer(nil)
	enc := json.NewEncoder(content)
	err := enc.Encode(md)
	if err != nil {
		panic(err)
	}
	mfs.addFile(name, content.Bytes())
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

func simpleYear() []fs.FS {
	return newInMemFS("filesystem").
		addJSONImage("Photos from 2023/PXL_20230922_144936660.jpg.json", "PXL_20230922_144936660.jpg").
		addImage("Photos from 2023/PXL_20230922_144936660.jpg", 10).
		addJSONImage("Photos from 2023/PXL_20230922_144956000.jpg.json", "PXL_20230922_144956000.jpg").
		addImage("Photos from 2023/PXL_20230922_144956000.jpg", 20).FSs()
}

func simpleAlbum() []fs.FS {
	return newInMemFS("filesystem").
		addJSONImage("Photos from 2020/IMG_8172.jpg.json", "IMG_8172.jpg", takenTime("20200101103000")).
		addImage("Photos from 2020/IMG_8172.jpg", 25).
		addJSONImage("Photos from 2023/PXL_20230922_144936660.jpg.json", "PXL_20230922_144936660.jpg", takenTime("PXL_20230922_144936660")).
		addJSONImage("Photos from 2023/PXL_20230922_144934440.jpg.json", "PXL_20230922_144934440.jpg", takenTime("PXL_20230922_144934440")).
		addJSONImage("Photos from 2023/IMG_8172.jpg.json", "IMG_8172.jpg", takenTime("20230922102100")).
		addImage("Photos from 2023/PXL_20230922_144936660.jpg", 10).
		addImage("Photos from 2023/PXL_20230922_144934440.jpg", 15).
		addImage("Photos from 2023/IMG_8172.jpg", 52).
		addJSONAlbum("Album/anyname.json", "Album").
		addJSONImage("Album/IMG_8172.jpg.json", "IMG_8172.jpg", takenTime("20230922102100")).
		addJSONImage("Album/PXL_20230922_144936660.jpg.json", "PXL_20230922_144936660.jpg", takenTime("PXL_20230922_144936660")).
		addImage("Album/IMG_8172.jpg", 52).
		addImage("Album/PXL_20230922_144936660.jpg", 10).FSs()
}

func albumWithoutImage() []fs.FS {
	return newInMemFS("filesystem").
		addJSONAlbum("Album/anyname.json", "Album").
		addJSONImage("Album/PXL_20230922_144936660.jpg.json", "PXL_20230922_144936660.jpg").
		addJSONImage("Album/PXL_20230922_144934440.jpg.json", "PXL_20230922_144934440.jpg").
		addImage("Album/PXL_20230922_144936660.jpg", 10).
		addJSONImage("Photos from 2023/PXL_20230922_144934440.jpg.json", "PXL_20230922_144934440.jpg").
		addJSONImage("Photos from 2023/PXL_20230922_144936660.jpg.json", "PXL_20230922_144936660.jpg").
		addImage("Photos from 2023/PXL_20230922_144934440.jpg", 15).
		addImage("Photos from 2023/PXL_20230922_144936660.jpg", 10).FSs()
}

func namesWithNumbers() []fs.FS {
	return newInMemFS("filesystem").
		addJSONImage("Photos from 2009/IMG_3479.JPG.json", "IMG_3479.JPG").
		addImage("Photos from 2009/IMG_3479.JPG", 10).
		addJSONImage("Photos from 2009/IMG_3479.JPG(1).json", "IMG_3479.JPG").
		addImage("Photos from 2009/IMG_3479(1).JPG", 12).
		addJSONImage("Photos from 2009/IMG_3479.JPG(2).json", "IMG_3479.JPG").
		addImage("Photos from 2009/IMG_3479(2).JPG", 15).FSs()
}

func namesTruncated() []fs.FS {
	return newInMemFS("filesystem").
		addJSONImage("Photos from 2023/😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋.json", "😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋😛😝😜🤪🤨🧐🤓😎🥸🤩🥳😏😒😞😔😟😕🙁☹️😣😖😫😩🥺😢😭😤😠😡🤬🤯😳🥵🥶.jpg").
		addImage("Photos from 2023/😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋😛.jpg", 10).
		addJSONImage("Photos from 2023/PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGIN.json", "PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGINAL.jpg").
		addImage("Photos from 2023/PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGINA.jpg", 40).
		addJSONImage("Photos from 2023/05yqt21kruxwwlhhgrwrdyb6chhwszi9bqmzu16w0 2.jp.json", "05yqt21kruxwwlhhgrwrdyb6chhwszi9bqmzu16w0 2.jpg").
		addImage("Photos from 2023/05yqt21kruxwwlhhgrwrdyb6chhwszi9bqmzu16w0 2.jpg", 25).FSs()
}

func imagesEditedJSON() []fs.FS {
	return newInMemFS("filesystem").
		addJSONImage("Photos from 2023/PXL_20220405_090123740.PORTRAIT.jpg.json", "PXL_20220405_090123740.PORTRAIT.jpg").
		addImage("Photos from 2023/PXL_20220405_090123740.PORTRAIT.jpg", 41).
		addImage("Photos from 2023/PXL_20220405_090123740.PORTRAIT-modifié.jpg", 21).
		addImage("Photos from 2023/PXL_20220405_090200110.PORTRAIT-modifié.jpg", 12).FSs()
}

func titlesWithForbiddenChars() []fs.FS {
	return newInMemFS("filesystem").
		addJSONImage("Photos from 2012/27_06_12 - 1.mov.json", "27/06/12 - 1", takenTime("20120627")).
		addImage("Photos from 2012/27_06_12 - 1.mov", 52).
		addJSONImage("Photos from 2012/27_06_12 - 2.json", "27/06/12 - 2", takenTime("20120627")).
		addImage("Photos from 2012/27_06_12 - 2.jpg", 24).FSs()
}

func namesIssue39() []fs.FS {
	return newInMemFS("filesystem").
		addJSONAlbum("Album/anyname.json", "Album").
		addJSONImage("Album/Backyard_ceremony_wedding_photography_xxxxxxx_.json", "Backyard_ceremony_wedding_photography_xxxxxxx_magnoliastudios-371.jpg", takenTime("20200101")).
		addJSONImage("Album/Backyard_ceremony_wedding_photography_xxxxxxx_(1).json", "Backyard_ceremony_wedding_photography_xxxxxxx_magnoliastudios-181.jpg", takenTime("20200101")).
		addJSONImage("Album/Backyard_ceremony_wedding_photography_xxxxxxx_(494).json", "Backyard_ceremony_wedding_photography_markham_magnoliastudios-19.jpg", takenTime("20200101")).
		addImage("Album/Backyard_ceremony_wedding_photography_xxxxxxx_m.jpg", 1).
		addImage("Album/Backyard_ceremony_wedding_photography_xxxxxxx_m(1).jpg", 181).
		addImage("Album/Backyard_ceremony_wedding_photography_xxxxxxx_m(494).jpg", 494).
		addJSONImage("Photos from 2020/Backyard_ceremony_wedding_photography_xxxxxxx_.json", "Backyard_ceremony_wedding_photography_xxxxxxx_magnoliastudios-371.jpg", takenTime("20200101")).
		addJSONImage("Photos from 2020/Backyard_ceremony_wedding_photography_xxxxxxx_(1).json", "Backyard_ceremony_wedding_photography_xxxxxxx_magnoliastudios-181.jpg", takenTime("20200101")).
		addJSONImage("Photos from 2020/Backyard_ceremony_wedding_photography_xxxxxxx_(494).json", "Backyard_ceremony_wedding_photography_markham_magnoliastudios-19.jpg", takenTime("20200101")).
		addImage("Photos from 2020/Backyard_ceremony_wedding_photography_xxxxxxx_m(1).jpg", 181).
		addImage("Photos from 2020/Backyard_ceremony_wedding_photography_xxxxxxx_m(494).jpg", 494).
		addImage("Photos from 2020/Backyard_ceremony_wedding_photography_xxxxxxx_m.jpg", 1).FSs()
}

func issue68MPFiles() []fs.FS {
	return newInMemFS("filesystem").
		addJSONImage("Photos from 2022/PXL_20221228_185930354.MP.jpg.json", "PXL_20221228_185930354.MP.jpg", takenTime("20220101")).
		addImage("Photos from 2022/PXL_20221228_185930354.MP", 1).
		addImage("Photos from 2022/PXL_20221228_185930354.MP.jpg", 2).FSs()
}

func issue68LongExposure() []fs.FS {
	return newInMemFS("filesystem").
		addJSONImage("Photos from 2023/PXL_20230814_201154491.LONG_EXPOSURE-01.COVER..json", "PXL_20230814_201154491.LONG_EXPOSURE-01.COVER.jpg", takenTime("20230101")).
		addImage("Photos from 2023/PXL_20230814_201154491.LONG_EXPOSURE-01.COVER.jpg", 1).
		addJSONImage("Photos from 2023/PXL_20230814_201154491.LONG_EXPOSURE-02.ORIGIN.json", "PXL_20230814_201154491.LONG_EXPOSURE-02.ORIGINAL.jpg", takenTime("20230101")).
		addImage("Photos from 2023/PXL_20230814_201154491.LONG_EXPOSURE-02.ORIGINA.jpg", 2).FSs()
}

func issue68ForgottenDuplicates() []fs.FS {
	return newInMemFS("filesystem").
		addJSONImage("Photos from 2022/original_1d4caa6f-16c6-4c3d-901b-9387de10e528_.json", "original_1d4caa6f-16c6-4c3d-901b-9387de10e528_PXL_20220516_164814158.jpg", takenTime("20220101")).
		addImage("Photos from 2022/original_1d4caa6f-16c6-4c3d-901b-9387de10e528_P.jpg", 1).
		addImage("Photos from 2022/original_1d4caa6f-16c6-4c3d-901b-9387de10e528_P(1).jpg", 2).FSs()
}

// #390 Question: report shows way less images uploaded than scanned
func issue390WrongCount() []fs.FS {
	return newInMemFS("filesystem").
		addJSONImage("Takeout/Google Photos/Photos from 2021/image000000.jpg.json", "image000000.jpg").
		addJSONImage("Takeout/Google Photos/Photos from 2021/image000000.jpg(1).json", "image000000.jpg").
		addJSONImage("Takeout/Google Photos/Photos from 2021/image000000.gif.json", "image000000.gif.json").
		addImage("Takeout/Google Photos/Photos from 2021/image000000.gif", 10).
		addImage("Takeout/Google Photos/Photos from 2021/image000000.jpg", 20).FSs()
}

// Not used, but kept for reference
// func issue390WrongCount2() []fs.FS {
// 	return newInMemFS("filesystem").
// 		addJSONImage("Takeout/Google Photos/2017 - Croatia/IMG_0170.jpg.json", "IMG_0170.jpg", takenTime("2017-01-01 12:00:00")).
// 		addJSONImage("Takeout/Google Photos/Photos from 2018/IMG_0170.JPG.json", "IMG_0170.JPG", takenTime("2018-01-01 12:00:00")).
// 		addJSONImage("Takeout/Google Photos/Photos from 2018/IMG_0170.HEIC.json", "IMG_0170.HEIC", takenTime("2018-01-01 12:00:00")).
// 		addJSONImage("Takeout/Google Photos/Photos from 2023/IMG_0170.HEIC.json", "IMG_0170.HEIC", takenTime("2023-01-01 12:00:00")).
// 		addJSONImage("Takeout/Google Photos/2018 - Cambodia/IMG_0170.JPG.json", "IMG_0170.JPG", takenTime("2018-01-01 12:00:00")).
// 		addJSONImage("Takeout/Google Photos/2023 - Belize/IMG_0170.HEIC.json", "IMG_0170.HEIC", takenTime("2023-01-01 12:00:00")).
// 		addJSONImage("Takeout/Google Photos/Photos from 2017/IMG_0170.jpg.json", "IMG_0170.jpg", takenTime("2017-01-01 12:00:00")).
// 		addImage("Takeout/Google Photos/2017 - Croatia/IMG_0170.jpg", 514963).
// 		addImage("Takeout/Google Photos/Photos from 2018/IMG_0170.HEIC", 1332980).
// 		addImage("Takeout/Google Photos/Photos from 2018/IMG_0170.JPG", 4570661).
// 		addImage("Takeout/Google Photos/Photos from 2023/IMG_0170.MP4", 6024972).
// 		addImage("Takeout/Google Photos/Photos from 2023/IMG_0170.HEIC", 4443973).
// 		addImage("Takeout/Google Photos/Photos from 2018/IMG_0170.MP4", 2288647).
// 		addImage("Takeout/Google Photos/2018 - Cambodia/IMG_0170.JPG", 4570661).
// 		addImage("Takeout/Google Photos/2023 - Belize/IMG_0170.MP4", 6024972).
// 		addImage("Takeout/Google Photos/2023 - Belize/IMG_0170.HEIC", 4443973).
// 		addImage("Takeout/Google Photos/Photos from 2017/IMG_0170.jpg", 514963).FSs()
// }

func checkLivePhoto() []fs.FS { // nolint:unused
	return newInMemFS("filesystem").
		addJSONImage("Motion test/20231227_152817.jpg.json", "20231227_152817.jpg").
		addImage("Motion test/20231227_152817.jpg", 7426453).
		addImage("Motion test/20231227_152817.MP4", 5192477).
		addJSONImage("Motion Test/PXL_20231118_035751175.MP.jpg.json", "PXL_20231118_035751175.MP.jpg").
		addImage("Motion Test/PXL_20231118_035751175.MP", 3478685).
		addImage("Motion Test/PXL_20231118_035751175.MP.jpg", 8025699).
		addJSONImage("Motion Test/MVIMG_20180418_113218.jpg.json", "MVIMG_20180418_113218.jpg").
		addImage("Motion Test/MVIMG_20180418_113218.jpg", 12345).
		addImage("Motion Test/MVIMG_20180418_113218", 5656).FSs()
}

func loadFromString(dateFormat string, s string) []fs.FS { // nolint:unused
	fss, err := fakefs.ScanStringList(dateFormat, s)
	if err != nil {
		panic(err.Error())
	}
	return fss
}

func checkLivePhotoPixil() []fs.FS { // nolint:unused
	return loadFromString("01-02-2006 15:04", `Part: takeout-20230720T065335Z-001.zip
Archive:  takeout-20230720T065335Z-001.zip
  Length      Date    Time    Name
---------  ---------- -----   ----
      309  03-05-2023 10:10   Takeout/Google Photos/2022 - Germany/metadata.json
      801  07-19-2023 23:59   Takeout/Google Photos/2022 - Germany/IMG_4573.HEIC.json
  2232086  07-19-2023 23:59   Takeout/Google Photos/2022 - Germany/IMG_4573.MP4
  3530351  07-20-2023 00:00   Takeout/Google Photos/2022 - Germany/IMG_4573.HEIC
      319  03-05-2023 10:10   Takeout/Google Photos/2022 - Germany - Private/metadata.json
      802  07-20-2023 00:03   Takeout/Google Photos/2022 - Germany - Private/IMG_4573.HEIC.json
  3530351  07-19-2023 23:56   Takeout/Google Photos/2022 - Germany - Private/IMG_4573.HEIC
  2232086  07-19-2023 23:56   Takeout/Google Photos/2022 - Germany - Private/IMG_4573.MP4
      803  07-19-2023 23:58   Takeout/Google Photos/Photos from 2022/IMG_4573.HEIC.json
  3530351  07-19-2023 23:59   Takeout/Google Photos/Photos from 2022/IMG_4573.HEIC
  2232086  07-19-2023 23:59   Takeout/Google Photos/Photos from 2022/IMG_4573.MP4
`)
}

func checkMissingJSON() []fs.FS { // nolint:unused
	return loadFromString("01-02-2006 15:04", `Part:  takeout-20230720T065335Z-001.zip
Archive:  takeout-20230720T065335Z-001.zip
  Length      Date    Time    Name
---------  ---------- -----   ----
      803  07-19-2023 23:58   Takeout/Google Photos/Photos from 2022/IMG_4573.HEIC.json
  3530351  07-19-2023 23:59   Takeout/Google Photos/Photos from 2022/IMG_4573.HEIC
  1352455  07-19-2023 15:18   Takeout/Google Foto/Photos from 2016/IMG-20161201-WA0035.jpeg
  3530351  07-19-2023 23:56   Takeout/Google Photos/2022 - Germany - Private/IMG_4553.HEIC
      309  03-05-2023 10:10   Takeout/Google Photos/2022 - Germany/metadata.json
  2232086  07-19-2023 23:59   Takeout/Google Photos/2022 - Germany/IMG_1234.MP4
  3530351  07-20-2023 00:00   Takeout/Google Photos/2022 - Germany/IMG_1234.HEIC
`)
}

func checkDuplicates() []fs.FS { // nolint:unused
	return loadFromString("01-02-2006 15:04", `Part:  takeout-20230720T065335Z-001.tgz
-rw-r--r-- 0/0          365022 2024-07-19 01:19 Takeout/Google Foto/[E&S] 2016-01-05 - Castello De Albertis e Mostra d/20160105_121621_LLS.jpg
-rw-r--r-- 0/0             708 2024-07-19 01:19 Takeout/Google Foto/[E&S] 2016-01-05 - Castello De Albertis e Mostra d/20160105_121621_LLS.jpg.json
-rw-r--r-- 0/0          364041 2024-07-19 01:51 Takeout/Google Foto/Photos from 2016/20160105_121621_LLS.jpg
-rw-r--r-- 0/0             709 2024-07-19 01:51 Takeout/Google Foto/Photos from 2016/20160105_121621_LLS.jpg.json
-rw-r--r-- 0/0             708 2024-07-19 02:13 Takeout/Google Foto/2016-01-05 - _3/20160105_121621_LLS.jpg.json
-rw-r--r-- 0/0          364041 2024-07-19 02:20 Takeout/Google Foto/2016-01-05 - _3/20160105_121621_LLS.jpg
Part:  takeout-20230720T065335Z-002.tgz
-rw-r--r-- 0/0          364041 2024-07-19 06:14 Takeout/Google Foto/2016-01-05 - _3/20160105_121621_LLS.jpg
-rw-r--r-- 0/0             708 2024-07-19 02:13 Takeout/Google Foto/2016-01-05 - _3/20160105_121621_LLS.jpg.json
`)
}

func checkMPissue405() []fs.FS { // nolint:unused
	return loadFromString("2006-01-02 15:04", `Part:  takeout-20230720T065335Z-001.zip
      895  2024-01-21 16:52   Takeout/Google Photos/Untitled(1)/PXL_20210102_221126856.MP~2.jpg.json
      893  2024-01-21 16:52   Takeout/Google Photos/Untitled(1)/PXL_20210102_221126856.MP.jpg.json
  3242290  2024-01-21 16:58   Takeout/Google Photos/Untitled(1)/PXL_20210102_221126856.MP
  1214365  2024-01-21 16:58   Takeout/Google Photos/Untitled(1)/PXL_20210102_221126856.MP~2
  4028710  2024-01-21 16:59   Takeout/Google Photos/Untitled(1)/PXL_20210102_221126856.MP~2.jpg
  6486725  2024-01-21 16:59   Takeout/Google Photos/Untitled(1)/PXL_20210102_221126856.MP.jpg`)
}
