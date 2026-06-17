package nextcloud

import (
	"context"
	"crypto/tls"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/studio-b12/gowebdav"
)

// Config defines how to connect to a Nextcloud instance.
type Config struct {
	BaseURL       string
	Username      string
	Password      string
	SkipVerifySSL bool
	Timeout       time.Duration
}

// Client combines a DAV client with a standard HTTP client for Nextcloud APIs
// that are outside the DAV surface, such as OCS and Memories endpoints.
type Client struct {
	baseURL    *url.URL
	davRootURL *url.URL
	username   string
	password   string
	httpClient *http.Client
	davClient  *gowebdav.Client
}

// NewClient creates a Nextcloud client that uses gowebdav for DAV operations and
// a shared net/http client for non-DAV endpoints.
func NewClient(cfg Config) (*Client, error) {
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}
	baseURL, err := normalizeBaseURL(cfg.BaseURL)
	if err != nil {
		return nil, err
	}

	transport := cloneDefaultTransport()
	transport.TLSClientConfig = &tls.Config{ //nolint:gosec // This is a user-selected source-side option for self-hosted servers.
		InsecureSkipVerify: cfg.SkipVerifySSL,
	}

	httpClient := &http.Client{
		Timeout:   cfg.Timeout,
		Transport: transport,
	}
	davRootURL := baseURL.ResolveReference(&url.URL{Path: joinURLPath(baseURL.Path, "remote.php/dav")})
	davClient := gowebdav.NewClient(davRootURL.String(), strings.TrimSpace(cfg.Username), strings.TrimSpace(cfg.Password))
	davClient.SetTransport(transport)
	// Keep the timeout on the plain HTTP client for OCS and Memories API calls,
	// but do not apply a whole-request timeout to DAV operations. Large imports
	// stream file bodies and PROPFIND directory listings from Nextcloud; using the
	// same total deadline there causes long reads to fail mid-transfer.

	return &Client{
		baseURL:    baseURL,
		davRootURL: davRootURL,
		username:   strings.TrimSpace(cfg.Username),
		password:   strings.TrimSpace(cfg.Password),
		httpClient: httpClient,
		davClient:  davClient,
	}, nil
}

// BaseURL returns the normalized Nextcloud base URL.
func (c *Client) BaseURL() string {
	return c.baseURL.String()
}

// DAVRoot returns the normalized DAV root URL used by the underlying DAV client.
func (c *Client) DAVRoot() string {
	return c.davRootURL.String()
}

// HTTPClient returns the shared HTTP client used for non-DAV endpoints.
func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
}

// DAV returns the underlying DAV client.
func (c *Client) DAV() *gowebdav.Client {
	return c.davClient
}

// Connect validates the DAV connection.
func (c *Client) Connect(ctx context.Context) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return c.davClient.Connect()
}

// NewRequest creates an authenticated request relative to the configured base URL.
func (c *Client) NewRequest(ctx context.Context, method string, relativePath string, body io.Reader) (*http.Request, error) {
	requestURL, err := appendURL(c.baseURL, relativePath)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, method, requestURL.String(), body)
	if err != nil {
		return nil, err
	}
	req.SetBasicAuth(c.username, c.password)
	return req, nil
}

// NewOCSRequest creates an authenticated request for the OCS v2 API.
func (c *Client) NewOCSRequest(ctx context.Context, method string, relativePath string, body io.Reader) (*http.Request, error) {
	req, err := c.NewRequest(ctx, method, "ocs/v2.php/"+strings.TrimPrefix(relativePath, "/"), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("OCS-APIRequest", "true")
	req.Header.Set("Accept", "application/json")
	return req, nil
}

// NewMemoriesRequest creates an authenticated request for the Memories app routes.
// It uses the explicit index.php entrypoint so it does not rely on rewrite rules.
// The OCS-APIRequest header is added as well because Nextcloud accepts it as a
// CSRF bypass signal for token-authenticated API clients.
func (c *Client) NewMemoriesRequest(ctx context.Context, method string, relativePath string, body io.Reader) (*http.Request, error) {
	req, err := c.NewRequest(ctx, method, "index.php/apps/memories/"+strings.TrimPrefix(relativePath, "/"), body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("OCS-APIRequest", "true")
	req.Header.Set("Accept", "application/json")
	return req, nil
}

// Do executes a non-DAV request with the configured shared HTTP client.
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.httpClient.Do(req)
}

func validateConfig(cfg Config) error {
	var joinedErr error

	if strings.TrimSpace(cfg.BaseURL) == "" {
		joinedErr = errors.Join(joinedErr, errors.New("missing Nextcloud base URL"))
	}
	if strings.TrimSpace(cfg.Username) == "" {
		joinedErr = errors.Join(joinedErr, errors.New("missing Nextcloud username"))
	}
	if strings.TrimSpace(cfg.Password) == "" {
		joinedErr = errors.Join(joinedErr, errors.New("missing Nextcloud password or app password"))
	}
	if cfg.Timeout <= 0 {
		joinedErr = errors.Join(joinedErr, errors.New("invalid Nextcloud timeout: must be greater than 0"))
	}

	return joinedErr
}

func normalizeBaseURL(rawURL string) (*url.URL, error) {
	rawURL = strings.TrimSpace(rawURL)
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, err
	}
	if parsedURL.Scheme == "" || parsedURL.Host == "" {
		return nil, errors.New("nextcloud base URL must include scheme and host")
	}
	if parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
		return nil, errors.New("nextcloud base URL must not include a query string or fragment")
	}
	parsedURL.Path = strings.TrimRight(parsedURL.Path, "/")
	for _, suffix := range []string{"/remote.php/dav", "/remote.php/webdav"} {
		if strings.HasSuffix(parsedURL.Path, suffix) {
			parsedURL.Path = strings.TrimSuffix(parsedURL.Path, suffix)
			break
		}
	}
	parsedURL.Path = strings.TrimRight(parsedURL.Path, "/")
	parsedURL.RawPath = ""
	return parsedURL, nil
}

func cloneDefaultTransport() *http.Transport {
	if transport, ok := http.DefaultTransport.(*http.Transport); ok {
		return transport.Clone()
	}
	return &http.Transport{}
}

func joinURLPath(basePath string, relativePath string) string {
	basePath = strings.TrimRight(basePath, "/")
	relativePath = strings.TrimSpace(relativePath)
	relativePath = strings.TrimLeft(relativePath, "/")

	switch {
	case basePath == "" && relativePath == "":
		return "/"
	case basePath == "":
		return "/" + relativePath
	case relativePath == "":
		return basePath
	default:
		return basePath + "/" + relativePath
	}
}

func appendURL(baseURL *url.URL, relativePath string) (*url.URL, error) {
	relativePath = strings.TrimSpace(relativePath)
	if relativePath == "" {
		resolvedURL := *baseURL
		return &resolvedURL, nil
	}

	relativeURL, err := url.Parse(strings.TrimPrefix(relativePath, "/"))
	if err != nil {
		return nil, err
	}

	resolvedURL := *baseURL
	if resolvedURL.Path == "" {
		resolvedURL.Path = "/"
	} else if !strings.HasSuffix(resolvedURL.Path, "/") {
		resolvedURL.Path += "/"
	}

	return resolvedURL.ResolveReference(relativeURL), nil
}
