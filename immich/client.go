package immich

import (
	"bytes"
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/simulot/immich-go/internal/filetypes"
	filetype "github.com/simulot/immich-go/internal/filetypes"
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
	supportedMediaTypes filetype.SupportedMedia // Server's list of supported medias
	dryRun              bool                    //  If true, do not send any data to the server
}

func (ic *ImmichClient) SetEndPoint(endPoint string) {
	ic.endPoint = endPoint
}

func (ic *ImmichClient) GetEndPoint() string {
	return ic.endPoint
}

func (ic *ImmichClient) SetDeviceUUID(deviceUUID string) {
	ic.DeviceUUID = deviceUUID
}

func (ic *ImmichClient) EnableAppTrace(w io.Writer) {
	ic.apiTraceWriter = w
}

func (ic *ImmichClient) SupportedMedia() filetypes.SupportedMedia {
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

func OptionDryRun(dryRun bool) clientOption {
	return func(ic *ImmichClient) error {
		ic.dryRun = dryRun
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
	b := bytes.NewBuffer(nil)
	err := ic.newServerCall(ctx, EndPointPingServer).do(getRequest("/server/ping", setAcceptJSON()), responseCopy(b), responseJSON(&r))
	if err != nil {
		return fmt.Errorf("unexpected response to the immich's ping API at this address: %s:\n%s", ic.endPoint+"/server/ping", b.String())
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

	err := ic.newServerCall(ctx, EndPointValidateConnection).
		do(getRequest("/users/me", setAcceptJSON()), responseJSON(&user))
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

	err := ic.newServerCall(ctx, EndPointGetServerStatistics).do(getRequest("/server/statistics", setAcceptJSON()), responseJSON(&s))
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
	err := ic.newServerCall(ctx, EndPointGetAssetStatistics).do(getRequest("/assets/statistics", setAcceptJSON()), responseJSON(&s))
	return s, err
}

func (ic *ImmichClient) GetSupportedMediaTypes(ctx context.Context) (filetypes.SupportedMedia, error) {
	var s map[string][]string

	err := ic.newServerCall(ctx, EndPointGetSupportedMediaTypes).do(getRequest("/server/media-types", setAcceptJSON()), responseJSON(&s))
	if err != nil {
		return nil, err
	}
	sm := make(filetypes.SupportedMedia)
	for t, l := range s {
		for _, e := range l {
			sm[e] = t
		}
	}
	sm[".mp"] = filetypes.TypeVideo
	return sm, err
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
