package stacking

import (
	"reflect"
	"testing"
	"time"

	"github.com/kr/pretty"
	"github.com/simulot/immich-go/immich/metadata"
)

type asset struct {
	ID        string
	FileName  string
	DateTaken time.Time
}

func Test_Stack(t *testing.T) {
	tc := []struct {
		name  string
		input []asset
		want  []Stack
	}{
		{
			name: "no stack JPG+DNG",
			input: []asset{
				{ID: "1", FileName: "IMG_1234.JPG", DateTaken: metadata.TakeTimeFromName("2023-10-01 10.15.00")},
				{ID: "2", FileName: "IMG_1234.DNG", DateTaken: metadata.TakeTimeFromName("2023-10-01 10.45.00")},
			},
		},
		{
			name: "stack JPG+DNG",
			input: []asset{
				{ID: "1", FileName: "IMG_1234.JPG", DateTaken: metadata.TakeTimeFromName("2023-10-01 10.15.00")},
				{ID: "2", FileName: "IMG_1234.DNG", DateTaken: metadata.TakeTimeFromName("2023-10-01 10.15.00")},
			},
			want: []Stack{
				{
					CoverID: "1",
					IDs:     []string{"2"},
					Date:    metadata.TakeTimeFromName("2023-10-01 10.15.00"),
					Names:   []string{"IMG_1234.JPG", "IMG_1234.DNG"},
				},
			},
		},
		{
			name: "stack BURST",
			input: []asset{
				{ID: "1", FileName: "IMG_20231014_183244.jpg", DateTaken: metadata.TakeTimeFromName("IMG_20231014_183244.jpg")},
				{ID: "2", FileName: "IMG_20231014_183246_BURST001_COVER.jpg", DateTaken: metadata.TakeTimeFromName("IMG_20231014_183246_BURST001_COVER.jpg")},
				{ID: "3", FileName: "IMG_20231014_183246_BURST002.jpg", DateTaken: metadata.TakeTimeFromName("IMG_20231014_183246_BURST002.jpg")},
				{ID: "4", FileName: "IMG_20231014_183246_BURST003.jpg", DateTaken: metadata.TakeTimeFromName("IMG_20231014_183246_BURST003.jpg")},
			},
			want: []Stack{
				{
					CoverID: "2",
					IDs:     []string{"3", "4"},
					Date:    metadata.TakeTimeFromName("IMG_20231014_183246_BURST001_COVER.jpg"),
					Names:   []string{"IMG_20231014_183246_BURST001_COVER.jpg", "IMG_20231014_183246_BURST002.jpg", "IMG_20231014_183246_BURST003.jpg"},
				},
			},
		},
		{
			name: "stack JPG+CR3",
			input: []asset{
				{ID: "1", FileName: "3H2A0018.CR3", DateTaken: metadata.TakeTimeFromName("2023-10-01 10.15.00")},
				{ID: "2", FileName: "3H2A0018.JPG", DateTaken: metadata.TakeTimeFromName("2023-10-01 10.15.00")},
				{ID: "3", FileName: "3H2A0019.CR3", DateTaken: metadata.TakeTimeFromName("2023-10-01 10.15.00")},
				{ID: "4", FileName: "3H2A0019.JPG", DateTaken: metadata.TakeTimeFromName("2023-10-01 10.15.00")},
			},
			want: []Stack{
				{
					CoverID: "2",
					IDs:     []string{"1"},
					Date:    metadata.TakeTimeFromName("2023-10-01 10.15.00"),
					Names:   []string{"3H2A0018.CR3", "3H2A0018.JPG"},
				},
				{
					CoverID: "4",
					IDs:     []string{"3"},
					Date:    metadata.TakeTimeFromName("2023-10-01 10.15.00"),
					Names:   []string{"3H2A0019.CR3", "3H2A0019.JPG"},
				},
			},
		},
	}

	for _, tt := range tc {
		t.Run(tt.name, func(t *testing.T) {
			sb := NewStackBuilder()
			for _, a := range tt.input {
				sb.ProcessAsset(a.ID, a.FileName, a.DateTaken)
			}

			got := sb.Stacks()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("difference\n")
				pretty.Ldiff(t, tt.want, got)
			}
		})

	}
}
