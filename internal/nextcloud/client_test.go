package nextcloud

import (
	"context"
	"net/http"
	"reflect"
	"testing"
	"time"
	"unsafe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/studio-b12/gowebdav"
)

func TestNewClientNormalizesBaseURLAndDAVRoot(t *testing.T) {
	t.Parallel()

	client, err := NewClient(Config{
		BaseURL:  " https://cloud.example.com/nextcloud/remote.php/dav/ ",
		Username: " alice ",
		Password: " secret ",
		Timeout:  3 * time.Minute,
	})
	require.NoError(t, err)

	assert.Equal(t, "https://cloud.example.com/nextcloud", client.BaseURL())
	assert.Equal(t, "https://cloud.example.com/nextcloud/remote.php/dav", client.DAVRoot())
	assert.Equal(t, 3*time.Minute, client.HTTPClient().Timeout)
	require.NotNil(t, client.DAV())
	assert.Zero(t, davHTTPClient(t, client.DAV()).Timeout)
	transport, ok := client.HTTPClient().Transport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, transport.TLSClientConfig)
	assert.False(t, transport.TLSClientConfig.InsecureSkipVerify)
}

func TestNewClientRejectsInvalidConfig(t *testing.T) {
	t.Parallel()

	_, err := NewClient(Config{})
	require.Error(t, err)
	assert.ErrorContains(t, err, "missing Nextcloud base URL")
	assert.ErrorContains(t, err, "missing Nextcloud username")
	assert.ErrorContains(t, err, "missing Nextcloud password or app password")
	assert.ErrorContains(t, err, "greater than 0")
}

func TestNewClientRejectsInvalidBaseURL(t *testing.T) {
	t.Parallel()

	_, err := NewClient(Config{
		BaseURL:  "cloud.example.com",
		Username: "alice",
		Password: "secret",
		Timeout:  time.Minute,
	})
	require.Error(t, err)
	assert.ErrorContains(t, err, "must include scheme and host")
}

func TestNewClientRespectsSkipVerifySSL(t *testing.T) {
	t.Parallel()

	client, err := NewClient(Config{
		BaseURL:       "https://cloud.example.com",
		Username:      "alice",
		Password:      "secret",
		SkipVerifySSL: true,
		Timeout:       time.Minute,
	})
	require.NoError(t, err)

	transport, ok := client.HTTPClient().Transport.(*http.Transport)
	require.True(t, ok)
	require.NotNil(t, transport.TLSClientConfig)
	assert.True(t, transport.TLSClientConfig.InsecureSkipVerify)
}

func TestNewRequestBuildsAuthenticatedURL(t *testing.T) {
	t.Parallel()

	client, err := NewClient(Config{
		BaseURL:  "https://cloud.example.com/nextcloud",
		Username: "alice",
		Password: "secret",
		Timeout:  time.Minute,
	})
	require.NoError(t, err)

	req, err := client.NewRequest(context.Background(), http.MethodGet, "/status.php", nil)
	require.NoError(t, err)
	assert.Equal(t, "https://cloud.example.com/nextcloud/status.php", req.URL.String())
	username, password, ok := req.BasicAuth()
	require.True(t, ok)
	assert.Equal(t, "alice", username)
	assert.Equal(t, "secret", password)
}

func TestNewOCSRequestAddsHeadersAndPath(t *testing.T) {
	t.Parallel()

	client, err := NewClient(Config{
		BaseURL:  "https://cloud.example.com/nextcloud",
		Username: "alice",
		Password: "secret",
		Timeout:  time.Minute,
	})
	require.NoError(t, err)

	req, err := client.NewOCSRequest(context.Background(), http.MethodGet, "apps/files_sharing/api/v1/shares?path=%2FPhotos", nil)
	require.NoError(t, err)
	assert.Equal(t, "https://cloud.example.com/nextcloud/ocs/v2.php/apps/files_sharing/api/v1/shares?path=%2FPhotos", req.URL.String())
	assert.Equal(t, "true", req.Header.Get("OCS-APIRequest"))
	assert.Equal(t, "application/json", req.Header.Get("Accept"))
}

func TestNewMemoriesRequestUsesExplicitIndexPath(t *testing.T) {
	t.Parallel()

	client, err := NewClient(Config{
		BaseURL:  "https://cloud.example.com/nextcloud",
		Username: "alice",
		Password: "secret",
		Timeout:  time.Minute,
	})
	require.NoError(t, err)

	req, err := client.NewMemoriesRequest(context.Background(), http.MethodGet, "api/describe", nil)
	require.NoError(t, err)
	assert.Equal(t, "https://cloud.example.com/nextcloud/index.php/apps/memories/api/describe", req.URL.String())
	assert.Equal(t, "true", req.Header.Get("OCS-APIRequest"))
	assert.Equal(t, "application/json", req.Header.Get("Accept"))
}

func TestConnectHonorsCancelledContext(t *testing.T) {
	t.Parallel()

	client, err := NewClient(Config{
		BaseURL:  "https://cloud.example.com/nextcloud",
		Username: "alice",
		Password: "secret",
		Timeout:  time.Minute,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err = client.Connect(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func davHTTPClient(t *testing.T, client *gowebdav.Client) *http.Client {
	t.Helper()

	value := reflect.ValueOf(client).Elem().FieldByName("c")
	require.True(t, value.IsValid())
	require.True(t, value.CanAddr())

	unsafeValue := reflect.NewAt(value.Type(), unsafe.Pointer(value.UnsafeAddr())).Elem()
	httpClient, ok := unsafeValue.Interface().(*http.Client)
	require.True(t, ok)
	return httpClient
}
