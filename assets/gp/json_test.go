package gp

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestPresentFields(t *testing.T) {

	tcs := []struct {
		name      string
		json      string
		isPartner bool
		isAlbum   bool
	}{
		{
			name: "regularJSON",
			json: `{
				"title": "title",
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
				"url": "https://photos.google.com/photo/AAMKMAKZMAZMKAZMKZMAK",
				"googlePhotosOrigin": {
				  "mobileUpload": {
					"deviceFolder": {
					  "localFolderName": ""
					},
					"deviceType": "ANDROID_PHONE"
				  }
				}
			  }`,
			isPartner: false,
			isAlbum:   false,
		},
		{
			name: "albumJson",
			json: `{
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
			  }`,
			isPartner: false,
			isAlbum:   true,
		},
		{
			name: "partner",
			json: `{
				"title": "IMG_1559.HEIC",
				"description": "",
				"imageViews": "4",
				"creationTime": {
				  "timestamp": "1687792236",
				  "formatted": "26 juin 2023, 15:10:36 UTC"
				},
				"photoTakenTime": {
				  "timestamp": "1687791968",
				  "formatted": "26 juin 2023, 15:06:08 UTC"
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
				"url": "https://photos.google.com/photo/AF1QipMiih4bHng7H2JcBe32Z70f86FWJxz3WwLjhc75",
				"googlePhotosOrigin": {
				  "fromPartnerSharing": {
				  }
				}
			  }`,
			isPartner: true,
			isAlbum:   false,
		},
	}

	for _, c := range tcs {
		t.Run(c.name, func(t *testing.T) {
			var md googleMetaData
			err := json.NewDecoder(strings.NewReader(c.json)).Decode(&md)
			if err != nil {
				t.Errorf("unexpected error: %s", err)
				return
			}
			if c.isAlbum != md.isAlbum() {
				t.Errorf("expected isAlbum to be %t, got %t", c.isAlbum, md.isAlbum())
			}
			if c.isPartner != md.isPartner() {
				t.Errorf("expected isPartner to be %t, got %t", c.isPartner, md.isPartner())
			}
		})
	}

}
