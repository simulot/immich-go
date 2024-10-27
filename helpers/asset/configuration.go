package asset

import (
	"strings"

	"github.com/simulot/immich-go/helpers/datatype"
)

type Configuration struct {
	SelectExtensions  datatype.StringList
	ExcludeExtensions datatype.StringList
	Recursive         bool
}

func (c *Configuration) Validate() {
	c.SelectExtensions = checkExtensions(c.SelectExtensions)
	c.ExcludeExtensions = checkExtensions(c.ExcludeExtensions)
}

func checkExtensions(l datatype.StringList) datatype.StringList {
	var r datatype.StringList

	for _, e := range l {
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		e = strings.ToLower(e)
		r = append(r, e)
	}
	return r
}
