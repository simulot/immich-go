package immich

import (
	"bytes"
	"encoding/json"
	"testing"
)

func Test_searchMetadataRequest(t *testing.T) {
	body := `{
		"albums": {
			"total": 0,
			"count": 0,
			"items": [],
			"facets": []
		},
		"assets": {
			"total": 2,
			"count": 2,
			"items": [
				{
					"id": "90f33f69-4bc1-4ec4-b683-7c3f30ea6e57",
					"type": "IMAGE",
					"thumbhash": "4SgGDQKzh2+Id3iYeIiYeH9s+cmW",
					"localDateTime": "2024-02-26T17:08:30.000Z",
					"resized": true,
					"duration": "0:00:00.00000",
					"livePhotoVideoId": null,
					"hasMetadata": true,
					"deviceAssetId": "1000007507",
					"ownerId": "bac3e8ec-d6ef-4721-b14b-079d655438d4",
					"deviceId": "335475d885a9894fad34eed6c4663867e7412499d69e2fcba8d7d5290542dbeb",
					"libraryId": "e4ca1939-6149-44a8-8be5-f6415183fb3d",
					"originalPath": "upload/library/admin/2024/2024-02-26/IMG-20240226-WA0000.jpg",
					"originalFileName": "IMG-20240226-WA0000",
					"fileCreatedAt": "2024-02-26T17:08:30.000Z",
					"fileModifiedAt": "2024-02-26T17:08:31.000Z",
					"updatedAt": "2024-02-26T17:30:18.517Z",
					"isFavorite": false,
					"isArchived": false,
					"isTrashed": false,
					"exifInfo": {
						"make": null,
						"model": null,
						"exifImageWidth": 1200,
						"exifImageHeight": 1600,
						"fileSizeInByte": 237206,
						"orientation": null,
						"dateTimeOriginal": "2024-02-26T17:08:30.000Z",
						"modifyDate": "2024-02-26T17:08:31.000Z",
						"timeZone": null,
						"lensModel": null,
						"fNumber": null,
						"focalLength": null,
						"iso": null,
						"exposureTime": null,
						"latitude": null,
						"longitude": null,
						"city": null,
						"state": null,
						"country": null,
						"description": "",
						"projectionType": null
					},
					"people": [],
					"checksum": "mDk/Ipx2lm6N28WW7amedczy7KY=",
					"stackCount": null,
					"isExternal": false,
					"isOffline": false,
					"isReadOnly": false
				},
				{
					"id": "d6f5a992-b3ce-42c3-bd0f-6d8785d164da",
					"type": "IMAGE",
					"thumbhash": "pBgKBYCKiHdvdYiSh6hnhKpfsMgG",
					"localDateTime": "2024-02-25T11:31:23.000Z",
					"resized": true,
					"duration": "0:00:00.00000",
					"livePhotoVideoId": null,
					"hasMetadata": true,
					"deviceAssetId": "1000007495",
					"ownerId": "bac3e8ec-d6ef-4721-b14b-079d655438d4",
					"deviceId": "335475d885a9894fad34eed6c4663867e7412499d69e2fcba8d7d5290542dbeb",
					"libraryId": "e4ca1939-6149-44a8-8be5-f6415183fb3d",
					"originalPath": "upload/library/admin/2024/2024-02-25/IMG-20240225-WA0001.jpg",
					"originalFileName": "IMG-20240225-WA0001",
					"fileCreatedAt": "2024-02-25T11:31:23.000Z",
					"fileModifiedAt": "2024-02-25T11:31:24.000Z",
					"updatedAt": "2024-02-25T18:26:31.410Z",
					"isFavorite": false,
					"isArchived": false,
					"isTrashed": false,
					"exifInfo": {
						"make": null,
						"model": null,
						"exifImageWidth": 1280,
						"exifImageHeight": 964,
						"fileSizeInByte": 95295,
						"orientation": null,
						"dateTimeOriginal": "2024-02-25T11:31:23.000Z",
						"modifyDate": "2024-02-25T11:31:24.000Z",
						"timeZone": null,
						"lensModel": null,
						"fNumber": null,
						"focalLength": null,
						"iso": null,
						"exposureTime": null,
						"latitude": null,
						"longitude": null,
						"city": null,
						"state": null,
						"country": null,
						"description": "",
						"projectionType": null
					},
					"people": [],
					"checksum": "m8eMvM0JiUYFp2A/OqV30qio780=",
					"stackCount": null,
					"isExternal": false,
					"isOffline": false,
					"isReadOnly": false
				}
			],
			"facets": [],
			"nextPage": "2"
		}
	}`

	rest := searchMetadataResponse{}
	err := json.NewDecoder(bytes.NewBufferString(body)).Decode(&rest)
	if err != nil {
		t.Error(err)
		return
	}
	if len(rest.Assets.Items) != 2 {
		t.Errorf("expecting 2 assets, got: %d", len(rest.Assets.Items))
	}
	if rest.Assets.NextPage != 2 {
		t.Errorf("expecting next page, got: %d", rest.Assets.NextPage)
	}
}
