package datatype

import (
	"slices"
	"strings"
)

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
