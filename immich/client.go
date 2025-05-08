package immich

import (
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/simulot/immich-go/internal/filetypes"
)

/*
ImmichClient is a proxy for immich services

Immich API documentation https://documentation.immich.app/docs/api/introduction
*/

type ImmichClient struct {
	client         *http.Client
	roundTripper   *http.Transport
	endPoint       string        // Server API url
	key            string        // User KEY
	DeviceUUID     string        // Device
	Retries        int           // Number of attempts on 500 errors
	RetriesDelay   time.Duration // Duration between retries
	apiTraceWriter io.Writer     // If not nil, logs API calls to this writer
	apiTraceLock   sync.Mutex    // Lock for API trace

	supportedMediaTypes filetypes.SupportedMedia // Server's list of supported medias
	dryRun              bool                     //  If true, do not send any data to the server
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
		ic.client.Transport.(*http.Transport).ResponseHeaderTimeout = d
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
			Dial: (&net.Dialer{
				Timeout:   30 * time.Second,
				KeepAlive: 30 * time.Second,
			}).Dial,
			TLSHandshakeTimeout:   30 * time.Second,
			ResponseHeaderTimeout: 20 * time.Minute,
		},
		key:          key,
		DeviceUUID:   deviceUUID,
		Retries:      1,
		RetriesDelay: time.Second * 1,
	}

	ic.client = &http.Client{
		Timeout:   time.Minute * 20,
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
