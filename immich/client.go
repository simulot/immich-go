package immich

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

/*
ImmichClient is a proxy for immich services

Immich API documentation https://documentation.immich.app/docs/api/introduction
*/

type ImmichClient struct {
	client              *http.Client
	endPoint            string        // Server API url
	key                 string        // User KEY
	DeviceUUID          string        // Device
	Retries             int           // Number of attempts on 500 errors
	RetriesDelay        time.Duration // Duration between retries
	APITrace            bool
	supportedMediaTypes SupportedMedia // Server's list of supported medias
}

func (ic *ImmichClient) SetEndPoint(endPoint string) {
	ic.endPoint = endPoint
}

func (ic *ImmichClient) SetDeviceUUID(deviceUUID string) {
	ic.DeviceUUID = deviceUUID
}

func (ic *ImmichClient) EnableAppTrace(state bool) {
	ic.APITrace = state
}

func (ic *ImmichClient) SupportedMedia() SupportedMedia {
	return ic.supportedMediaTypes
}

// Create a new ImmichClient
func NewImmichClient(endPoint string, key string, sslVerify bool) (*ImmichClient, error) {
	var err error
	deviceUUID, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	// Create a custom HTTP client with SSL verification disabled
	transportOptions := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: sslVerify},
	}
	tlsClient := &http.Client{Transport: transportOptions}

	ic := ImmichClient{
		endPoint:     endPoint + "/api",
		key:          key,
		client:       tlsClient,
		DeviceUUID:   deviceUUID,
		Retries:      1,
		RetriesDelay: time.Second * 1,
	}

	return &ic, nil
}

// Ping server
func (ic *ImmichClient) PingServer(ctx context.Context) error {
	r := PingResponse{}
	err := ic.newServerCall(ctx, "PingServer").do(get("/server-info/ping", setAcceptJSON()), responseJSON(&r))
	if err != nil {
		return err
	}
	if r.Res != "pong" {
		return fmt.Errorf("incorrect ping response: %s", r.Res)
	}
	return nil
}

// ValidateConnection
// Validate the connection by querying the identity of the user having the given key

func (ic *ImmichClient) ValidateConnection(ctx context.Context) (User, error) {
	var user User
	err := ic.newServerCall(ctx, "ValidateConnection").
		do(get("/user/me", setAcceptJSON()), responseJSON(&user))
	if err != nil {
		return user, err
	}
	sm, err := ic.GetSupportedMediaTypes(ctx)
	if err != nil {
		return user, err
	}
	sm["ignored-files"] = []string{".html", ".mp"}
	ic.supportedMediaTypes = sm
	return user, nil
}

type ServerStatistics struct {
	Photos      int   `json:"photos"`
	Videos      int   `json:"videos"`
	Usage       int64 `json:"usage"`
	UsageByUser []struct {
		UserID           string `json:"userId"`
		UserName         string `json:"userName"`
		Photos           int    `json:"photos"`
		Videos           int    `json:"videos"`
		Usage            int64  `json:"usage"`
		QuotaSizeInBytes any    `json:"quotaSizeInBytes"`
	} `json:"usageByUser"`
}

// getServerStatistics
// Get server stats

func (ic *ImmichClient) GetServerStatistics(ctx context.Context) (ServerStatistics, error) {
	var s ServerStatistics

	err := ic.newServerCall(ctx, "GetServerStatistics").do(get("/server-info/statistics", setAcceptJSON()), responseJSON(&s))
	return s, err
}

type SupportedMedia map[string][]string

const (
	TypeVideo   = "video"
	TypeImage   = "image"
	TypeSidecar = "sidecar"
	TypeIgnored = "ignored"
	TypeUnknown = ""
)

var DefaultSupportedMedia = SupportedMedia{
	TypeVideo: []string{
		".3gp",
		".avi",
		".flv",
		".insv",
		".m2ts",
		".m4v",
		".mkv",
		".mov",
		".mp4",
		".mpg",
		".mts",
		".webm",
		".wmv",
	},
	TypeImage: []string{
		".3fr",
		".ari",
		".arw",
		".avif",
		".bmp",
		".cap",
		".cin",
		".cr2",
		".cr3",
		".crw",
		".dcr",
		".dng",
		".erf",
		".fff",
		".gif",
		".heic",
		".heif",
		".hif",
		".iiq",
		".insp",
		".jpe",
		".jpeg",
		".jpg",
		".jxl",
		".k25",
		".kdc",
		".mrw",
		".nef",
		".orf",
		".ori",
		".pef",
		".png",
		".psd",
		".raf",
		".raw",
		".rw2",
		".rwl",
		".sr2",
		".srf",
		".srw",
		".tif",
		".tiff",
		".webp",
		".x3f",
	},
	TypeSidecar: []string{
		".xmp",
	},
	TypeIgnored: []string{
		".mp",
		".html",
	},
}

func (ic *ImmichClient) GetSupportedMediaTypes(ctx context.Context) (SupportedMedia, error) {
	var s SupportedMedia

	err := ic.newServerCall(ctx, "GetSupportedMediaTypes").do(get("/server-info/media-types", setAcceptJSON()), responseJSON(&s))
	s[TypeIgnored] = []string{".mp", ".html"}
	return s, err
}

func (sm SupportedMedia) TypeFromExt(ext string) string {
	ext = strings.ToLower(ext)
	for t, l := range sm {
		for _, e := range l {
			if e == ext {
				return t
			}
		}
	}
	return ""
}

func (sm SupportedMedia) IsMedia(ext string) bool {
	t := sm.TypeFromExt(ext)
	return t == TypeVideo || t == TypeImage
}

func (sm SupportedMedia) IsExtensionPrefix(ext string) bool {
	ext = strings.ToLower(ext)
	for t, l := range sm {
		if t == TypeVideo || t == TypeImage {
			for _, e := range l {
				if ext == e[:len(e)-1] {
					return true
				}
			}
		}
	}
	return false
}

func (sm SupportedMedia) IsIgnoredExt(ext string) bool {
	t := sm.TypeFromExt(ext)
	return t == ""
}

func (ic *ImmichClient) TypeFromExt(ext string) string {
	return ic.supportedMediaTypes.TypeFromExt(ext)
}

func (ic *ImmichClient) IsExtensionPrefix(ext string) bool {
	return ic.supportedMediaTypes.IsExtensionPrefix(ext)
}

func (ic *ImmichClient) IsIgnoredExt(ext string) bool {
	return ic.supportedMediaTypes.IsIgnoredExt(ext)
}
