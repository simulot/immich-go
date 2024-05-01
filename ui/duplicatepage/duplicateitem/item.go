package duplicateitem

import (
	"github.com/simulot/immich-go/immich"
)

type Item struct {
	asset *immich.Asset
}

func (i Item) FilterValue() string {
	return i.asset.OriginalFileName
}

func (i Item) Title() string {
	return i.asset.OriginalFileName
}

func (i Item) Description() string {
	return ""
}
