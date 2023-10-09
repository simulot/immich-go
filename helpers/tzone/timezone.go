package tzone

import (
	"strings"
	"sync"
	"time"
)

var (
	_local       *time.Location
	_err         error
	onceSetLocal sync.Once
)

func Local() (*time.Location, error) {
	onceSetLocal.Do(func() {
		var tz string
		tz, _err = getTimezoneName()
		if _err != nil {
			return
		}

		_local, _err = time.LoadLocation(strings.TrimSuffix(tz, "\n"))
	})
	return _local, _err
}
