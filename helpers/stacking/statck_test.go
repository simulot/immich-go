package stacking

import (
	"reflect"
	"testing"
	"time"

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
				{ID: "1", FileName: "IMG_1234.DNG", DateTaken: metadata.TakeTimeFromName("2023-10-01 10.45.00")},
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
				t.Errorf("Stacks()=%v, want: %v", got, tt.want)
			}

		})

	}
}
