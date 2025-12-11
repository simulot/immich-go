//go:build !darwin
// +build !darwin

package macos

import "errors"

var ErrNotSupported = errors.New("macOS Finder tags are only supported on macOS")

type FinderTag struct {
	Name  string
	Color int
}

var (
	TagRed    = FinderTag{}
	TagOrange = FinderTag{}
	TagYellow = FinderTag{}
	TagGreen  = FinderTag{}
	TagBlue   = FinderTag{}
	TagPurple = FinderTag{}
	TagGray   = FinderTag{}
)

func SetFinderTag(filePath string, tag FinderTag) error {
	return ErrNotSupported
}

func IsSupported() bool {
	return false
}
