package nextcloudmemories

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const managedAlbumStateSource = "nextcloud-memories"

const (
	memoriesManagedAlbumStateHeader  = "--- immich-go:nextcloud-memories:v1 ---"
	memoriesManagedAlbumStateFooter  = "--- /immich-go ---"
	memoriesAlbumMembershipTagPrefix = "immich-go/src/nextcloud-memories/album/"
)

var errMalformedManagedAlbumState = errors.New("malformed immich-go Nextcloud Memories album state")

type managedAlbumState struct {
	Source        string `json:"source"`
	SchemaVersion int    `json:"schema_version"`
	AlbumID       int    `json:"album_id"`
	OwnerUID      string `json:"owner_uid,omitempty"`
	AlbumName     string `json:"album_name,omitempty"`
}

func memoriesAlbumMembershipTag(albumID int) string {
	if albumID <= 0 {
		return ""
	}
	return fmt.Sprintf("%s%d", memoriesAlbumMembershipTagPrefix, albumID)
}

func applyManagedAlbumState(description string, state managedAlbumState) (string, error) {
	humanDescription, existingState, err := parseManagedAlbumState(description)
	if err != nil {
		return "", err
	}
	if existingState != nil {
		if existingState.Source != state.Source || existingState.AlbumID != state.AlbumID || existingState.OwnerUID != state.OwnerUID {
			return "", fmt.Errorf("conflicting managed album state: existing source=%q album_id=%d owner_uid=%q", existingState.Source, existingState.AlbumID, existingState.OwnerUID)
		}
	}
	payload, err := json.Marshal(state)
	if err != nil {
		return "", err
	}
	block := strings.Join([]string{
		memoriesManagedAlbumStateHeader,
		string(payload),
		memoriesManagedAlbumStateFooter,
	}, "\n")
	humanDescription = strings.TrimRight(humanDescription, "\n")
	if humanDescription == "" {
		return block, nil
	}
	return humanDescription + "\n\n" + block, nil
}

func parseManagedAlbumState(description string) (string, *managedAlbumState, error) {
	start := strings.Index(description, memoriesManagedAlbumStateHeader)
	if start == -1 {
		return description, nil, nil
	}
	searchFrom := start + len(memoriesManagedAlbumStateHeader)
	endOffset := strings.Index(description[searchFrom:], memoriesManagedAlbumStateFooter)
	if endOffset == -1 {
		return "", nil, errMalformedManagedAlbumState
	}
	end := searchFrom + endOffset
	payload := strings.TrimSpace(description[searchFrom:end])
	if payload == "" {
		return "", nil, errMalformedManagedAlbumState
	}
	var state managedAlbumState
	if err := json.Unmarshal([]byte(payload), &state); err != nil {
		return "", nil, fmt.Errorf("%w: %v", errMalformedManagedAlbumState, err)
	}
	humanDescription := strings.TrimRight(description[:start], "\n")
	return humanDescription, &state, nil
}