package tzone

import (
	"strings"
	"sync"
	"time"

	"github.com/thlib/go-timezone-local/tzlocal"
)

var (
	_local       *time.Location
	_err         error
	onceSetLocal sync.Once
)

// return the local location
// use tzlocal package
//
//	to determine the local even on Windows
//	check the env variable TZ
func Local() (*time.Location, error) {
	onceSetLocal.Do(func() {
		var tz string
		tz, _err = tzlocal.RuntimeTZ()
		if _err != nil {
			return
		}

		_local, _err = time.LoadLocation(strings.TrimSuffix(tz, "\n"))
	})
	return _local, _err
}
