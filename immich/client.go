package immich

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"
)

/*
ImmichClient is a proxy for immich services

Immich API documentation https://documentation.immich.app/docs/api/introduction
*/

type ImmichClient struct {
	client       *http.Client
	endPoint     string        // Server API url
	key          string        // User KEY
	DeviceUUID   string        // Device
	Retries      int           // Number of attempts on 500 errors
	RetriesDelay time.Duration // Duration between retries
	ApiTrace     bool
}

func (ic *ImmichClient) SetEndPoint(endPoint string) *ImmichClient {
	ic.endPoint = endPoint
	return ic
}

func (ic *ImmichClient) SetDeviceUUID(DeviceUUID string) *ImmichClient {
	ic.DeviceUUID = DeviceUUID
	return ic
}

func (ic *ImmichClient) EnableAppTrace(state bool) *ImmichClient {
	ic.ApiTrace = state
	return ic
}

// Create a new ImmichClient
func NewImmichClient(endPoint string, key string, sslVerify bool) (*ImmichClient, error) {
	var err error
	DeviceUUID, err := os.Hostname()
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
		DeviceUUID:   DeviceUUID,
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
