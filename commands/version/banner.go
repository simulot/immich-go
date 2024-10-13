package version

import (
	"fmt"
	"strings"
)

// Banner Ascii art
// Generator : http://patorjk.com/software/taag-v1/
// Font: Three point

var _banner = []string{
	". _ _  _ _ . _|_     _  _ ",
	"|| | || | ||(_| | ─ (_|(_)",
	"                     _)   ",
}

// String generate a string with new lines and place the given text on the latest line
func Banner() string {
	const lenVersion = 20
	var text string
	if Version != "" {
		text = fmt.Sprintf("v %s", Version)
	}
	sb := strings.Builder{}
	for i := range _banner {
		if i == len(_banner)-1 && text != "" {
			if len(text) >= lenVersion {
				text = text[:lenVersion]
			}
			sb.WriteString(_banner[i][:lenVersion-len(text)] + text + _banner[i][lenVersion:])
		} else {
			sb.WriteString(_banner[i])
		}
		sb.WriteRune('\n')
	}
	return sb.String()
}
