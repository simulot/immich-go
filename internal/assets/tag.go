package assets

type Tag struct {
	ID    string `json:"-"`
	Name  string `json:"-"`
	Value string `json:"value,omitempty"`
}

func (t Tag) LogValuer() string {
	return t.Name
}
