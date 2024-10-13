package fileevent

import (
	"io/fs"
	"log/slog"

	"github.com/simulot/immich-go/internal/fshelper"
)

type FileAndName struct {
	fsys fs.FS
	name string
}

func (fn FileAndName) LogValue() slog.Value {
	return slog.StringValue(fn.Name())
}

func AsFileAndName(fsys fs.FS, name string) FileAndName {
	return FileAndName{fsys: fsys, name: name}
}

func (fn FileAndName) Name() string {
	fsys := fn.fsys
	if fsys, ok := fsys.(fshelper.NameFS); ok {
		return fsys.Name() + ":" + fn.name
	}
	return fn.name
}
