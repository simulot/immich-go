//go:build darwin
// +build darwin

package macos

import (
	"fmt"
	"os/exec"
	"path/filepath"
)

type FinderTag struct {
	Name  string
	Color int
}

var (
	TagRed    = FinderTag{Name: "Rot", Color: 6}
	TagOrange = FinderTag{Name: "Orange", Color: 7}
	TagYellow = FinderTag{Name: "Gelb", Color: 5}
	TagGreen  = FinderTag{Name: "Grün", Color: 2}
	TagBlue   = FinderTag{Name: "Blau", Color: 4}
	TagPurple = FinderTag{Name: "Violett", Color: 3}
	TagGray   = FinderTag{Name: "Grau", Color: 1}
)

func SetFinderTag(filePath string, tag FinderTag) error {
	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	script := fmt.Sprintf(`
		tell application "Finder"
			set theFile to POSIX file "%s" as alias
			set label index of theFile to %d
		end tell
	`, absPath, tag.Color)

	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to set Finder tag: %w", err)
	}

	return nil
}

func IsSupported() bool {
	cmd := exec.Command("osascript", "-e", "return 1")
	return cmd.Run() == nil
}
