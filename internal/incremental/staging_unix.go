//go:build !windows

package incremental

import (
	"errors"
	"math"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

func stagingSupported() error { return nil }

func availableStagingBytes(directory string) (uint64, error) {
	var stat unix.Statfs_t
	if err := unix.Statfs(directory, &stat); err != nil {
		return 0, err
	}
	return availableBlockBytes(stat.Bavail, stat.Bsize)
}

// Statfs uses signed available blocks on FreeBSD and unsigned blocks on Linux
// and Darwin. Validate before conversion so a negative quota cannot look free.
func availableBlockBytes[B ~int64 | ~uint64, S ~int32 | ~int64 | ~uint32 | ~uint64](blocks B, blockSize S) (uint64, error) {
	if blocks < 0 || blockSize <= 0 {
		return 0, errors.New("invalid filesystem availability measurement")
	}
	available, size := uint64(blocks), uint64(blockSize)
	if available > math.MaxUint64/size {
		return 0, errors.New("filesystem availability measurement overflows bytes")
	}
	return available * size, nil
}

func syncStagingParent(directory string) error {
	dir, err := os.Open(filepath.Dir(directory))
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}

func syncStage(root *os.Root) error {
	dir, err := root.Open(".")
	if err != nil {
		return err
	}
	return errors.Join(dir.Sync(), dir.Close())
}
