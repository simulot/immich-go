package version

import (
	"fmt"
	"runtime/debug"
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

func GetVersion() string {
	return fmt.Sprintf("immich-go version:%s,  commit:%s, date:%s", Version, Commit, Date)
}
