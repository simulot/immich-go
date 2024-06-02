package myflag

import (
	"fmt"
	"strings"
	"time"
)

func DurationFlagFn(flag *time.Duration, defaultValue time.Duration) func(string) error {
	*flag = defaultValue
	return func(v string) error {
		v = strings.ToLower(v)
		d, err := time.ParseDuration(v)
		if err != nil {
			return fmt.Errorf("can't parse the duration parameter: %w", err)
		}
		*flag = d
		return nil
	}
}
