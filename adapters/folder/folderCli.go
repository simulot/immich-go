package folder

import (
	"fmt"
	"strings"
)

// AlbumFolderMode represents the mode in which album folders are organized.
// Implement the interface pflag.Value

type AlbumFolderMode string

const (
	FolderModeNone   AlbumFolderMode = "NONE"
	FolderModeFolder AlbumFolderMode = "FOLDER"
	FolderModePath   AlbumFolderMode = "PATH"
)

func (m AlbumFolderMode) String() string {
	return string(m)
}

func (m *AlbumFolderMode) Set(v string) error {
	v = strings.TrimSpace(strings.ToUpper(v))
	switch v {
	case string(FolderModeFolder), string(FolderModePath), string(FolderModeNone):
		*m = AlbumFolderMode(v)
	default:
		return fmt.Errorf("invalid value for folder mode, expected %s, %s or %s", FolderModeFolder, FolderModePath, FolderModeNone)
	}
	return nil
}

func (m AlbumFolderMode) Type() string {
	return "folderMode"
}

// MarshalJSON implements json.Marshaler
func (m AlbumFolderMode) MarshalJSON() ([]byte, error) {
	return []byte(`"` + m.String() + `"`), nil
}

// UnmarshalJSON implements json.Unmarshaler
func (m *AlbumFolderMode) UnmarshalJSON(data []byte) error {
	if len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return fmt.Errorf("invalid JSON string for AlbumFolderMode")
	}
	s := string(data[1 : len(data)-1])
	return m.Set(s)
}

// MarshalYAML implements yaml.Marshaler
func (m AlbumFolderMode) MarshalYAML() (interface{}, error) {
	return m.String(), nil
}

// UnmarshalYAML implements yaml.Unmarshaler
func (m *AlbumFolderMode) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	return m.Set(s)
}

// MarshalText implements encoding.TextMarshaler
func (m AlbumFolderMode) MarshalText() ([]byte, error) {
	return []byte(m.String()), nil
}

// UnmarshalText implements encoding.TextUnmarshaler
func (m *AlbumFolderMode) UnmarshalText(data []byte) error {
	return m.Set(string(data))
}
