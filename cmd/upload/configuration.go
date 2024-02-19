package upload

import (
	"slices"
	"strings"
)

type Configuration struct {
	SelectExtensions  StringList
	ExcludeExtensions StringList
	Recursive         bool
}

func (c *Configuration) Validate() {
	c.SelectExtensions = checkExtensions(c.SelectExtensions)
	c.ExcludeExtensions = checkExtensions(c.ExcludeExtensions)
}

func checkExtensions(l StringList) StringList {
	var r StringList

	for _, e := range l {
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		e = strings.ToLower(e)
		r = append(r, e)
	}
	return r
}

type StringList []string

func (sl *StringList) Set(s string) error {
	l := strings.Split(s, ",")
	(*sl) = append((*sl), l...)
	return nil
}

func (sl StringList) String() string {
	return strings.Join(sl, ", ")
}

func (sl StringList) Include(s string) bool {
	if len(sl) == 0 {
		return true
	}
	return slices.Contains(sl, strings.ToLower(s))
}

func (sl StringList) Exclude(s string) bool {
	if len(sl) == 0 {
		return false
	}
	return slices.Contains(sl, strings.ToLower(s))
}
