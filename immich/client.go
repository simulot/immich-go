package immich

import (
	"context"
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

func (ic *ImmichClient) SetDeviceUUID(deviceUUID string) *ImmichClient {
	ic.DeviceUUID = deviceUUID
	return ic
}

func (ic *ImmichClient) EnableAppTrace(state bool) *ImmichClient {
	ic.ApiTrace = state
	return ic
}

// Create a new ImmichClient
func NewImmichClient(endPoint string, key string) (*ImmichClient, error) {
	var err error
	deviceUUID, err := os.Hostname()
	if err != nil {
		return nil, err
	}

	ic := ImmichClient{
		endPoint:     endPoint + "/api",
		key:          key,
		client:       &http.Client{},
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
	return user, nil
}
