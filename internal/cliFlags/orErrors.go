package cliflags

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/pelletier/go-toml/v2"
	"github.com/spf13/pflag"
)

type OnErrorsFlag int

const (
	OnErrorsStop OnErrorsFlag = iota
	OnErrorsStopAfter
	OnErrorsNeverStop = -1
)

func (f *OnErrorsFlag) RegisterFlags(fs *pflag.FlagSet, prefix string) {
	fs.Var(f, prefix+"on-errors", "Action to take on errors, (stop|continue| <n> errors)")
}

func (f OnErrorsFlag) String() string {
	switch {
	case f == OnErrorsStop:
		return "stop"
	case f == OnErrorsNeverStop:
		return "continue"
	case f >= OnErrorsStopAfter:
		return fmt.Sprintf("%d", f)
	default:
		return "unknown"
	}
}

func (f *OnErrorsFlag) Set(value string) error {
	switch strings.ToLower(value) {
	case "stop":
		*f = OnErrorsStop
	case "continue":
		*f = OnErrorsNeverStop
	default:
		n, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid value for on-server-errors: %s", value)
		}
		*f = OnErrorsFlag(n)
	}
	return nil
}

func (OnErrorsFlag) Type() string {
	return "OnErrorsFlag"
}

// MarshalJSON implements the json.Marshaler interface.
func (f OnErrorsFlag) MarshalJSON() ([]byte, error) {
	return json.Marshal(f.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (f *OnErrorsFlag) UnmarshalJSON(data []byte) error {
	var value string
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	return f.Set(value)
}

// MarshalYAML implements the yaml.Marshaler interface.
func (f OnErrorsFlag) MarshalYAML() (interface{}, error) {
	return f.String(), nil
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (f *OnErrorsFlag) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var value string
	if err := unmarshal(&value); err != nil {
		return err
	}
	return f.Set(value)
}

// MarshalTOML implements the toml.Marshaler interface.
func (f OnErrorsFlag) MarshalTOML() ([]byte, error) {
	return toml.Marshal(f.String())
}

// UnmarshalTOML implements the toml.Unmarshaler interface.
func (f *OnErrorsFlag) UnmarshalTOML(data []byte) error {
	var value string
	if err := toml.Unmarshal(data, &value); err != nil {
		return err
	}
	return f.Set(value)
}

// MarshalText implements encoding.TextMarshaler
func (f OnErrorsFlag) MarshalText() ([]byte, error) {
	return []byte(f.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler
func (f *OnErrorsFlag) UnmarshalText(data []byte) error {
	return f.Set(string(data))
}
