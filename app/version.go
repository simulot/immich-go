package app

import (
	"context"
	"fmt"
	"runtime/debug"
	"strings"

	"github.com/spf13/cobra"
)

var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// initialize version and commit at the runtime
func init() {
	dirty := false
	buildvcs := false

	buildinfo, _ := debug.ReadBuildInfo()
	for _, s := range buildinfo.Settings {
		switch s.Key {
		case "vcs.revision":
			buildvcs = true
			Commit = s.Value
		case "vcs.modified":
			if s.Value == "true" {
				dirty = true
			}
		case "vcs.time":
			Date = s.Value
		}
	}
	if buildvcs && dirty {
		Commit += "-dirty"
	}
}

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

func GetVersion() string {
	return fmt.Sprintf("immich-go version:%s,  commit:%s, date:%s", Version, Commit, Date)
}

// NewUploadCommand adds the Upload command
func NewVersionCommand(ctx context.Context, app *Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Give immich-go version",
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		fmt.Println(GetVersion())
		return nil
	}
	return cmd
}
