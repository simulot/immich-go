package ui

import "strings"

type banner []string

// Banner Ascii art
// Generator : http://patorjk.com/software/taag-v1/
// Font: Three point
var Banner = banner{
	". _ _  _ _ . _|_  __  _  _ ",
	"|| | || | ||(_| |    (_|(_)",
	"                      _)   ",
}

// ToString generate a string with new lines and place the given text on the latest line
func (b banner) ToString(text string) string {
	sb := strings.Builder{}
	for i := range b {
		sb.WriteString(b[i])
		if i == len(b)-1 && text != "" {
			sb.WriteString("  ")
			sb.WriteString(text)
		}
		sb.WriteRune('\n')
	}
	return sb.String()
}
