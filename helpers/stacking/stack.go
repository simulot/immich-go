package stacking

import (
	"immich-go/helpers/gen"
	"immich-go/immich"
	"path"
	"slices"
	"sort"
	"strings"
	"time"
)

type Key struct {
	date     time.Time // time rounded at 5 min
	baseName string    // stack group
}

type Stack struct {
	CoverID string
	IDs     []string
	Date    time.Time
	Names   []string
}

type StackBuilder struct {
	dateRange immich.DateRange // Set capture date range
	stacks    map[Key]Stack
}

func NewStackBuilder() *StackBuilder {
	sb := StackBuilder{
		stacks: map[Key]Stack{},
	}
	sb.dateRange.Set("1850-01-04,2030-01-01")

	return &sb

}

func (sb *StackBuilder) ProcessAsset(ID string, fileName string, captureDate time.Time) {
	if !sb.dateRange.InRange(captureDate) {
		return
	}
	ext := path.Ext(fileName)
	base := strings.TrimSuffix(path.Base(fileName), ext)
	ext = strings.ToLower(ext)

	if idx := strings.Index(base, "_BURST"); idx > 1 {
		base = base[:idx]
	}

	k := Key{
		date:     captureDate.Round(time.Minute),
		baseName: base,
	}
	s, ok := sb.stacks[k]
	if !ok {
		s.CoverID = ID
		s.Date = captureDate
	}
	s.IDs = append(s.IDs, ID)
	s.Names = append(s.Names, path.Base(fileName))
	if slices.Contains([]string{".jpeg", ".jpg", ".jpe"}, ext) || strings.Contains(fileName, "COVER.") {
		s.CoverID = ID
	}
	sb.stacks[k] = s
}

func (sb *StackBuilder) Stacks() []Stack {
	keys := gen.MapFilterKeys(sb.stacks, func(i Stack) bool {
		return len(i.IDs) > 1
	})

	r := []Stack{}
	for _, v := range sb.stacks {
		if len(v.IDs) > 1 {
			r = append(r, v)
		}
	}
	sort.Slice(r, func(i, j int) bool {
		c := keys[i].date.Compare(keys[j].date)
		switch c {
		case -1:
			return true
		case +1:
			return false
		}
		c = strings.Compare(keys[i].baseName, keys[j].baseName)
		switch c {
		case -1:
			return true
		}
		return false
	})
	return r
}
