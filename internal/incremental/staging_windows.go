package incremental

import (
	"errors"
	"os"
)

// Windows requires ACL-based privacy and a durable, handle-relative publication
// implementation. POSIX permission bits and directory Sync cannot provide those
// guarantees. Default nonstaging imports still use replace_windows.go normally.
func stagingSupported() error {
	return errors.New("durable media staging is unsupported on Windows: private ACL and durable publication support are required; use re-present mode without staging, or keep this staged state on its original supported filesystem")
}

func syncStage(_ *os.Root) error { return stagingSupported() }

func syncStagingParent(_ string) error { return stagingSupported() }

func availableStagingBytes(_ string) (uint64, error) { return 0, stagingSupported() }
