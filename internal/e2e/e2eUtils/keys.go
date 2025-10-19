package e2eutils

import (
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Keys map[string]any

func KeysFromFile(path string) (Keys, error) {
	// read the keys from the file
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var m map[string]any
	if err := yaml.Unmarshal(b, &m); err != nil {
		return nil, err
	}

	return m, nil
}

// Get retrieves the value associated with the given key path (using "/" as a separator for nested keys).
// If the key is not found or the value is not a string, it returns an empty string.
func (keys Keys) Get(k string) string {
	if keys == nil {
		return ""
	}

	k = strings.TrimSpace(k)
	k = strings.Trim(k, "/")
	if k == "" {
		return ""
	}

	segments := strings.Split(k, "/")
	curMap := keys
	for i, seg := range segments {
		v, ok := curMap[seg]
		if !ok {
			return ""
		}
		if i == len(segments)-1 {
			if s, ok := v.(string); ok {
				return s
			}
			return ""
		} else {
			if m, ok := v.(map[string]any); ok {
				curMap = m
			} else {
				return ""
			}
		}
	}
	return ""
}
