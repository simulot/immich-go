package duplicateitem

import (
	"fmt"
	"time"

	"github.com/simulot/immich-go/immich"
)

const TimeFormat = "2006/01/02 15:04:05 Z07:00"

// Group groups duplicated assets
type Group struct {
	Date   time.Time
	Name   string
	Assets []*immich.Asset
}

// Implements the list.Item interface
func (i Group) FilterValue() string {
	return i.Name + i.Date.Format(TimeFormat)
}

func (i Group) Description() string {
	return fmt.Sprintf("%d files", len(i.Assets))
}

func (i Group) Title() string {
	return fmt.Sprintf("%s %s", i.Date.Format(TimeFormat), i.Name)
}
