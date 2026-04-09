package upload

import (
	"testing"
	"time"

	"github.com/simulot/immich-go/internal/assets"
	"github.com/stretchr/testify/require"
)

func TestFindSnapchatFuzzyDuplicateImage(t *testing.T) {
	ii := newAssetIndex()
	day := time.Date(2019, 11, 1, 13, 34, 35, 0, time.UTC)

	server := &assets.Asset{
		ID:               "s1",
		OriginalFileName: "Snapchat-359293977.jpg",
		CaptureDate:      day,
		FileSize:         232793,
		Checksum:         "c1",
		NameInfo:         assets.NameInfo{Type: "image"},
	}
	ii.add(server, false)

	local := &assets.Asset{
		OriginalFileName: "2019-11-01_1097af19-9bd1-d7aa-9baa-bbdc4a311bce-main.jpg",
		CaptureDate:      time.Date(2019, 11, 1, 0, 0, 0, 0, time.UTC),
		FileDate:         day,
		FileSize:         233218,
		NameInfo:         assets.NameInfo{Type: "image"},
	}

	best := ii.findSnapchatFuzzyDuplicate(local)
	require.NotNil(t, best)
	require.Equal(t, "s1", best.ID)
}

func TestFindSnapchatFuzzyDuplicateVideo(t *testing.T) {
	ii := newAssetIndex()
	day := time.Date(2019, 11, 1, 19, 10, 9, 0, time.UTC)

	server := &assets.Asset{
		ID:               "s1",
		OriginalFileName: "Snapchat-35037201.mp4",
		CaptureDate:      day,
		FileSize:         524977,
		Checksum:         "c1",
		NameInfo:         assets.NameInfo{Type: "video"},
	}
	ii.add(server, false)

	local := &assets.Asset{
		OriginalFileName: "2019-11-01_3078fe42-7c55-4ed8-28d7-b0a5fae6b10d-main.mp4",
		CaptureDate:      time.Date(2019, 11, 1, 0, 0, 0, 0, time.UTC),
		FileDate:         day,
		FileSize:         1312244,
		NameInfo:         assets.NameInfo{Type: "video"},
	}

	best := ii.findSnapchatFuzzyDuplicate(local)
	require.NotNil(t, best)
	require.Equal(t, "s1", best.ID)
}

func TestFindSnapchatFuzzyDuplicateNotSnapchatLocalName(t *testing.T) {
	ii := newAssetIndex()
	server := &assets.Asset{
		ID:               "s1",
		OriginalFileName: "Snapchat-359293977.jpg",
		CaptureDate:      time.Date(2019, 11, 1, 13, 34, 35, 0, time.UTC),
		FileSize:         232793,
		Checksum:         "c1",
		NameInfo:         assets.NameInfo{Type: "image"},
	}
	ii.add(server, false)

	local := &assets.Asset{
		OriginalFileName: "IMG_1234.jpg",
		CaptureDate:      time.Date(2019, 11, 1, 13, 34, 35, 0, time.UTC),
		FileSize:         232793,
		NameInfo:         assets.NameInfo{Type: "image"},
	}

	require.Nil(t, ii.findSnapchatFuzzyDuplicate(local))
}

func TestFindSnapchatFuzzyDuplicateAdjacentDay(t *testing.T) {
	ii := newAssetIndex()
	server := &assets.Asset{
		ID:               "s1",
		OriginalFileName: "Snapchat-12345.jpg",
		CaptureDate:      time.Date(2019, 12, 23, 0, 2, 0, 0, time.UTC),
		FileSize:         101000,
		Checksum:         "c1",
		NameInfo:         assets.NameInfo{Type: "image"},
	}
	ii.add(server, false)

	local := &assets.Asset{
		OriginalFileName: "2019-12-22_67f47d95-3530-3822-f062-be9a127cf71b-main.jpg",
		CaptureDate:      time.Date(2019, 12, 22, 23, 59, 0, 0, time.UTC),
		FileSize:         100000,
		NameInfo:         assets.NameInfo{Type: "image"},
	}

	best := ii.findSnapchatFuzzyDuplicate(local)
	require.NotNil(t, best)
	require.Equal(t, "s1", best.ID)
}

func TestFindSnapchatFuzzyDuplicateAmbiguousRejected(t *testing.T) {
	ii := newAssetIndex()
	day := time.Date(2020, 5, 1, 12, 0, 0, 0, time.UTC)

	server1 := &assets.Asset{
		ID:               "s1",
		OriginalFileName: "Snapchat-111.jpg",
		CaptureDate:      day,
		FileSize:         100000,
		Checksum:         "c1",
		NameInfo:         assets.NameInfo{Type: "image"},
	}
	server2 := &assets.Asset{
		ID:               "s2",
		OriginalFileName: "Snapchat-222.jpg",
		CaptureDate:      day,
		FileSize:         101000,
		Checksum:         "c2",
		NameInfo:         assets.NameInfo{Type: "image"},
	}
	ii.add(server1, false)
	ii.add(server2, false)

	local := &assets.Asset{
		OriginalFileName: "2020-05-01_aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee-main.jpg",
		CaptureDate:      day,
		FileSize:         100400,
		NameInfo:         assets.NameInfo{Type: "image"},
	}

	require.Nil(t, ii.findSnapchatFuzzyDuplicate(local))
}

func TestFindSnapchatFuzzyDuplicateRejectTooFarDay(t *testing.T) {
	ii := newAssetIndex()
	server := &assets.Asset{
		ID:               "s1",
		OriginalFileName: "Snapchat-98765.mp4",
		CaptureDate:      time.Date(2020, 5, 3, 12, 0, 0, 0, time.UTC),
		FileSize:         500000,
		Checksum:         "c1",
		NameInfo:         assets.NameInfo{Type: "video"},
	}
	ii.add(server, false)

	local := &assets.Asset{
		OriginalFileName: "2020-05-01_bbbbbbbb-cccc-dddd-eeee-ffffffffffff-main.mp4",
		CaptureDate:      time.Date(2020, 5, 1, 12, 0, 0, 0, time.UTC),
		FileSize:         510000,
		NameInfo:         assets.NameInfo{Type: "video"},
	}

	require.Nil(t, ii.findSnapchatFuzzyDuplicate(local))
}
