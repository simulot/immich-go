package nextcloud

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	pathpkg "path"
	"slices"
	"strconv"
	"strings"
)

// OCSCapabilities contains the subset of Nextcloud capability metadata needed by
// the importer discovery flow.
type OCSCapabilities struct {
	VersionString string
	Edition       string
	ProductName   string
}

// MemoriesDescribe contains the public API description exposed by Memories.
type MemoriesDescribe struct {
	Version      string  `json:"version"`
	BaseURL      string  `json:"baseUrl"`
	LoginFlowURL string  `json:"loginFlowUrl"`
	UID          *string `json:"uid"`
}

// MemoriesConfig contains the subset of user config needed to scope imports and
// explain source-side limitations.
type MemoriesConfig struct {
	Version                  string `json:"version"`
	TimelinePath             string `json:"timeline_path"`
	FoldersPath              string `json:"folders_path"`
	AlbumsEnabled            bool   `json:"albums_enabled"`
	SystemTagsEnabled        bool   `json:"systemtags_enabled"`
	PreviewGeneratorEnabled  bool   `json:"preview_generator_enabled"`
	RecognizeInstalled       bool   `json:"recognize_installed"`
	RecognizeEnabled         bool   `json:"recognize_enabled"`
	FaceRecognitionInstalled bool   `json:"facerecognition_installed"`
	FaceRecognitionEnabled   bool   `json:"facerecognition_enabled"`
}

// MemoriesDiscovery combines the stable Nextcloud and app-specific Memories
// information needed by the discovery-only flow.
type MemoriesDiscovery struct {
	BaseURL       string
	DAVRoot       string
	Capabilities  OCSCapabilities
	Describe      MemoriesDescribe
	Config        MemoriesConfig
	TimelineRoots []string
}

// MemoriesTimelineQuery scopes day and photo listing requests to the same
// effective views exposed by the Memories timeline APIs.
type MemoriesTimelineQuery struct {
	Folder    string
	Recursive bool
	Archive   bool
	Hidden    bool
}

// MemoriesImageInfoQuery controls optional expansions for per-file image info.
type MemoriesImageInfoQuery struct {
	Tags     bool
	Clusters []string
}

// MemoriesDay represents a single day bucket from the Memories timeline API.
type MemoriesDay struct {
	DayID int `json:"dayid"`
	Count int `json:"count"`
}

// MemoriesPhoto represents a file returned by the Memories day listing API.
type MemoriesPhoto struct {
	FileID     int          `json:"fileid"`
	Basename   string       `json:"basename"`
	MimeType   string       `json:"mimetype"`
	DayID      int          `json:"dayid"`
	DateTaken  int64        `json:"datetaken"`
	Archived   bool         `json:"-"`
	IsFavorite memoriesBool `json:"isfavorite,omitempty"`
	IsHidden   memoriesBool `json:"ishidden,omitempty"`
}

// MemoriesAlbum is the subset of album data returned by image info cluster
// expansions and the album list API that is needed for import mapping.
type MemoriesAlbum struct {
	AlbumID     int          `json:"album_id"`
	ClusterID   string       `json:"cluster_id"`
	Name        string       `json:"name"`
	User        string       `json:"user"`
	UserDisplay string       `json:"user_display"`
	Shared      memoriesBool `json:"shared"`
	Location    string       `json:"location"`
}

// MemoriesImageInfo contains the file path and per-file metadata used to enrich
// imported assets.
type MemoriesImageInfo struct {
	FileID    int            `json:"fileid"`
	DateTaken int64          `json:"datetaken"`
	Basename  string         `json:"basename"`
	MimeType  string         `json:"mimetype"`
	FileName  string         `json:"filename,omitempty"`
	Tags      memoriesTags   `json:"tags,omitempty"`
	Exif      map[string]any `json:"exif,omitempty"`
	Clusters  struct {
		Albums []MemoriesAlbum `json:"albums,omitempty"`
	} `json:"clusters,omitempty"`
}

type memoriesTags map[string]string

