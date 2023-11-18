package browser

import (
	"context"
	"errors"
	"fmt"
	"immich-go/helpers/fshelper"
	"strings"
)

type Browser interface {
	Browse(cxt context.Context) chan *LocalAssetFile
}

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

	c.SelectExtensions, err = checkExtensions(c.SelectExtensions)
	jerr = errors.Join(jerr, fmt.Errorf("some selected extensions are unknown: %w", err))

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
