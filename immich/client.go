package immich

import (
	"fmt"
	"net/http"
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
}

// Create a new ImmichClient
func NewImmichClient(endPoint, key, deviceUUID string) (*ImmichClient, error) {
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
func (ic *ImmichClient) PingServer() error {
	r := PingResponse{}
	err := ic.newServerCall("PingServer").do(get("/server-info/ping", setAcceptJSON()), responseJSON(&r))
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

func (ic *ImmichClient) ValidateConnection() (User, error) {
	var user User
	err := ic.newServerCall("ValidateConnection").
		do(get("/user/me", setAcceptJSON()), responseJSON(&user))
	if err != nil {
		return user, err
	}
	return user, nil
}

// // Get all asset IDs belonging to the user
// func (ic *ImmichClient) GetUserAssetsByDeviceId(deviceID string) (*StringList, error) {
// 	list := StringList{}
// 	err := ic.newServerCall("GetUserAssetsByDeviceId").
// 		do(get("/asset/"+ic.DeviceUUID, setAcceptJSON()), responseJSON(&list))
// 	if err != nil {
// 		return &list, err
// 	}
// 	return &list, nil
// }
