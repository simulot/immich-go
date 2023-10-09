//go:build windows

package tzone

import (
	"golang.org/x/sys/windows/registry"
)

func getTimezoneName() (string, error) {
	key, err := registry.OpenKey(registry.LOCAL_MACHINE, `SYSTEM\CurrentControlSet\Control\TimeZoneInformation`, registry.QUERY_VALUE)
	if err != nil {
		return "", err
	}
	defer key.Close()

	var timeZoneName string
	if val, valType, err := key.GetStringValue("TimeZoneKeyName"); err == nil && valType == registry.SZ {
		timeZoneName = val
	} else {
		return "", err
	}
	return timeZoneName, nil
}
