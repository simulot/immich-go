package immich

import (
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
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
	Retries        int           // Number of attempts on 500 errors
	RetriesDelay   time.Duration // Duration between retries
	apiTraceWriter io.Writer     // If not nil, logs API calls to this writer

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

// OptionClientCertificate configures a client certificate and key for mutual TLS
// (mTLS) authentication. The server will receive this certificate during the TLS
// handshake. Both certFile and keyFile must point to PEM-encoded files. When both
// are empty the option is a no-op.
func OptionClientCertificate(certFile, keyFile string) clientOption {
	return func(ic *ImmichClient) error {
		if certFile == "" && keyFile == "" {
			return nil
		}
		if certFile == "" || keyFile == "" {
			return errors.New("mTLS requires both --client-cert and --client-key to be set")
		}
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			return fmt.Errorf("loading client certificate: %w", err)
		}
		ic.transport.TLSClientConfig.Certificates = append(ic.transport.TLSClientConfig.Certificates, cert)
		return nil
	}
}

// OptionCACertificate adds a PEM-encoded certificate authority bundle used to
// verify the server's certificate. This is typically needed when the Immich
// server presents a certificate signed by a private CA, as is common in mTLS
// deployments. When caFile is empty the option is a no-op.
func OptionCACertificate(caFile string) clientOption {
	return func(ic *ImmichClient) error {
		if caFile == "" {
			return nil
		}
		caCert, err := os.ReadFile(caFile)
		if err != nil {
			return fmt.Errorf("reading CA certificate: %w", err)
		}
		pool, err := x509.SystemCertPool()
		if err != nil || pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM(caCert) {
			return fmt.Errorf("no valid certificate found in CA file %q", caFile)
		}
		ic.transport.TLSClientConfig.RootCAs = pool
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
		Retries:      1,
		RetriesDelay: time.Second * 1,
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
