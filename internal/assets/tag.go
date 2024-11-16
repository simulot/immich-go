package assets

type Tag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (t Tag) LogValuer() string {
	return t.Name
}
