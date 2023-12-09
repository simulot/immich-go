package stacking

import (
	"path"
	"regexp"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/simulot/immich-go/helpers/fshelper"
	"github.com/simulot/immich-go/helpers/gen"
	"github.com/simulot/immich-go/immich"
)

type Key struct {
	date     time.Time // time rounded at 5 min
	baseName string    // stack group
}

type Stack struct {
	CoverID   string
	StackType StackType
	IDs       []string
	Date      time.Time
	Names     []string
}

type StackType int

const (
	StackRawJpg StackType = iota
	StackBurst
)

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
	cover := false
	burst := false
	ext := path.Ext(fileName)
	base := strings.TrimSuffix(path.Base(fileName), ext)
	ext = strings.ToLower(ext)

	// Do we recognize a burst pattern?
	for _, matcherFn := range stackMatchers {
		if isBurst, theBase, isCover := matcherFn(path.Base(fileName)); isBurst {
			base = theBase
			cover = isCover
			burst = isBurst
			break
		}
	}

	// may be .MP.jpg
	if !burst {
		ext := path.Ext(base)
		if ext == ".MP" {
			base = strings.TrimSuffix(base, ext)
		}
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
	if burst {
		s.StackType = StackBurst
	}
	if cover {
		s.CoverID = ID
	} else if !burst && slices.Contains([]string{".jpeg", ".jpg", ".jpe"}, ext) {
		s.CoverID = ID
	}
	sb.stacks[k] = s
}

// stackMatcher analyze the name and return
// bool -> true when name is a part of burst
// string -> base name of the burst
// bool -> is this is the cover if the burst
type stackMatcher func(name string) (bool, string, bool)

var stackMatchers = []stackMatcher{huaweiBurst, pixelBurst}

var huaweiBurstRE = regexp.MustCompile(`^(.*)(_BURST\d+)(_COVER)?(\..*)$`)

func huaweiBurst(name string) (bool, string, bool) {
	parts := huaweiBurstRE.FindStringSubmatch(name)
	if len(parts) == 0 {
		return false, "", false
	}
	return true, parts[1], len(parts[3]) > 0
}

var pixelBurstRE = regexp.MustCompile(`^(.*)(.RAW-\d+)(\.MP)?(\.COVER)?(\..*)$`)

func pixelBurst(name string) (bool, string, bool) {
	parts := pixelBurstRE.FindStringSubmatch(name)
	if len(parts) == 0 {
		return false, "", false
	}
	return true, parts[1], len(parts[4]) > 0
}

func (sb *StackBuilder) Stacks() []Stack {
	keys := gen.MapFilterKeys(sb.stacks, func(i Stack) bool {
		return len(i.IDs) > 1
	})

	var stacks []Stack
	for _, k := range keys {
		s := sb.stacks[k]

		// Exclude live photos
		hasPhoto := 0
		hasVideo := 0

		for _, n := range s.Names {
			mime, err := fshelper.MimeFromExt(path.Ext(n))
			if err != nil {
				continue
			}
			s := strings.Split(mime[0], "/")
			switch s[0] {
			case "video":
				hasVideo++
			case "image":
				hasPhoto++
			}
		}

		if hasPhoto == 1 && hasVideo == 1 {
			// oh, a live photo!
			continue
		}

		ids := gen.Filter(s.IDs, func(id string) bool {
			return id != s.CoverID
		})
		s.IDs = ids
		stacks = append(stacks, s)

	}
	sort.Slice(stacks, func(i, j int) bool {
		c := stacks[i].Date.Compare(stacks[j].Date)
		switch c {
		case -1:
			return true
		case +1:
			return false
		}
		c = strings.Compare(stacks[i].Names[0], stacks[j].Names[0])
		switch c {
		case -1:
			return true
		}
		return false
	})
	return stacks
}
