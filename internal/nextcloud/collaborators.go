package nextcloud

import (
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
)

type AlbumCollaboratorType int

const (
	AlbumCollaboratorTypeUser    AlbumCollaboratorType = 0
	AlbumCollaboratorTypeGroup   AlbumCollaboratorType = 1
	AlbumCollaboratorTypeLink    AlbumCollaboratorType = 3
	AlbumCollaboratorTypeUnknown AlbumCollaboratorType = -1
)

type AlbumCollaborator struct {
	ID    string
	Label string
	Type  AlbumCollaboratorType
}

type davPropfindMultiStatus struct {
	Responses []struct {
		PropStats []struct {
			Props struct {
				Collaborators struct {
					Items []struct {
						ID    string `xml:"id"`
						Label string `xml:"label"`
						Type  string `xml:"type"`
					} `xml:"collaborator"`
				} `xml:"collaborators"`
			} `xml:"prop"`
		} `xml:"propstat"`
	} `xml:"response"`
}

// GetAlbumCollaborators returns the explicit collaborators stored on a Nextcloud
// Memories album DAV resource.
func GetAlbumCollaborators(ctx context.Context, client *Client, ownerUID string, albumName string) ([]AlbumCollaborator, error) {
	if client == nil {
		return nil, fmt.Errorf("missing Nextcloud client")
	}
	ownerUID = strings.TrimSpace(ownerUID)
	albumName = strings.TrimSpace(albumName)
	if ownerUID == "" {
		return nil, fmt.Errorf("missing album owner uid")
	}
	if albumName == "" {
		return nil, fmt.Errorf("missing album name")
	}

	relativePath := path.Join("remote.php/dav/photos", url.PathEscape(ownerUID), "albums", url.PathEscape(albumName))
	body := strings.NewReader(`<?xml version="1.0" encoding="utf-8"?>
<d:propfind xmlns:d="DAV:" xmlns:nc="http://nextcloud.org/ns">
  <d:prop>
    <nc:collaborators/>
  </d:prop>
</d:propfind>`)
	req, err := client.NewRequest(ctx, "PROPFIND", relativePath, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Depth", "0")
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	req.Header.Set("Accept", "application/xml, text/xml")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusMultiStatus {
		responseBody, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<10))
		return nil, fmt.Errorf("album collaborator lookup failed: %s: %s", resp.Status, strings.TrimSpace(string(responseBody)))
	}

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var multiStatus davPropfindMultiStatus
	decoder := xml.NewDecoder(bytes.NewReader(responseBody))
	if err := decoder.Decode(&multiStatus); err != nil {
		return nil, err
	}

	collaborators := []AlbumCollaborator{}
	for _, response := range multiStatus.Responses {
		for _, propstat := range response.PropStats {
			for _, item := range propstat.Props.Collaborators.Items {
				collaborators = append(collaborators, AlbumCollaborator{
					ID:    strings.TrimSpace(item.ID),
					Label: strings.TrimSpace(item.Label),
					Type:  parseAlbumCollaboratorType(item.Type),
				})
			}
		}
	}
	return collaborators, nil
}

func parseAlbumCollaboratorType(value string) AlbumCollaboratorType {
	value = strings.TrimSpace(strings.ToLower(value))
	switch value {
	case "user":
		return AlbumCollaboratorTypeUser
	case "group":
		return AlbumCollaboratorTypeGroup
	case "link", "public_link", "public-link":
		return AlbumCollaboratorTypeLink
	}
	parsedValue, err := strconv.Atoi(value)
	if err != nil {
		return AlbumCollaboratorTypeUnknown
	}
	switch AlbumCollaboratorType(parsedValue) {
	case AlbumCollaboratorTypeUser, AlbumCollaboratorTypeGroup, AlbumCollaboratorTypeLink:
		return AlbumCollaboratorType(parsedValue)
	default:
		return AlbumCollaboratorTypeUnknown
	}
}
