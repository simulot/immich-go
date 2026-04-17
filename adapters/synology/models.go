package synology

import (
	"path"
	"strings"
	"time"
)

// LoginData represents the login response data
type LoginData struct {
	DID       string `json:"did"`       // Device ID
	SID       string `json:"sid"`       // Session ID
	SynoToken string `json:"synotoken"` // CSRF token for some APIs
	DeviceID  string `json:"device_id"` // Device ID (alternative field)
}

// LoginResponse represents the response from SYNO.API.Auth login
type LoginResponse struct {
	Data    LoginData `json:"data"`
	Success bool      `json:"success"`
	Error   *Error    `json:"error,omitempty"`
}

// APIResponse is the base response structure for most Synology APIs
type APIResponse[T any] struct {
	Data    T      `json:"data"`
	Success bool   `json:"success"`
	Error   *Error `json:"error,omitempty"`
}

// Error represents a Synology API error
type Error struct {
	Code int `json:"code"`
}

// Album represents a Synology Photos album
type Album struct {
	ID                   int         `json:"id"`
	Name                 string      `json:"name"`
	ItemCount            int         `json:"item_count"`
	OwnerUserID          int         `json:"owner_user_id"`
	CreateTime           int64       `json:"create_time"` // Unix timestamp
	EndTime              int64       `json:"end_time"`
	StartTime            int64       `json:"start_time"`
	Passphrase           string      `json:"passphrase"`
	Shared               bool        `json:"shared"`
	TemporaryShared      bool        `json:"temporary_shared"`
	FreezeAlbum          bool        `json:"freeze_album"`
	SortBy               string      `json:"sort_by"`
	SortDirection        string      `json:"sort_direction"`
	Type                 string      `json:"type"` // "normal" or "condition"
	Version              int         `json:"version"`
	Condition            Condition   `json:"condition,omitempty"`
	CantMigrateCondition interface{} `json:"cant_migrate_condition,omitempty"`
}

// Condition represents album filter conditions for conditional albums
type Condition struct {
	FolderFilter []int `json:"folder_filter,omitempty"`
	UserID       int   `json:"user_id,omitempty"`
}

// AlbumListResponse represents the response from Browse.Album list method
type AlbumListResponse struct {
	List []Album `json:"list"`
}

// Item represents a photo or video in Synology Photos
type Item struct {
	ID          int        `json:"id"`
	Filename    string     `json:"filename"`
	Filesize    int64      `json:"filesize"`
	FolderID    int        `json:"folder_id"`
	Time        int64      `json:"time"` // Local timestamp of capture time
	IndexedTime int64      `json:"indexed_time"`
	Type        string     `json:"type"`                // "photo", "video", or "live"
	LiveType    string     `json:"live_type,omitempty"` // "photo" for live photos
	OwnerUserID int        `json:"owner_user_id"`
	Additional  Additional `json:"additional,omitempty"`
}

// Additional contains extra metadata for items
type Additional struct {
	Thumbnail           ThumbnailInfo  `json:"thumbnail,omitempty"`
	Resolution          ResolutionInfo `json:"resolution,omitempty"`
	Orientation         int            `json:"orientation,omitempty"`
	OrientationOriginal int            `json:"orientation_original,omitempty"`
	Exif                ExifInfo       `json:"exif,omitempty"`
	Tag                 []TagInfo      `json:"tag,omitempty"`
	Description         string         `json:"description,omitempty"`
	GPS                 GPSInfo        `json:"gps,omitempty"`
	GeocodingID         int            `json:"geocoding_id,omitempty"`
	Address             AddressInfo    `json:"address,omitempty"`
	Person              []PersonInfo   `json:"person,omitempty"`
	VideoConvert        interface{}    `json:"video_convert,omitempty"`
	VideoMeta           interface{}    `json:"video_meta,omitempty"`
	ProviderUserID      int            `json:"provider_user_id,omitempty"`
}

// ThumbnailInfo contains thumbnail availability information
type ThumbnailInfo struct {
	CacheKey string `json:"cache_key"`
	UnitID   int    `json:"unit_id"`
	SM       string `json:"sm"`      // "ready" or "broken"
	M        string `json:"m"`       // "ready" or "broken"
	XL       string `json:"xl"`      // "ready" or "broken"
	Preview  string `json:"preview"` // "ready" or "broken"
}

// ResolutionInfo contains image/video resolution
type ResolutionInfo struct {
	Height int `json:"height"`
	Width  int `json:"width"`
}

