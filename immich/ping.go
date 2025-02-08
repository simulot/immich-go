package immich

import (
	"bytes"
	"context"
	"fmt"

	"github.com/simulot/immich-go/internal/filetypes"
)

type PingResponse struct {
	Res string `json:"res"`
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
	sm[".mp"] = filetypes.TypeUseless
	sm[".json"] = filetypes.TypeSidecar
	return sm, err
}
