package synology

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	defaultTimeout      = 30 * time.Second
	defaultMaxRetries   = 3
	defaultRetryDelay   = 1 * time.Second
	defaultLimit        = 100
)

// Client is a Synology Photos API client
type Client struct {
	baseURL    string
	account    string
	password   string
	sid        string
	did        string
	httpClient *http.Client
	maxRetries int
	retryDelay time.Duration
}

// ClientOption is a functional option for the Client
type ClientOption func(*Client)

// WithHTTPClient sets a custom HTTP client
func WithHTTPClient(httpClient *http.Client) ClientOption {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// WithInsecureSkipVerify skips SSL certificate verification
func WithInsecureSkipVerify(skip bool) ClientOption {
	return func(c *Client) {
		if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
			transport.TLSClientConfig.InsecureSkipVerify = skip
		}
	}
}

// WithTimeout sets the HTTP client timeout
func WithTimeout(timeout time.Duration) ClientOption {
	return func(c *Client) {
		c.httpClient.Timeout = timeout
	}
}

// WithRetries sets the maximum number of retries
func WithRetries(maxRetries int) ClientOption {
	return func(c *Client) {
		c.maxRetries = maxRetries
	}
}

// NewClient creates a new Synology Photos API client
func NewClient(baseURL, account, password string, opts ...ClientOption) (*Client, error) {
	// Clean up the base URL
	baseURL = strings.TrimRight(baseURL, "/")

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true, // Default to insecure for self-signed certs
		},
		MaxIdleConns:        10,
		IdleConnTimeout:     90 * time.Second,
		MaxIdleConnsPerHost: 10,
	}

	client := &Client{
		baseURL:    baseURL,
		account:    account,
		password:   password,
		httpClient: &http.Client{Timeout: defaultTimeout, Transport: transport},
		maxRetries: defaultMaxRetries,
		retryDelay: defaultRetryDelay,
	}

	// Apply options
	for _, opt := range opts {
		opt(client)
	}

	return client, nil
}

// Login authenticates with the Synology server
func (c *Client) Login(ctx context.Context) error {
	params := url.Values{
		"api":     {"SYNO.API.Auth"},
		"version": {"3"},
		"method":  {"login"},
		"account": {c.account},
		"passwd":  {c.password},
	}

	var resp LoginResponse
	if err := c.doRequest(ctx, http.MethodPost, "/webapi/auth.cgi", params, nil, &resp); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	if !resp.Success {
		return fmt.Errorf("login failed: error code %d", resp.Error.Code)
	}

	c.sid = resp.Data.SID
	c.did = resp.Data.DID
	return nil
}

// Logout ends the session
func (c *Client) Logout(ctx context.Context) error {
	if c.sid == "" {
		return nil
	}

	params := url.Values{
		"api":     {"SYNO.API.Auth"},
		"version": {"3"},
		"method":  {"logout"},
		"_sid":    {c.sid},
	}

	var resp APIResponse[struct{ Success bool }]
	err := c.doRequest(ctx, http.MethodGet, "/webapi/auth.cgi", params, nil, &resp)

	c.sid = ""
	c.did = ""

	return err
}

// IsAuthenticated returns true if the client has a valid session
func (c *Client) IsAuthenticated() bool {
	return c.sid != ""
}