// ExifInfo contains EXIF metadata
type ExifInfo struct {
	Aperture     string `json:"aperture,omitempty"`
	Camera       string `json:"camera,omitempty"`
	ExposureTime string `json:"exposure_time,omitempty"`
	FocalLength  string `json:"focal_length,omitempty"`
	ISO          int    `json:"iso,omitempty"`
	Lens         string `json:"lens,omitempty"`
}

// TagInfo represents a tag assigned to an item
type TagInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// GPSInfo contains GPS coordinates
type GPSInfo struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// AddressInfo contains reverse geocoding address
type AddressInfo struct {
	City       string `json:"city,omitempty"`
	CityID     int    `json:"city_id,omitempty"`
	Country    string `json:"country,omitempty"`
	CountryID  int    `json:"country_id,omitempty"`
	County     string `json:"county,omitempty"`
	CountyID   int    `json:"county_id,omitempty"`
	District   string `json:"district,omitempty"`
	DistrictID int    `json:"district_id,omitempty"`
	Landmark   string `json:"landmark,omitempty"`
	LandmarkID int    `json:"landmark_id,omitempty"`
	Route      string `json:"route,omitempty"`
	RouteID    int    `json:"route_id,omitempty"`
	State      string `json:"state,omitempty"`
	StateID    int    `json:"state_id,omitempty"`
	Town       string `json:"town,omitempty"`
	TownID     int    `json:"town_id,omitempty"`
	Village    string `json:"village,omitempty"`
	VillageID  int    `json:"village_id,omitempty"`
}

// PersonInfo represents a recognized person in Synology Photos
type PersonInfo struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

// ItemListResponse represents the response from Browse.Item list method
type ItemListResponse struct {
	List []Item `json:"list"`
}

// Tag represents a general tag in Synology Photos
type Tag struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	ItemCount int    `json:"item_count,omitempty"`
}

// TagListResponse represents the response from Browse.GeneralTag list method
type TagListResponse struct {
	List []Tag `json:"list"`
}

// Person represents a person in the face recognition system
type Person struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	ItemCount   int    `json:"item_count,omitempty"`
	CoverItemID int    `json:"cover_item_id,omitempty"`
}

// PersonListResponse represents the response from Browse.Person list method
type PersonListResponse struct {
	List []Person `json:"list"`
}

// Folder represents a folder in the Personal Space
type Folder struct {
	ID            int    `json:"id"`
	Name          string `json:"name"`
	Parent        int    `json:"parent"`
	OwnerUserID   int    `json:"owner_user_id"`
	Passphrase    string `json:"passphrase"`
	Shared        bool   `json:"shared"`
	SortBy        string `json:"sort_by"`
	SortDirection string `json:"sort_direction"`
}

// FolderListResponse represents the response from Browse.Folder list method
type FolderListResponse struct {
	List []Folder `json:"list"`
}

// IndexedTime returns the indexing time as a time.Time
func (i Item) IndexedAt() time.Time {
	// Indexed time is in milliseconds
	return time.Unix(0, i.IndexedTime*int64(time.Millisecond))
}

// IsPhoto returns true if the item is a photo
func (i Item) IsPhoto() bool {
	return i.Type == "photo"
}

// IsVideo returns true if the item is a video
func (i Item) IsVideo() bool {
	return i.Type == "video"
}

// IsLivePhoto returns true if the item is a live photo
func (i Item) IsLivePhoto() bool {
	return i.Type == "live"
}

// LivePhotoVideoFilename returns the inferred video filename for a live photo
// Live photos have two files: image (e.g., .HEIC) and video (e.g., .MOV)
// with the same basename
func (i Item) LivePhotoVideoFilename() string {
	if !i.IsLivePhoto() {
		return ""
	}
	ext := strings.ToLower(path.Ext(i.Filename))
	basename := i.Filename[:len(i.Filename)-len(ext)]

	// Map of image extensions to video extensions (common for live photos)
	videoExts := map[string]string{
		".heic": ".mov",
		".jpg":  ".mov",
		".jpeg": ".mov",
		".png":  ".mov",
	}

	if videoExt, ok := videoExts[ext]; ok {
		return basename + videoExt
	}

	// Default to .mov if extension not recognized
	return basename + ".mov"
}

// CreateTime returns the album creation time as time.Time
func (a Album) CreateAt() time.Time {
	return time.Unix(a.CreateTime, 0)
}
