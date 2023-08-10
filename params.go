package main

import (
	"strings"
)

func Initialize() (*Application, error) {

}

type StringSliceFlag []string

func (ss *StringSliceFlag) String() string {
	if ss == nil {
		return ""
	}
	return strings.Join(*ss, ",")
}

func (ss *StringSliceFlag) Set(value string) error {
	values := strings.Split(value, ",")
	for _, v := range values {
		*ss = append(*ss, v)
	}
	return nil
}

func (ss *StringSliceFlag) Get() interface{} {
	return []string(*ss)
}

func (ss StringSliceFlag) IsIn(v string) bool {
	for i := range ss {
		if ss[i] == v {
			return true
		}
	}
	return false
}