func (t *memoriesTags) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*t = nil
		return nil
	}

	var objectValue map[string]string
	if err := json.Unmarshal(data, &objectValue); err == nil {
		*t = objectValue
		return nil
	}

	var stringList []string
	if err := json.Unmarshal(data, &stringList); err == nil {
		mapped := make(map[string]string, len(stringList))
		for i, value := range stringList {
			value = strings.TrimSpace(value)
			if value == "" {
				continue
			}
			mapped[strconv.Itoa(i)] = value
		}
		*t = mapped
		return nil
	}

	var mixedList []any
	if err := json.Unmarshal(data, &mixedList); err == nil {
		mapped := make(map[string]string, len(mixedList))
		for i, item := range mixedList {
			switch value := item.(type) {
			case string:
				value = strings.TrimSpace(value)
				if value != "" {
					mapped[strconv.Itoa(i)] = value
				}
			case map[string]any:
				for _, key := range []string{"name", "label", "value"} {
					if raw, ok := value[key]; ok {
						if text, ok := raw.(string); ok {
							text = strings.TrimSpace(text)
							if text != "" {
								mapped[strconv.Itoa(i)] = text
								break
							}
						}
					}
				}
			}
		}
		*t = mapped
		return nil
	}

	return fmt.Errorf("invalid Memories tags payload %q", trimmed)
}

type memoriesBool bool

func (b *memoriesBool) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		*b = false
		return nil
	}

	var boolValue bool
	if err := json.Unmarshal(data, &boolValue); err == nil {
		*b = memoriesBool(boolValue)
		return nil
	}

	var intValue int
	if err := json.Unmarshal(data, &intValue); err == nil {
		*b = memoriesBool(intValue != 0)
		return nil
	}

	var stringValue string
	if err := json.Unmarshal(data, &stringValue); err == nil {
		switch strings.TrimSpace(strings.ToLower(stringValue)) {
		case "1", "true", "yes", "y":
			*b = true
			return nil
		case "0", "false", "no", "n", "":
			*b = false
			return nil
		}
	}

	return fmt.Errorf("invalid Memories boolean value %q", trimmed)
}

type ocsEnvelope[T any] struct {
	OCS struct {
		Meta ocsMeta `json:"meta"`
		Data T       `json:"data"`
	} `json:"ocs"`
}

type ocsMeta struct {
	Status     string `json:"status"`
	StatusCode int    `json:"statuscode"`
	Message    string `json:"message"`
}

// GetOCSCapabilities validates OCS access and returns a small subset of server
// metadata useful in discovery output.
func GetOCSCapabilities(ctx context.Context, client *Client) (*OCSCapabilities, error) {
	req, err := client.NewOCSRequest(ctx, http.MethodGet, "cloud/capabilities?format=json", nil)
	if err != nil {
		return nil, err
	}

	type capabilitiesData struct {
		Version struct {
			String      string `json:"string"`
			Edition     string `json:"edition"`
			ProductName string `json:"productname"`
		} `json:"version"`
	}

	var envelope ocsEnvelope[capabilitiesData]
	if err := doJSON(req, client, &envelope); err != nil {
		return nil, err
	}
	if !strings.EqualFold(envelope.OCS.Meta.Status, "ok") || !isSuccessfulOCSStatusCode(envelope.OCS.Meta.StatusCode) {
		return nil, fmt.Errorf("OCS capabilities request failed: %s (%d)", envelope.OCS.Meta.Message, envelope.OCS.Meta.StatusCode)
	}

	return &OCSCapabilities{
		VersionString: envelope.OCS.Data.Version.String,
		Edition:       envelope.OCS.Data.Version.Edition,
		ProductName:   envelope.OCS.Data.Version.ProductName,
	}, nil
}

func isSuccessfulOCSStatusCode(statusCode int) bool {
	switch statusCode {
	case 100, 200:
		return true
	default:
		return false
	}
}

// DescribeMemories fetches the public API description exposed by the Memories app.
func DescribeMemories(ctx context.Context, client *Client) (*MemoriesDescribe, error) {
	req, err := client.NewMemoriesRequest(ctx, http.MethodGet, "api/describe", nil)
	if err != nil {
		return nil, err
	}

	var describe MemoriesDescribe
	if err := doJSON(req, client, &describe); err != nil {
		return nil, err
	}
	if describe.Version == "" {
		return nil, errors.New("memories describe response did not include a version")
	}
	return &describe, nil
}

// GetMemoriesConfig fetches the current user's Memories configuration.
func GetMemoriesConfig(ctx context.Context, client *Client) (*MemoriesConfig, error) {
	req, err := client.NewMemoriesRequest(ctx, http.MethodGet, "api/config", nil)
	if err != nil {
		return nil, err
	}

	var config MemoriesConfig
	if err := doJSON(req, client, &config); err != nil {
		return nil, err
	}
	return &config, nil
}

// GetMemoriesDays lists all day buckets visible in the requested timeline scope.
func GetMemoriesDays(ctx context.Context, client *Client, query MemoriesTimelineQuery) ([]MemoriesDay, error) {
	req, err := client.NewMemoriesRequest(ctx, http.MethodGet, "api/days", nil)
	if err != nil {
		return nil, err
	}
	applyMemoriesTimelineQuery(req.URL.Query(), req.URL, query)

	var days []MemoriesDay
	if err := doJSON(req, client, &days); err != nil {
		return nil, err
	}
	return days, nil
}

