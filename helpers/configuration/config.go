package configuration

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type Configuration struct {
	APIURL    string `json:",omitempty"`
	ServerURL string `json:",omitempty"`
	APIKey    string
}

// DefaultConfigFile return the default configuration file name
// Return a local file when the default UserHomeDir can't be determined,
func DefaultConfigFile() string {
	config, err := os.UserConfigDir()
	if err != nil {
		// $XDG_CONFIG_HOME nor $HOME is set
		// return current
		return "./immich-go.json"
	}
	return filepath.Join(config, "immich-go", "immich-go.json")
}

// ConfigRead the configuration in file name
func ConfigRead(name string) (Configuration, error) {
	f, err := os.Open(name)
	if err != nil {
		return Configuration{}, err
	}
	defer f.Close()
	var c Configuration
	err = json.NewDecoder(f).Decode(&c)
	if err != nil {
		return Configuration{}, err
	}
	return c, nil
}

// Write the configuration in the file name
// Create the needed sub directories as needed
func (c Configuration) Write(name string) error {
	d, _ := filepath.Split(name)
	if d != "" {
		err := os.MkdirAll(d, 0o700)
		if err != nil {
			return err
		}
	}
	f, err := os.OpenFile(name, os.O_CREATE|os.O_RDWR|os.O_TRUNC, 0o700)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	return enc.Encode(c)
}

// DefaultLogDir give the default log file
// Return the current dir when $HOME not $XDG_CACHE_HOME are not set
func DefaultLogFile() string {
	f := time.Now().Format("immich-go_2006-01-02_15-04-05.log")
	d, err := os.UserCacheDir()
	if err != nil {
		return f
	}
	return filepath.Join(d, "immich-go", f)
}

// MakeDirForFile create all dirs to write the given file
func MakeDirForFile(f string) error {
	dir := filepath.Dir(f)
	return os.MkdirAll(dir, 0o700)
}
