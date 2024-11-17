package immich

import (
	"encoding/json"
	"strings"
	"testing"
)

func Test_AssetJSON(t *testing.T) {
	js := `{
 "id": "9a2fff7a-f226-48e8-a888-fdac199f3d56",
 "deviceAssetId": "IMG_20180811_173822_1.jpg-2082855",
 "ownerId": "13e05729-8933-494e-982e-5910a0c4420f",
 "deviceId": "DESKTOP-ILBKKE7",
 "type": "IMAGE",
 "originalPath": "upload/upload/13e05729-8933-494e-982e-5910a0c4420f/17/6c/176c335a-fbc0-412f-a46f-c187351a55bd.jpg",
 "originalFileName": "IMG_20180811_173822_1.jpg",
 "resized": true,
 "thumbhash": "WRgGDQTZeaiYNz6FCUQXZg4BtAAV",
 "fileCreatedAt": "\"2018-08-11T19:38:22+02:00\"",
 "fileModifiedAt": "\"2024-07-07T17:29:15+02:00\"",
 "updatedAt": "\"2024-11-17T18:57:15+01:00\"",
 "isFavorite": false,
 "isArchived": false,
 "isTrashed": false,
 "duration": "0:00:00.00000",
 "rating": 0,
 "exifInfo": {
  "make": "HUAWEI",
  "model": "CLT-L09",
  "exifImageWidth": 2736,
  "exifImageHeight": 3648,
  "fileSizeInByte": 2082855,
  "orientation": "0",
  "dateTimeOriginal": "\"2018-08-11T19:38:22+02:00\"",
  "timeZone": "Europe/Paris",
  "latitude": 48.8413085936111,
  "longitude": 2.4199056625,
  "description": "oznor"
 },
 "livePhotoVideoId": "",
 "checksum": "fDpZUcgYJjZnzLAHfIddp8BLzjE=",
 "stackParentId": "",
 "tags": [
  {
   "id": "e6745272-71d2-4a61-976e-d4ac6b7de3b8",
   "name": "tag2",
   "value": "tag1/tag2"
  },
  {
   "id": "bbfd950a-f1b5-4e2d-acc9-e000a27d41e5",
   "name": "activities",
   "value": "activities"
  }
 ]
}`

	asset := Asset{}
	dec := json.NewDecoder(strings.NewReader(js))
	err := dec.Decode(&asset)
	if err != nil {
		t.Error(err)
	}
	if len(asset.Tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(asset.Tags))
	}
	expectedTags := []struct {
		ID    string
		Name  string
		Value string
	}{
		{"e6745272-71d2-4a61-976e-d4ac6b7de3b8", "tag2", "tag1/tag2"},
		{"bbfd950a-f1b5-4e2d-acc9-e000a27d41e5", "activities", "activities"},
	}

	for i, tag := range asset.Tags {
		if tag.ID != expectedTags[i].ID || tag.Name != expectedTags[i].Name || tag.Value != expectedTags[i].Value {
			t.Errorf("expected tag %v, got %v", expectedTags[i], tag)
		}
	}
}
