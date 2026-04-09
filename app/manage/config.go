package manage

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// SyncMode controls how multiple people are matched when searching for assets.
type SyncMode string

const (
	// ModeAll returns only photos where all listed people appear together.
	ModeAll SyncMode = "all"
	// ModeAny returns photos where any listed person appears, even solo.
	ModeAny SyncMode = "any"
)

// PeopleSelector identifies people by name and/or Immich UUID.
type PeopleSelector struct {
	Names []string `yaml:"names"`
	IDs   []string `yaml:"ids"`
}

// AlbumMapping maps one album to a set of people whose photos should be in it.
type AlbumMapping struct {
	Album   string         `yaml:"album,omitempty"`    // album by name (resolved at runtime)
	AlbumID string         `yaml:"album_id,omitempty"` // album by Immich UUID
	People  PeopleSelector `yaml:"people"`
	Mode    SyncMode       `yaml:"mode,omitempty"` // "all" (default): only photos with everyone; "any": photos with any listed person
}

// ManageConfig is the top-level YAML configuration for the manage command.
// Each subcommand has its own section.
type ManageConfig struct {
	PeopleAlbumSync *PeopleAlbumSyncConfig `yaml:"people-album-sync"`
}

// PeopleAlbumSyncConfig holds the configuration for the people-album-sync subcommand.
type PeopleAlbumSyncConfig struct {
	Albums []AlbumMapping `yaml:"albums"`
}

// ParseConfig reads and validates a YAML config file.
func ParseConfig(path string) (*ManageConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg ManageConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return &cfg, nil
}

func (c *PeopleAlbumSyncConfig) validate() error {
	if len(c.Albums) == 0 {
		return fmt.Errorf("config: people-album-sync.albums list is empty")
	}
	for i := range c.Albums {
		a := &c.Albums[i]
		if a.Album == "" && a.AlbumID == "" {
			return fmt.Errorf("config: people-album-sync.albums entry %d has no 'album' name or 'album_id'", i)
		}
		if a.Album != "" && a.AlbumID != "" {
			return fmt.Errorf("config: people-album-sync.albums entry %d has both 'album' and 'album_id'; specify only one", i)
		}
		if len(a.People.Names) == 0 && len(a.People.IDs) == 0 {
			return fmt.Errorf("config: people-album-sync.albums entry %d (%s) has no people specified", i, a.albumLabel())
		}
		if a.Mode != "" && a.Mode != ModeAll && a.Mode != ModeAny {
			return fmt.Errorf("config: people-album-sync.albums entry %d (%s) has invalid mode %q (must be \"all\" or \"any\")", i, a.albumLabel(), a.Mode)
		}
		if a.Mode == "" {
			a.Mode = ModeAll
		}
	}
	return nil
}

func (a *AlbumMapping) albumLabel() string {
	if a.Album != "" {
		return a.Album
	}
	return a.AlbumID
}
