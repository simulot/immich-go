package assets

type Tag struct {
	ID    string `json:"id,omitempty"`
	Name  string `json:"name,omitempty"`
	Value string `json:"value,omitempty"`
}

func (t Tag) LogValuer() string {
	return t.Name
}
