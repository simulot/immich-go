package configuration

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Configuration struct {
	APIURL    string `json:",omitempty"`
	ServerURL string `json:",omitempty"`
	APIKey    string
}

// DefaultFile return the default configuration file name
// Return a local file nama when the default UserHomeDir can't be determined,
func DefaultFile() string {
	d, err := os.UserHomeDir()
	if err != nil {
		return "immich-go.json"
	}
	return filepath.Join(d, ".immich-go", "immich-go.json")
}

// Read the configuration in file name
func Read(name string) (Configuration, error) {
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
