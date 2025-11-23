package services

// ColorToken enumerates semantic color usages that each UI shell must map to concrete values.
type ColorToken string

const (
	ColorPrimary   ColorToken = "primary"
	ColorSuccess   ColorToken = "success"
	ColorWarning   ColorToken = "warning"
	ColorDanger    ColorToken = "danger"
	ColorMuted     ColorToken = "muted"
	ColorHighlight ColorToken = "highlight"
)

// Theme defines a palette + typography scale shared between shells.
type Theme struct {
	Colors map[ColorToken]string
	Font   string
	Scale  map[string]int
}

// DefaultTheme returns the neutral palette used when no overrides are provided.
func DefaultTheme() Theme {
	return Theme{
		Colors: map[ColorToken]string{
			ColorPrimary:   "#5E81AC",
			ColorSuccess:   "#A3BE8C",
			ColorWarning:   "#EBCB8B",
			ColorDanger:    "#BF616A",
			ColorMuted:     "#4C566A",
			ColorHighlight: "#88C0D0",
		},
		Font: "monospace",
		Scale: map[string]int{
			"xs": 8,
			"sm": 10,
			"md": 12,
			"lg": 14,
			"xl": 16,
		},
	}
}
