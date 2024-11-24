package tzone

import (
	"strings"
	"time"

	"github.com/thlib/go-timezone-local/tzlocal"
)

type Timezone struct {
	name string
	TZ   *time.Location
}

func (tz *Timezone) Set(tzName string) error {
	var err error

	tzName = strings.TrimSpace(tzName)
	switch strings.ToUpper(tzName) {
	case "LOCAL":
		tzName, err = tzlocal.RuntimeTZ()
		if err != nil {
			return err
		}
		tz.name = "Local"
	case "UTC":
		tzName = "UTC"
	default:
		tz.name = tzName
	}
	tz.TZ, err = time.LoadLocation(tzName)
	return err
}

func (tz *Timezone) String() string {
	return tz.name
}

func (tz *Timezone) Type() string {
	return "timezone"
}

func (tz *Timezone) Location() *time.Location {
	return tz.TZ
}
