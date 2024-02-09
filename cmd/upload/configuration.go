package upload

import (
	"errors"
	"fmt"
	"slices"
	"strings"

	"github.com/simulot/immich-go/helpers/fshelper"
)

type Configuration struct {
	SelectExtensions  StringList
	ExcludeExtensions StringList
	Recursive         bool
}

func (c *Configuration) IsValid() error {
	var (
		jerr error
		err  error
	)

	if c.SelectExtensions, err = checkExtensions(c.SelectExtensions); err != nil {
		jerr = errors.Join(jerr, fmt.Errorf("some selected extensions are unknown: %w", err))
	}

	c.ExcludeExtensions, _ = checkExtensions(c.ExcludeExtensions)

	return jerr
}

func checkExtensions(l StringList) (StringList, error) {
	var (
		r   StringList
		err error
	)

	for _, e := range l {
		if !strings.HasPrefix(e, ".") {
			e = "." + e
		}
		e = strings.ToLower(e)
		if _, err = fshelper.MimeFromExt(e); err != nil {
			err = errors.Join(err, fmt.Errorf("unsupported extension '%s'", e))
		}
		r = append(r, e)
	}
	if err != nil {
		return nil, err
	}
	return r, nil
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
