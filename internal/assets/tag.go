package assets

import "path"

type Tag struct {
	ID    string `json:"-"`               // Tag ID in immich
	Name  string `json:"-"`               // the leaf name of the tag: subtag
	Value string `json:"value,omitempty"` // the full tag name rootTag/subtag
}

func (t Tag) LogValuer() string {
	return t.Value
}

func (a *Asset) AddTag(tag string) {
	for _, t := range a.Tags {
		if t.Value == tag {
			return
		}
	}
	a.Tags = append(a.Tags, Tag{Name: path.Base(tag), Value: tag})
}
