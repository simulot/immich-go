package e2eutils

import (
	"encoding/json"
	"fmt"
)

// TagSimplified represents a tag returned from the server
type TagSimplified struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Value string `json:"value"`
}

// GetAllTags retrieves all tags for a user
// Returns a slice of tag values
func GetAllTags(email, password string) ([]string, error) {
	// Login to get access token
	token, err := UserLogin(email, password)
	if err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	resp, err := get(getAPIURL()+"/tags", token)
	if err != nil {
		return nil, fmt.Errorf("failed to get tags: %w", err)
	}
	defer resp.Body.Close()

	var tags []TagSimplified
	err = json.NewDecoder(resp.Body).Decode(&tags)
	if err != nil {
		return nil, fmt.Errorf("failed to decode tags response: %w", err)
	}

	tagValues := make([]string, len(tags))
	for i, tag := range tags {
		tagValues[i] = tag.Value
	}

	return tagValues, nil
}
