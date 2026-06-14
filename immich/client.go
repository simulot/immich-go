package immich

import (
	"context"
	"crypto/tls"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/simulot/immich-go/internal/filetypes"
)

/*
ImmichClient is a proxy for immich services

Immich API documentation https://documentation.immich.app/docs/api/introduction
*/

type ImmichClient struct {
	client         *http.Client
	transport      *http.Transport
	endPoint       string        // Server API url
	key            string        // User KEY
	DeviceUUID     string        // Device
	RetryAttempts  int           // Number of attempts for transient failures
	RetryBackoff   time.Duration // Initial duration between retries
	RetryMaxDelay  time.Duration // Maximum duration between retries
	apiTraceWriter io.Writer     // If not nil, logs API calls to this writer
	retryLogger    func(context.Context, string, ...any)
	randSource     *rand.Rand

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

func (ic *ImmichClient) SupportedMedia() filetypes.SupportedMedia {
	return ic.supportedMediaTypes
}

func (ic *ImmichClient) EnableAppTrace(rtd RoundTripperDecorator) {
	if rtd != nil {
		ic.client.Transport = rtd(ic.client.Transport)
	} else {
		ic.client.Transport = ic.transport
	}
}

type clientOption func(ic *ImmichClient) error

func OptionSetAPITrace(rtd RoundTripperDecorator) clientOption {
	return func(ic *ImmichClient) error {
		ic.EnableAppTrace(rtd)
		return nil
	}
}

func OptionVerifySSL(verify bool) clientOption {
	return func(ic *ImmichClient) error {
		ic.transport.TLSClientConfig.InsecureSkipVerify = verify
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

func OptionRetryLogger(fn func(context.Context, string, ...any)) clientOption {
	return func(ic *ImmichClient) error {
		ic.retryLogger = fn
		return nil
	}
}

func OptionRetryPolicy(attempts int, backoff, maxDelay time.Duration) clientOption {
	return func(ic *ImmichClient) error {
		if attempts > 0 {
			ic.RetryAttempts = attempts
		}
		if backoff > 0 {
			ic.RetryBackoff = backoff
		}
		if maxDelay > 0 {
			ic.RetryMaxDelay = maxDelay
		}
		if ic.RetryMaxDelay < ic.RetryBackoff {
			ic.RetryMaxDelay = ic.RetryBackoff
		}
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
		transport: &http.Transport{
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
		RetryAttempts: 6,
		RetryBackoff:  time.Second,
		RetryMaxDelay: 30 * time.Second,
		randSource:    rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	ic.client = &http.Client{
		Timeout:   time.Minute * 20,
		Transport: ic.transport,
	}

	for _, fn := range options {
		err := fn(&ic)
		if err != nil {
			return nil, err
		}
	}

	return &ic, nil
}

func (ic *ImmichClient) retryDelay(attempt int) time.Duration {
	if ic == nil {
		return 0
	}
	base := ic.RetryBackoff
	if base <= 0 {
		base = time.Second
	}
	maxDelay := ic.RetryMaxDelay
	if maxDelay <= 0 {
		maxDelay = 30 * time.Second
	}
	delay := base
	for i := 1; i < attempt; i++ {
		if delay >= maxDelay {
			delay = maxDelay
			break
		}
		delay *= 2
		if delay > maxDelay {
			delay = maxDelay
			break
		}
	}
	if delay <= 0 {
		return 0
	}
	if ic.randSource == nil {
		return delay
	}
	jitter := time.Duration(ic.randSource.Int63n(int64(delay / 2 + 1)))
	return delay + jitter
}
