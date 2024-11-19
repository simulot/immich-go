package assets

import "path"

type Tag struct {
	ID    string `json:"-"`
	Name  string `json:"-"`
	Value string `json:"value,omitempty"`
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