// GetMemoriesDay lists all files for the requested Memories day IDs.
func GetMemoriesDay(ctx context.Context, client *Client, dayIDs []int, query MemoriesTimelineQuery) ([]MemoriesPhoto, error) {
	if len(dayIDs) == 0 {
		return nil, nil
	}

	parts := make([]string, 0, len(dayIDs))
	for _, dayID := range dayIDs {
		parts = append(parts, strconv.Itoa(dayID))
	}

	req, err := client.NewMemoriesRequest(ctx, http.MethodGet, "api/days/"+strings.Join(parts, ","), nil)
	if err != nil {
		return nil, err
	}
	applyMemoriesTimelineQuery(req.URL.Query(), req.URL, query)

	var photos []MemoriesPhoto
	if err := doJSON(req, client, &photos); err != nil {
		return nil, err
	}
	return photos, nil
}

// GetMemoriesImageInfo loads the detailed metadata for a single indexed file.
func GetMemoriesImageInfo(ctx context.Context, client *Client, fileID int, query MemoriesImageInfoQuery) (*MemoriesImageInfo, error) {
	req, err := client.NewMemoriesRequest(ctx, http.MethodGet, fmt.Sprintf("api/image/info/%d", fileID), nil)
	if err != nil {
		return nil, err
	}
	params := req.URL.Query()
	if query.Tags {
		params.Set("tags", "1")
	}
	clusters := make([]string, 0, len(query.Clusters))
	for _, cluster := range query.Clusters {
		cluster = strings.TrimSpace(cluster)
		if cluster == "" {
			continue
		}
		clusters = append(clusters, cluster)
	}
	if len(clusters) > 0 {
		params.Set("clusters", strings.Join(clusters, ","))
	}
	req.URL.RawQuery = params.Encode()

	var info MemoriesImageInfo
	if err := doJSON(req, client, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// DiscoverMemories collects both OCS and Memories app metadata to drive the
// discovery-only command flow.
func DiscoverMemories(ctx context.Context, client *Client) (*MemoriesDiscovery, error) {
	capabilities, err := GetOCSCapabilities(ctx, client)
	if err != nil {
		return nil, err
	}
	describe, err := DescribeMemories(ctx, client)
	if err != nil {
		return nil, err
	}
	config, err := GetMemoriesConfig(ctx, client)
	if err != nil {
		return nil, err
	}

	timelineRoots := splitTimelineRoots(config.TimelinePath)
	if len(timelineRoots) == 0 {
		return nil, errors.New("memories config did not provide any timeline roots")
	}

	return &MemoriesDiscovery{
		BaseURL:       client.BaseURL(),
		DAVRoot:       client.DAVRoot(),
		Capabilities:  *capabilities,
		Describe:      *describe,
		Config:        *config,
		TimelineRoots: timelineRoots,
	}, nil
}

func splitTimelineRoots(raw string) []string {
	parts := strings.Split(raw, ";")
	roots := make([]string, 0, len(parts))

	for _, part := range parts {
		root := normalizeTimelineRoot(part)
		if root == "" {
			continue
		}
		if slices.Contains(roots, root) {
			continue
		}
		roots = append(roots, root)
	}

	return roots
}

func applyMemoriesTimelineQuery(params url.Values, requestURL *url.URL, query MemoriesTimelineQuery) {
	if folder := strings.TrimSpace(query.Folder); folder != "" {
		params.Set("folder", normalizeTimelineRoot(folder))
	}
	if query.Recursive {
		params.Set("recursive", "1")
	}
	if query.Archive {
		params.Set("archive", "1")
	}
	if query.Hidden {
		params.Set("hidden", "1")
	}
	requestURL.RawQuery = params.Encode()
}

func normalizeTimelineRoot(raw string) string {
	root := strings.TrimSpace(raw)
	if root == "" {
		return ""
	}
	if !strings.HasPrefix(root, "/") {
		root = "/" + root
	}
	root = pathpkg.Clean(root)
	if root == "." {
		return ""
	}
	if !strings.HasPrefix(root, "/") {
		root = "/" + root
	}
	return root
}

func doJSON(req *http.Request, client *Client, target any) error {
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("request %s %s failed with status %d: %s", req.Method, req.URL.String(), resp.StatusCode, strings.TrimSpace(string(body)))
	}

	decoder := json.NewDecoder(resp.Body)
	if err := decoder.Decode(target); err != nil {
		return err
	}

	return nil
}
