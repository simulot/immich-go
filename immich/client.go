package immich

import (
	"fmt"
	"net/http"
	"time"
)

/*
	Immich API documentation https://documentation.immich.app/docs/api/introduction

	ImmichClient is a proxy for immich services

*/

type ImmichClient struct {
	client       *http.Client
	endPoint     string        // Server API url
	key          string        // User KEY
	DeviceUUID   string        // Device
	Retries      int           // Number of attempts on 500 errors
	RetriesDelay time.Duration // Duration between retries
}

// Create a new ImmichClient
func NewImmichClient(endPoint, key, deviceUUID string) (*ImmichClient, error) {
	ic := ImmichClient{
		endPoint:     endPoint + "/api",
		key:          key,
		client:       &http.Client{},
		DeviceUUID:   deviceUUID,
		Retries:      3,
		RetriesDelay: time.Second * 1,
	}
	return &ic, nil
}

// Ping server
func (ic *ImmichClient) PingServer() error {
	r := PingResponse{}
	sc := ic.newServerCall("PingServer").getRequest("/server-info/ping").callServer().decodeJSONResponse(&r)
	if sc.err != nil {
		return sc.Err()
	}
	if r.Res != "pong" {
		return fmt.Errorf("incorrect ping response: %s", r.Res)
	}
	return nil
}

// ValidateConnection
// Validate the connection by quering the identity of the user having the given key

func (ic *ImmichClient) ValidateConnection() (User, error) {
	var user User
	sc := ic.newServerCall("ValidateConnection").getRequest("/user/me").callServer().decodeJSONResponse(&user)
	if sc.err != nil {
		return user, sc.Err()
	}

	return user, nil
}

// Get all asset IDs belonging to the user
func (ic *ImmichClient) GetUserAssetsByDeviceId(deviceID string) (*StringList, error) {
	list := StringList{}
	sc := ic.newServerCall("GetUserAssetsByDeviceId").getRequest("/asset/" + ic.DeviceUUID).callServer().decodeJSONResponse(&list)
	if sc.err != nil {
		return &list, sc.Err()
	}

	return &list, nil
}
