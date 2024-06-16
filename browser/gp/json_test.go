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
		isAsset   bool
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
			isAsset:   true,
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
			isAsset:   true,
		},
		{
			name: "new_takeout_album",
			json: `{
				"title": "Trip to Gdańsk",
				"description": "",
				"access": "protected",
				"date": {
					"timestamp": "1502439626",
					"formatted": "11 sie 2017, 08:20:26 UTC"
				},
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
			name: "old_takeout_album",
			json: `{
						"albumData": {
							"title": "Trip to Gdańsk",
							"description": "",
							"access": "protected",
							"location": "",
							"date": {
							"timestamp": "1502439626",
							"formatted": "11 sie 2017, 08:20:26 UTC"
							},
							"geoData": {
							"latitude": 0.0,
							"longitude": 0.0,
							"altitude": 0.0,
							"latitudeSpan": 0.0,
							"longitudeSpan": 0.0
							}
						}
						}`,
			isPartner: false,
			isAlbum:   true,
		},
		{
			name: "old_takeout_photo",
			json: `{
					"title": "IMG_20170803_115431469_HDR.jpg",
					"description": "",
					"imageViews": "0",
					"creationTime": {
						"timestamp": "1502439626",
						"formatted": "11 sie 2017, 08:20:26 UTC"
					},
					"modificationTime": {
						"timestamp": "1585318092",
						"formatted": "27 mar 2020, 14:08:12 UTC"
					},
					"geoData": {
						"latitude": 54.51708608333333,
						"longitude": 18.54171638888889,
						"altitude": 0.0,
						"latitudeSpan": 0.0,
						"longitudeSpan": 0.0
					},
					"geoDataExif": {
						"latitude": 54.51708608333333,
						"longitude": 18.54171638888889,
						"altitude": 0.0,
						"latitudeSpan": 0.0,
						"longitudeSpan": 0.0
					},
					"photoTakenTime": {
						"timestamp": "1501754071",
						"formatted": "3 sie 2017, 09:54:31 UTC"
					}
				}`,
			isAsset: true,
		},
		{
			name: "new takeout_asset",
			json: `{
						"title": "IMG_20170803_115431469_HDR.jpg",
						"description": "",
						"imageViews": "15",
						"creationTime": {
							"timestamp": "1502439626",
							"formatted": "11 sie 2017, 08:20:26 UTC"
						},
						"photoTakenTime": {
							"timestamp": "1501754071",
							"formatted": "3 sie 2017, 09:54:31 UTC"
						},
						"geoData": {
							"latitude": 54.5170861,
							"longitude": 18.5417164,
							"altitude": 0.0,
							"latitudeSpan": 0.0,
							"longitudeSpan": 0.0
						},
						"geoDataExif": {
							"latitude": 54.5170861,
							"longitude": 18.5417164,
							"altitude": 0.0,
							"latitudeSpan": 0.0,
							"longitudeSpan": 0.0
						},
						"url": "https://photos.google.com/photo/AF1QipNp7f29ZWPIDWAPMXJcNB2z7EMAGXWeTT066p9H",
						"googlePhotosOrigin": {
							"mobileUpload": {
							"deviceFolder": {
								"localFolderName": ""
							},
							"deviceType": "ANDROID_PHONE"
							}
						}
						}`,
			isAsset: true,
		},
		{
			name: "print_order",
			json: `
			{
			"externalOrderId": "417796788285446498760",
			"type": "PURCHASED",
			"quantity": 1,
			"numPages": 142,
			"creationTime": {
				"formatted": "Dec 12, 2022, 5:51:01 AM UTC"
			},
			"modificationTime": {
				"formatted": "Dec 12, 2022, 6:00:07 AM UTC"
			},
			"client": "WEB_DESKTOP",
			"category": "SHIPPED_PRINTS"
			}`,
		},
	}

	for _, c := range tcs {
		t.Run(c.name, func(t *testing.T) {
			var md GoogleMetaData
			err := json.NewDecoder(strings.NewReader(c.json)).Decode(&md)
			if err != nil {
				t.Errorf("unexpected error: %s", err)
				return
			}
			if c.isAsset != md.isAsset() {
				t.Errorf("expected isAsset to be %t, got %t", c.isAsset, md.isAsset())
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