// ListAlbums retrieves a list of albums
func (c *Client) ListAlbums(ctx context.Context, offset, limit int) ([]Album, error) {
	if limit <= 0 {
		limit = defaultLimit
	}

	params := url.Values{
		"api":     {"SYNO.Foto.Browse.Album"},
		"version": {"1"},
		"method":  {"list"},
		"offset":  {strconv.Itoa(offset)},
		"limit":   {strconv.Itoa(limit)},
	}

	var resp APIResponse[AlbumListResponse]
	if err := c.doAuthenticatedRequest(ctx, http.MethodGet, "/webapi/entry.cgi", params, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Data.List, nil
}

// GetAlbumItems retrieves items in a specific album
func (c *Client) GetAlbumItems(ctx context.Context, albumID int, offset, limit int, additional []string) ([]Item, error) {
	if limit <= 0 {
		limit = defaultLimit
	}

	params := url.Values{
		"api":        {"SYNO.Foto.Browse.Item"},
		"version":    {"1"},
		"method":     {"list"},
		"offset":     {strconv.Itoa(offset)},
		"limit":      {strconv.Itoa(limit)},
		"album_id":   {strconv.Itoa(albumID)},
	}

	if len(additional) > 0 {
		additionalJSON, _ := json.Marshal(additional)
		params.Set("additional", string(additionalJSON))
	}

	var resp APIResponse[ItemListResponse]
	if err := c.doAuthenticatedRequest(ctx, http.MethodGet, "/webapi/entry.cgi", params, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Data.List, nil
}

// ListItems retrieves items from folders in the Personal Space
func (c *Client) ListItems(ctx context.Context, folderID *int, offset, limit int, additional []string) ([]Item, error) {
	if limit <= 0 {
		limit = defaultLimit
	}

	params := url.Values{
		"api":     {"SYNO.Foto.Browse.Item"},
		"version": {"1"},
		"method":  {"list"},
		"offset":  {strconv.Itoa(offset)},
		"limit":   {strconv.Itoa(limit)},
	}

	if folderID != nil {
		params.Set("folder_id", strconv.Itoa(*folderID))
	}

	if len(additional) > 0 {
		additionalJSON, _ := json.Marshal(additional)
		params.Set("additional", string(additionalJSON))
	}

	var resp APIResponse[ItemListResponse]
	if err := c.doAuthenticatedRequest(ctx, http.MethodGet, "/webapi/entry.cgi", params, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Data.List, nil
}

// ListAllItems retrieves all items (no folder filter)
func (c *Client) ListAllItems(ctx context.Context, offset, limit int, additional []string) ([]Item, error) {
	return c.ListItems(ctx, nil, offset, limit, additional)
}

// GetItemDetails retrieves detailed information about a specific item
func (c *Client) GetItemDetails(ctx context.Context, itemID int, additional []string) (*Item, error) {
	params := url.Values{
		"api":     {"SYNO.Foto.Browse.Item"},
		"version": {"1"},
		"method":  {"get"},
		"id":      {strconv.Itoa(itemID)},
	}

	if len(additional) > 0 {
		additionalJSON, _ := json.Marshal(additional)
		params.Set("additional", string(additionalJSON))
	}

	var resp APIResponse[Item]
	if err := c.doAuthenticatedRequest(ctx, http.MethodGet, "/webapi/entry.cgi", params, nil, &resp); err != nil {
		return nil, err
	}

	return &resp.Data, nil
}

// ListTags retrieves all general tags
func (c *Client) ListTags(ctx context.Context, offset, limit int) ([]Tag, error) {
	if limit <= 0 {
		limit = defaultLimit
	}

	params := url.Values{
		"api":     {"SYNO.Foto.Browse.GeneralTag"},
		"version": {"1"},
		"method":  {"list"},
		"offset":  {strconv.Itoa(offset)},
		"limit":   {strconv.Itoa(limit)},
	}

	var resp APIResponse[TagListResponse]
	if err := c.doAuthenticatedRequest(ctx, http.MethodGet, "/webapi/entry.cgi", params, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Data.List, nil
}

// ListPeople retrieves all people (face recognition)
func (c *Client) ListPeople(ctx context.Context, offset, limit int) ([]Person, error) {
	if limit <= 0 {
		limit = defaultLimit
	}

	params := url.Values{
		"api":     {"SYNO.Foto.Browse.Person"},
		"version": {"1"},
		"method":  {"list"},
		"offset":  {strconv.Itoa(offset)},
		"limit":   {strconv.Itoa(limit)},
	}

	var resp APIResponse[PersonListResponse]
	if err := c.doAuthenticatedRequest(ctx, http.MethodGet, "/webapi/entry.cgi", params, nil, &resp); err != nil {
		return nil, err
	}

	return resp.Data.List, nil
}

// DownloadItem downloads the original file
func (c *Client) DownloadItem(ctx context.Context, itemID int, cacheKey string) (io.ReadCloser, error) {
	params := url.Values{
		"api":       {"SYNO.Foto.Download"},
		"version":   {"1"},
		"method":    {"download"},
		"unit_id":   {fmt.Sprintf("[%d]", itemID)},
		"cache_key": {fmt.Sprintf("\"%s\"", cacheKey)},
	}

	reqURL, err := url.JoinPath(c.baseURL, "/webapi/entry.cgi")
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	fullURL := reqURL + "?" + params.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("download request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	return resp.Body, nil
}

// GetThumbnailURL returns the URL for a thumbnail
func (c *Client) GetThumbnailURL(itemID int, cacheKey string, size string) string {
	if size == "" {
		size = "xl" // default size
	}

	params := url.Values{
		"api":       {"SYNO.Foto.Thumbnail"},
		"version":   {"1"},
		"method":    {"get"},
		"id":        {strconv.Itoa(itemID)},
		"cache_key": {cacheKey},
		"type":      {"unit"},
		"size":      {size},
	}

	if c.sid != "" {
		params.Set("_sid", c.sid)
	}

	return c.baseURL + "/webapi/entry.cgi?" + params.Encode()
}

// doAuthenticatedRequest makes an authenticated API request
func (c *Client) doAuthenticatedRequest(ctx context.Context, method, path string, params url.Values, body io.Reader, result interface{}) error {
	// Ensure we're authenticated
	if !c.IsAuthenticated() {
		if err := c.Login(ctx); err != nil {
			return err
		}
	}

	// Add session ID to params
	params.Set("_sid", c.sid)

	return c.doRequest(ctx, method, path, params, body, result)
}

// doRequest performs an HTTP request with retries
func (c *Client) doRequest(ctx context.Context, method, path string, params url.Values, body io.Reader, result interface{}) error {
	var lastErr error

	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.retryDelay * time.Duration(attempt)):
				// Exponential backoff
			}
		}

		reqURL, err := url.JoinPath(c.baseURL, path)
		if err != nil {
			return fmt.Errorf("invalid URL: %w", err)
		}

		// Add query parameters
		if len(params) > 0 {
			reqURL = reqURL + "?" + params.Encode()
		}

		req, err := http.NewRequestWithContext(ctx, method, reqURL, body)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}

		req.Header.Set("Accept", "application/json")
		if body != nil {
			req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue // Retry on network error
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = fmt.Errorf("read response: %w", err)
			continue
		}

		if resp.StatusCode != http.StatusOK {
			lastErr = fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(respBody))
			continue
		}

		// Parse response
		var baseResp struct {
			Success bool   `json:"success"`
			Error   *Error `json:"error,omitempty"`
		}

		if err := json.Unmarshal(respBody, &baseResp); err != nil {
			lastErr = fmt.Errorf("parse response: %w", err)
			continue
		}

		// Handle authentication error - try to re-login
		if !baseResp.Success && baseResp.Error != nil && baseResp.Error.Code == 119 {
			// Session expired
			c.sid = ""
			if err := c.Login(ctx); err != nil {
				lastErr = err
				continue
			}
			// Update params with new session ID
			params.Set("_sid", c.sid)
			continue
		}

		if result != nil {
			if err := json.Unmarshal(respBody, result); err != nil {
				return fmt.Errorf("parse result: %w", err)
			}
		}

		return nil
	}

	return fmt.Errorf("max retries exceeded: %w", lastErr)
}
