package myflag

import (
	"fmt"
	"strconv"
	"strings"
)

// BoolFlagFn returns a convenient function for handling boolean option on CLI to be used as parameter of the flag.BoolFn.
// It works has the flag.BoolVar but, the presence of the flag, without value, set the flag to True

func BoolFlagFn(b *bool, defaultValue bool) func(string) error {
	*b = defaultValue
	return func(v string) error {

		switch strings.ToLower(v) {
		case "":
			*b = true
			return nil
		default:
			var err error
			*b, err = strconv.ParseBool(v)
			if err != nil {
				err = fmt.Errorf("can't parse the parameter value: %w", err)
			}
			return err
		}
	}

}
