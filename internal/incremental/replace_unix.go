//go:build !windows

package incremental

import (
	"errors"
	"os"
	"path/filepath"
)

func replaceState(source, target string) error {
	if err := os.Rename(source, target); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(target))
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}
