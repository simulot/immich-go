package gp_test

import (
	"errors"
	"fmt"
	"path"

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
	mfs.err = errors.Join(mfs.err, mfs.MkdirAll(dir, 0777))
	mfs.err = errors.Join(mfs.err, mfs.WriteFile(name, content, 0777))
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
func (mfs *inMemFS) addJSONImage(name string, content string) *inMemFS {
	mfs.addFile(name, []byte(content))
	return mfs
}

func generateJSONTitle(title string) string {
	return fmt.Sprintf(`{
		"title": "%s",
		"description": "",
		"imageViews": "0",
		"creationTime": {
		  "timestamp": "1695397525",
		  "formatted": "22 sept. 2023, 15:45:25 UTC"
		},
		"photoTakenTime": {
		  "timestamp": "1695394176",
		  "formatted": "22 sept. 2023, 14:49:36 UTC"
		},
		"geoData": {
		  "latitude": 48.7981917,
		  "longitude": 2.4866832999999997,
		  "altitude": 90.25,
		  "latitudeSpan": 0.0,
		  "longitudeSpan": 0.0
		},
		"geoDataExif": {
		  "latitude": 48.7981917,
		  "longitude": 2.4866832999999997,
		  "altitude": 90.25,
		  "latitudeSpan": 0.0,
		  "longitudeSpan": 0.0
		},
		"url": "https://photos.google.com/photo/AAA",
		"googlePhotosOrigin": {
		  "mobileUpload": {
			"deviceFolder": {
			  "localFolderName": ""
			},
			"deviceType": "ANDROID_PHONE"
		  }
		}
	  }`, title)
}

func simpleYear() *inMemFS {
	return newInMemFS().
		addJSONImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144936660.jpg.json",
			generateJSONTitle("PXL_20230922_144936660.jpg")).
		addImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144936660.jpg", 10).
		addJSONImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144956000.jpg.json",
			generateJSONTitle("PXL_20230922_144956000.jpg")).
		addImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144956000.jpg", 20)

}

func simpleAlbum() *inMemFS {
	return newInMemFS().
		addJSONImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144936660.jpg.json",
			generateJSONTitle("PXL_20230922_144936660.jpg")).
		addImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144936660.jpg", 10).
		addJSONImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144934440.jpg.json",
			generateJSONTitle("PXL_20230922_144934440.jpg")).
		addImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144934440.jpg", 15).
		addFile("TakeoutGoogle Photos/Album/anyname.json", []byte(`{
			"title": "Album Name",
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
		  }`)).
		addJSONImage("TakeoutGoogle Photos/Album/PXL_20230922_144936660.jpg.json",
			generateJSONTitle("PXL_20230922_144936660.jpg")).
		addImage("TakeoutGoogle Photos/Album/PXL_20230922_144936660.jpg", 10)
}

func albumWithoutImage() *inMemFS {
	return newInMemFS().
		addJSONImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144936660.jpg.json",
			generateJSONTitle("PXL_20230922_144936660.jpg")).
		addImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144936660.jpg", 10).
		addJSONImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144934440.jpg.json",
			generateJSONTitle("PXL_20230922_144934440.jpg")).
		addImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230922_144934440.jpg", 15).
		addFile("TakeoutGoogle Photos/Album/anyname.json", []byte(`{
			"title": "Album Name",
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
		  }`)).
		addJSONImage("TakeoutGoogle Photos/Album/PXL_20230922_144936660.jpg.json",
			generateJSONTitle("PXL_20230922_144936660.jpg")).
		addImage("TakeoutGoogle Photos/Album/PXL_20230922_144936660.jpg", 10).
		addJSONImage("TakeoutGoogle Photos/Album/PXL_20230922_144934440.jpg.json",
			generateJSONTitle("PXL_20230922_144934440.jpg"))
}

func namesWithNumbers() *inMemFS {
	return newInMemFS().
		addJSONImage("TakeoutGoogle Photos/Photos from 2009/IMG_3479.JPG.json", generateJSONTitle("IMG_3479.JPG")).
		addImage("TakeoutGoogle Photos/Photos from 2009/IMG_3479.JPG", 10).
		addJSONImage("TakeoutGoogle Photos/Photos from 2009/IMG_3479.JPG(1).json", generateJSONTitle("IMG_3479.JPG")).
		addImage("TakeoutGoogle Photos/Photos from 2009/IMG_3479(1).JPG", 12).
		addJSONImage("TakeoutGoogle Photos/Photos from 2009/IMG_3479.JPG(2).json", generateJSONTitle("IMG_3479.JPG")).
		addImage("TakeoutGoogle Photos/Photos from 2009/IMG_3479(2).JPG", 15)
}

func namesTruncated() *inMemFS {
	return newInMemFS().
		addFile("TakeoutGoogle Photos/Photos from 2023/😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋.json", []byte(`{
			"title": "😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋😛😝😜🤪🤨🧐🤓😎🥸🤩🥳😏😒😞😔😟😕🙁☹️😣😖😫😩🥺😢😭😤😠😡🤬🤯😳🥵🥶.jpg",
			"description": "",
			"imageViews": "0",
			"creationTime": {
			  "timestamp": "1692638384",
			  "formatted": "21 août 2023, 17:19:44 UTC"
			},
			"photoTakenTime": {
			  "timestamp": "1689020522",
			  "formatted": "10 juil. 2023, 20:22:02 UTC"
			},
			"geoData": {
			  "latitude": 0.0,
			  "longitude": 0.0,
			  "altitude": 0.0,
			  "latitudeSpan": 0.0,
			  "longitudeSpan": 0.0
			},
			"geoDataExif": {
			  "latitude": 0.0,
			  "longitude": 0.0,
			  "altitude": 0.0,
			  "latitudeSpan": 0.0,
			  "longitudeSpan": 0.0
			},
			"url": "https://photos.google.com/photo/AAAA",
			"googlePhotosOrigin": {
			  "webUpload": {
				"computerUpload": {
				}
			  }
			}
		  }`)).
		addImage("TakeoutGoogle Photos/Photos from 2023/😀😃😄😁😆😅😂🤣🥲☺️😊😇🙂🙃😉😌😍🥰😘😗😙😚😋😛.jpg", 10).
		addJSONImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGIN.json", generateJSONTitle("PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGINAL.jpg")).
		addImage("TakeoutGoogle Photos/Photos from 2023/PXL_20230809_203449253.LONG_EXPOSURE-02.ORIGINA.jpg", 40)
}

func imagesWithoutJSON() *inMemFS {
	return newInMemFS().
		addJSONImage("TakeoutGoogle Photos/Photos from 2023/PXL_20220405_090123740.PORTRAIT.jpg.json", generateJSONTitle("PXL_20220405_090123740.PORTRAIT.jpg")).
		addImage("TakeoutGoogle Photos/Photos from 2023/PXL_20220405_090123740.PORTRAIT.jpg", 41).
		addImage("TakeoutGoogle Photos/Photos from 2023/PXL_20220405_090123740.PORTRAIT-modifié.jpg", 21).
		addImage("TakeoutGoogle Photos/Photos from 2023/PXL_20220405_090200110.PORTRAIT-modifié.jpg", 12)
}

func titlesWithForbiddenChars() *inMemFS {
	return newInMemFS().
		addJSONImage("TakeoutGoogle Photos/Photos from 2012/27_06_12 - 1.mov.json", generateJSONTitle("27/06/12 - 1.mov")).
		addImage("TakeoutGoogle Photos/Photos from 2012/27_06_12 - 1.mov", 52)

}
