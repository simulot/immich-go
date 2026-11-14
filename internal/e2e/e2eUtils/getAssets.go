package e2eutils

import (
	"encoding/json"
	"fmt"
)

// Asset represents a simplified Immich asset returned from search
type Asset struct {
	ID               string   `json:"id"`
	DeviceAssetID    string   `json:"deviceAssetId"`
	DeviceID         string   `json:"deviceId"`
	Type             string   `json:"type"`
	OriginalPath     string   `json:"originalPath"`
	OriginalFileName string   `json:"originalFileName"`
	Resized          bool     `json:"resized"`
	Thumbhash        string   `json:"thumbhash"`
	FileCreatedAt    string   `json:"fileCreatedAt"`
	FileModifiedAt   string   `json:"fileModifiedAt"`
	LocalDateTime    string   `json:"localDateTime"`
	UpdatedAt        string   `json:"updatedAt"`
	IsFavorite       bool     `json:"isFavorite"`
	IsArchived       bool     `json:"isArchived"`
	IsTrashed        bool     `json:"isTrashed"`
	Duration         string   `json:"duration"`
	Checksum         string   `json:"checksum"`
	LivePhotoVideoID string   `json:"livePhotoVideoId"`
	Tags             []string `json:"tags"`
	Rating           int      `json:"rating"`
	Visibility       string   `json:"visibility"`
}

// Stack is a stack of assets as returned by /stacks
type Stack struct {
	ID             string   `json:"id"`
	PrimaryAssetID string   `json:"primaryAssetId"`
	Assets         []*Asset `json:"assets"`
}

// GetAllStacks retrieves all stacks of a user
func GetAllStacks(email, password string) ([]*Stack, error) {
	token, err := UserLogin(email, password)
	if err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}
	resp, err := get(getAPIURL()+"/stacks", token)
	if err != nil {
		return nil, fmt.Errorf("failed to get stacks: %w", err)
	}
	defer resp.Body.Close()
	var stacks []*Stack
	if err := json.NewDecoder(resp.Body).Decode(&stacks); err != nil {
		return nil, fmt.Errorf("failed to decode stacks: %w", err)
	}
	return stacks, nil
}

// SearchMetadataRequest represents the request body for /search/metadata
type SearchMetadataRequest struct {
	Page        int  `json:"page"`
	Size        int  `json:"size"`
	WithExif    bool `json:"withExif"`
	WithStacked bool `json:"withStacked"`
}

// SearchMetadataResponse represents the response from /search/metadata
type SearchMetadataResponse struct {
	Assets struct {
		Count    int      `json:"count"`
		Items    []*Asset `json:"items"`
		NextPage int      `json:"nextPage"`
	} `json:"assets"`
}

// GetAllAssets retrieves all assets for a user using the search/metadata endpoint
// It ignores albums, exifInfo, owner, and people fields
// Returns a map of assets indexed by OriginalFileName. Assets sharing a name overwrite each other;
// use GetAllAssetList when that matters.
func GetAllAssets(email, password string) (map[string]*Asset, error) {
	list, err := GetAllAssetList(email, password)
	if err != nil {
		return nil, err
	}
	assetsByName := make(map[string]*Asset, len(list))
	for _, asset := range list {
		assetsByName[asset.OriginalFileName] = asset
	}
	return assetsByName, nil
}

// GetAllAssetList retrieves all assets for a user using the search/metadata endpoint, in the
// server's order.
func GetAllAssetList(email, password string) ([]*Asset, error) {
	// Login to get access token
	token, err := UserLogin(email, password)
	if err != nil {
		return nil, fmt.Errorf("failed to login: %w", err)
	}

	list := []*Asset{}
	page := 1
	pageSize := 1000

	for {
		request := SearchMetadataRequest{
			Page:        page,
			Size:        pageSize,
			WithExif:    false, // We don't need exifInfo
			WithStacked: true,
		}

		resp, err := post(getAPIURL()+"/search/metadata", &request, token)
		if err != nil {
			return nil, fmt.Errorf("failed to search assets (page %d): %w", page, err)
		}
		defer resp.Body.Close()

		var searchResp SearchMetadataResponse
		err = json.NewDecoder(resp.Body).Decode(&searchResp)
		if err != nil {
			return nil, fmt.Errorf("failed to decode search response: %w", err)
		}

		list = append(list, searchResp.Assets.Items...)

		// Check if there are more pages
		if searchResp.Assets.NextPage == 0 {
			break
		}
		page = searchResp.Assets.NextPage
	}

	return list, nil
}
