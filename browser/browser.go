package browser

import (
	"context"
	"regexp"
)

type Browser interface {
	Browse(cxt context.Context, excludeFiles []string) chan *LocalAssetFile
}

// Matches takes a directory (not the full path, just the name) and matches it
// against a set of regexp patterns, returning true if matched.
func Matches(directory string, regexpPatterns []string) bool {
	for _, pattern := range regexpPatterns {
		match, err := regexp.MatchString(pattern, directory)
		if err != nil {
			return false
		}
		if match {
			return true
		}
	}
	return false
}
