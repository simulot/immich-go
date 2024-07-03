package immich

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
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
	roundTripper        *http.Transport
	endPoint            string        // Server API url
	key                 string        // User KEY
	DeviceUUID          string        // Device
	Retries             int           // Number of attempts on 500 errors
	RetriesDelay        time.Duration // Duration between retries
	apiTraceWriter      io.Writer
	supportedMediaTypes SupportedMedia // Server's list of supported medias
}

func (ic *ImmichClient) SetEndPoint(endPoint string) {
	ic.endPoint = endPoint
}

func (ic *ImmichClient) SetDeviceUUID(deviceUUID string) {
	ic.DeviceUUID = deviceUUID
}

func (ic *ImmichClient) EnableAppTrace(w io.Writer) {
	ic.apiTraceWriter = w
}

func (ic *ImmichClient) SupportedMedia() SupportedMedia {
	return ic.supportedMediaTypes
}

type clientOption func(ic *ImmichClient) error

func OptionVerifySSL(verify bool) clientOption {
	return func(ic *ImmichClient) error {
		ic.roundTripper.TLSClientConfig.InsecureSkipVerify = verify
		return nil
	}
}

func OptionConnectionTimeout(d time.Duration) clientOption {
	return func(ic *ImmichClient) error {
		ic.client.Timeout = d
		return nil
	}
}

// Create a new ImmichClient
func NewImmichClient(endPoint string, key string, options ...clientOption) (*ImmichClient, error) {
	var err error
	deviceUUID, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	// Create a custom HTTP client with SSL verification disabled
	// Add timeouts for #219
	// Info at https://www.loginradius.com/blog/engineering/tune-the-go-http-client-for-high-performance/
	// https://blog.cloudflare.com/the-complete-guide-to-golang-net-http-timeouts/
	// ![image](https://blog.cloudflare.com/content/images/2016/06/Timeouts-002.png)

	ic := ImmichClient{
		endPoint: endPoint + "/api",
		roundTripper: &http.Transport{
			MaxIdleConns:        100,
			IdleConnTimeout:     90 * time.Second,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
			MaxIdleConnsPerHost: 100,
			MaxConnsPerHost:     100,
		},
		key:          key,
		DeviceUUID:   deviceUUID,
		Retries:      1,
		RetriesDelay: time.Second * 1,
	}

	ic.client = &http.Client{
		Timeout:   time.Second * 60,
		Transport: ic.roundTripper,
	}

	for _, fn := range options {
		err := fn(&ic)
		if err != nil {
			return nil, err
		}
	}

	return &ic, nil
}

// Ping server
func (ic *ImmichClient) PingServer(ctx context.Context) error {
	r := PingResponse{}
	err := ic.newServerCall(ctx, "PingServer").do(get("/server-info/ping", setAcceptJSON()), responseJSON(&r))
	if err != nil {
		return fmt.Errorf("the ping API end point doesn't respond at this address: %s", ic.endPoint+"/server-info/ping")
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
		do(get("/users/me", setAcceptJSON()), responseJSON(&user))
	if err != nil {
		return user, err
	}

	sm, err := ic.GetSupportedMediaTypes(ctx)
	if err != nil {
		return user, err
	}
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

// getAssetStatistics
// Get user's stats

type UserStatistics struct {
	Images int `json:"images"`
	Videos int `json:"videos"`
	Total  int `json:"total"`
}

func (ic *ImmichClient) GetAssetStatistics(ctx context.Context) (UserStatistics, error) {
	var s UserStatistics
	err := ic.newServerCall(ctx, "GetAssetStatistics").do(get("/assets/statistics", setAcceptJSON()), responseJSON(&s))
	return s, err
}

type SupportedMedia map[string]string

const (
	TypeVideo   = "video"
	TypeImage   = "image"
	TypeSidecar = "sidecar"
	TypeUnknown = ""
)

var DefaultSupportedMedia = SupportedMedia{
	".3gp": TypeVideo, ".avi": TypeVideo, ".flv": TypeVideo, ".insv": TypeVideo, ".m2ts": TypeVideo, ".m4v": TypeVideo, ".mkv": TypeVideo, ".mov": TypeVideo, ".mp4": TypeVideo, ".mpg": TypeVideo, ".mts": TypeVideo, ".webm": TypeVideo, ".wmv": TypeVideo,
	".3fr": TypeImage, ".ari": TypeImage, ".arw": TypeImage, ".avif": TypeImage, ".bmp": TypeImage, ".cap": TypeImage, ".cin": TypeImage, ".cr2": TypeImage, ".cr3": TypeImage, ".crw": TypeImage, ".dcr": TypeImage, ".dng": TypeImage, ".erf": TypeImage,
	".fff": TypeImage, ".gif": TypeImage, ".heic": TypeImage, ".heif": TypeImage, ".hif": TypeImage, ".iiq": TypeImage, ".insp": TypeImage, ".jpe": TypeImage, ".jpeg": TypeImage, ".jpg": TypeImage,
	".jxl": TypeImage, ".k25": TypeImage, ".kdc": TypeImage, ".mrw": TypeImage, ".nef": TypeImage, ".orf": TypeImage, ".ori": TypeImage, ".pef": TypeImage, ".png": TypeImage, ".psd": TypeImage, ".raf": TypeImage, ".raw": TypeImage, ".rw2": TypeImage,
	".rwl": TypeImage, ".sr2": TypeImage, ".srf": TypeImage, ".srw": TypeImage, ".tif": TypeImage, ".tiff": TypeImage, ".webp": TypeImage, ".x3f": TypeImage,
	".xmp": TypeSidecar,
	".mp":  TypeVideo,
}

func (ic *ImmichClient) GetSupportedMediaTypes(ctx context.Context) (SupportedMedia, error) {
	var s map[string][]string

	err := ic.newServerCall(ctx, "GetSupportedMediaTypes").do(get("/server-info/media-types", setAcceptJSON()), responseJSON(&s))
	if err != nil {
		return nil, err
	}
	sm := make(SupportedMedia)
	for t, l := range s {
		for _, e := range l {
			sm[e] = t
		}
	}
	sm[".mp"] = TypeVideo
	return sm, err
}

func (sm SupportedMedia) TypeFromExt(ext string) string {
	ext = strings.ToLower(ext)
	return sm[ext]
}

func (sm SupportedMedia) IsMedia(ext string) bool {
	t := sm.TypeFromExt(ext)
	return t == TypeVideo || t == TypeImage
}

func (sm SupportedMedia) IsExtensionPrefix(ext string) bool {
	ext = strings.ToLower(ext)
	for e, t := range sm {
		if t == TypeVideo || t == TypeImage {
			if ext == e[:len(e)-1] {
				return true
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
