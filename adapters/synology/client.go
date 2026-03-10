package synology

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
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
	synotoken  string // Required for some APIs
	httpClient *http.Client
	maxRetries int
	retryDelay time.Duration
	logger     *slog.Logger // Optional logger for debugging
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

// WithLogger sets a logger for debugging
func WithLogger(logger *slog.Logger) ClientOption {
	return func(c *Client) {
		c.logger = logger
	}
}

// NewClient creates a new Synology Photos API client
func NewClient(baseURL, account, password string, opts ...ClientOption) (*Client, error) {
	// Clean up the base URL
	baseURL = strings.TrimRight(baseURL, "/")

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment, // Enable HTTP_PROXY/HTTPS_PROXY support
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

// APIInfo represents the SYNO.API.Info response
type APIInfo struct {
	Path       string `json:"path"`
	MinVersion int    `json:"minVersion"`
	MaxVersion int    `json:"maxVersion"`
}

// QueryAPIInfo queries available APIs and returns API paths and versions
func (c *Client) QueryAPIInfo(ctx context.Context, apiName string) (*APIInfo, error) {
	params := url.Values{
		"api":     {"SYNO.API.Info"},
		"version": {"1"},
		"method":  {"query"},
		"query":   {apiName},
	}

	var resp APIResponse[map[string]APIInfo]
	if err := c.doRequestNoAuth(ctx, http.MethodGet, "/webapi/query.cgi", params, &resp); err != nil {
		return nil, fmt.Errorf("query API info failed: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("query API info failed: error %d", resp.Error.Code)
	}

	if apiInfo, ok := resp.Data[apiName]; ok {
		return &apiInfo, nil
	}

	return nil, fmt.Errorf("API %s not found", apiName)
}

// Login authenticates with the Synology server
func (c *Client) Login(ctx context.Context) error {
	// First, query API info to get the correct path and version
	apiInfo, err := c.QueryAPIInfo(ctx, "SYNO.API.Auth")
	if err != nil {
		// Fallback to default values
		apiInfo = &APIInfo{
			Path:       "auth.cgi",
			MaxVersion: 6,
		}
	}

	// Build login parameters according to DSM API spec
	params := url.Values{
		"api":               {"SYNO.API.Auth"},
		"version":           {strconv.Itoa(apiInfo.MaxVersion)},
		"method":            {"login"},
		"account":           {c.account},
		"passwd":            {c.password},
		"format":            {"sid"},
		"enable_syno_token": {"yes"},
	}

	// Use the correct API path from API info
	authPath := "/webapi/" + apiInfo.Path

	var resp LoginResponse
	if err := c.doRequestNoAuth(ctx, http.MethodGet, authPath, params, &resp); err != nil {
		return fmt.Errorf("login failed: %w", err)
	}

	c.sid = resp.Data.SID
	c.did = resp.Data.DID
	c.synotoken = resp.Data.SynoToken
	if c.did == "" {
		c.did = resp.Data.DeviceID // Fallback to device_id if did is empty
	}
	return nil
}

// getErrorMessage returns a human-readable error message for Synology API error codes
func (c *Client) getErrorMessage(code int) string {
	switch code {
	case 100:
		return "Unknown error"
	case 101:
		return "Invalid parameters - wrong API, method, or version"
	case 102:
		return "The requested method does not exist"
	case 103:
		return "The requested method does not support the requested version"
	case 104:
		return "Version not supported"
	case 105:
		return "Session ID not found (need to login again)"
	case 106:
		return "Incorrect account or password"
	case 107:
		return "Permission denied"
	case 108:
		return "OTP code required (two-factor authentication)"
	case 109:
		return "Failed to authenticate with OTP"
	case 110:
		return "Max TOTP retries reached"
	case 111:
		return "Password change required"
	case 112:
		return "Strong password required"
	case 113:
		return "Strong password required for admin"
	case 119:
		return "SID not found or session timeout"
	case 400:
		return "Invalid credentials in request"
	case 401:
		return "Guest account disabled"
	case 402:
		return "Account disabled"
	case 403:
		return "Permission denied"
	case 404:
		return "OTP code required"
	case 405:
		return "OTP authenticate failed"
	default:
		return fmt.Sprintf("Unknown error code %d", code)
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
	err := c.doRequestNoAuth(ctx, http.MethodGet, "/webapi/auth.cgi", params, &resp)

	c.sid = ""
	c.did = ""

	return err
}

// IsAuthenticated returns true if the client has a valid session
func (c *Client) IsAuthenticated() bool {
	return c.sid != ""
}

// ListAlbums retrieves a list of albums
// Note: SYNO.Foto.Browse.Album may not be available in all DSM versions
func (c *Client) ListAlbums(ctx context.Context, offset, limit int) ([]Album, error) {
	if limit <= 0 {
		limit = defaultLimit
	}

	// Try SYNO.Foto.Browse.Album first (version 1)
	params := url.Values{
		"api":     {"SYNO.Foto.Browse.Album"},
		"version": {"1"},
		"method":  {"list"},
		"offset":  {strconv.Itoa(offset)},
		"limit":   {strconv.Itoa(limit)},
	}

	var resp APIResponse[AlbumListResponse]
	err := c.doAuthenticatedRequest(ctx, http.MethodPost, "/webapi/entry.cgi", params, nil, &resp)
	if err != nil {
		// If Album API fails, try getting items from timeline instead
		return nil, fmt.Errorf("album API not available: %w", err)
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
		"version":    {"4"},
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
	if err := c.doAuthenticatedRequest(ctx, http.MethodPost, "/webapi/entry.cgi", params, nil, &resp); err != nil {
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
		"version": {"4"},
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
	if err := c.doAuthenticatedRequest(ctx, http.MethodPost, "/webapi/entry.cgi", params, nil, &resp); err != nil {
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
		"version": {"4"},
		"method":  {"list"},
		"offset":  {strconv.Itoa(offset)},
		"limit":   {strconv.Itoa(limit)},
	}

	var resp APIResponse[TagListResponse]
	if err := c.doAuthenticatedRequest(ctx, http.MethodPost, "/webapi/entry.cgi", params, nil, &resp); err != nil {
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
		"cache_key": {cacheKey},
	}

	// Build request URL
	reqURL, err := url.JoinPath(c.baseURL, "/webapi/entry.cgi")
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	var lastErr error
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.retryDelay * time.Duration(attempt)):
			}
		}

		reqBody := strings.NewReader(params.Encode())
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, reqBody)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		// Set required headers
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
		req.Header.Set("Accept", "*/*")
		if c.synotoken != "" {
			req.Header.Set("X-SYNO-TOKEN", c.synotoken)
		}
		if c.sid != "" {
			req.Header.Set("Cookie", fmt.Sprintf("id=%s", c.sid))
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Check if response is JSON error
		contentType := resp.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			var errResp struct {
				Success bool   `json:"success"`
				Error   *Error `json:"error,omitempty"`
			}
			if json.Unmarshal(body, &errResp) == nil && !errResp.Success && errResp.Error != nil {
				if errResp.Error.Code == 119 && attempt < c.maxRetries-1 {
					// Session expired, re-login and retry
					if err := c.Login(ctx); err != nil {
						lastErr = fmt.Errorf("re-login failed: %w", err)
						continue
					}
					continue
				}
				return nil, fmt.Errorf("download API error %d: %s", errResp.Error.Code, c.getErrorMessage(errResp.Error.Code))
			}
			return nil, fmt.Errorf("download failed: %s", string(body))
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("http %d", resp.StatusCode)
			continue
		}

		return resp.Body, nil
	}

	return nil, fmt.Errorf("download failed after retries: %w", lastErr)
}

// DownloadLivePhoto downloads a live photo using the source download API.
// Returns the response body, content-type, and whether it's a ZIP file (based on Content-Disposition).
// Note: Synology may return either a ZIP (containing both image and video) or just the image file.
func (c *Client) DownloadLivePhoto(ctx context.Context, itemID int, filename string, logger *slog.Logger) (io.ReadCloser, string, bool, error) {
	params := url.Values{
		"api":            {"SYNO.Foto.Download"},
		"version":        {"2"},
		"method":         {"download"},
		"item_id":        {fmt.Sprintf("[%d]", itemID)},
		"download_type":  {"source"},
		"force_download": {"true"},
	}

	// Build request URL with filename in path (like browser does)
	reqURL, err := url.JoinPath(c.baseURL, "/webapi/entry.cgi", filename)
	if err != nil {
		return nil, "", false, fmt.Errorf("invalid URL: %w", err)
	}

	// Add synotoken as query param if available
	if c.synotoken != "" {
		reqURL = reqURL + "?SynoToken=" + url.QueryEscape(c.synotoken)
	}

	var lastErr error
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, "", false, ctx.Err()
			case <-time.After(c.retryDelay * time.Duration(attempt)):
			}
		}

		reqBody := strings.NewReader(params.Encode())
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, reqURL, reqBody)
		if err != nil {
			return nil, "", false, fmt.Errorf("create request: %w", err)
		}

		// Set required headers
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")
		req.Header.Set("Accept", "*/*")
		if c.synotoken != "" {
			req.Header.Set("X-SYNO-TOKEN", c.synotoken)
		}
		if c.sid != "" {
			req.Header.Set("Cookie", fmt.Sprintf("id=%s", c.sid))
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			continue
		}

		// Check if response is JSON error
		contentType := resp.Header.Get("Content-Type")
		if strings.Contains(contentType, "application/json") {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			var errResp struct {
				Success bool   `json:"success"`
				Error   *Error `json:"error,omitempty"`
			}
			if json.Unmarshal(body, &errResp) == nil && !errResp.Success && errResp.Error != nil {
				if errResp.Error.Code == 119 && attempt < c.maxRetries-1 {
					// Session expired, re-login and retry
					if err := c.Login(ctx); err != nil {
						lastErr = fmt.Errorf("re-login failed: %w", err)
						continue
					}
					continue
				}
				return nil, "", false, fmt.Errorf("download API error %d: %s", errResp.Error.Code, c.getErrorMessage(errResp.Error.Code))
			}
			return nil, "", false, fmt.Errorf("download failed: %s", string(body))
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close()
			lastErr = fmt.Errorf("http %d", resp.StatusCode)
			continue
		}

		// Check if response is a ZIP by looking at Content-Disposition header
		contentDisposition := resp.Header.Get("Content-Disposition")
		isZip := strings.HasSuffix(strings.ToLower(contentDisposition), ".zip") ||
			strings.Contains(strings.ToLower(contentDisposition), "filename=\"download.zip\"") ||
			strings.Contains(strings.ToLower(contentDisposition), "filename*=utf-8''download.zip")

		if c.logger != nil {
			c.logger.Debug("Downloaded live photo", "item_id", itemID, "content_type", contentType,
				"content_disposition", contentDisposition, "is_zip", isZip, "content_length", resp.Header.Get("Content-Length"))
		}
		return resp.Body, contentType, isZip, nil
	}

	return nil, "", false, fmt.Errorf("download failed after retries: %w", lastErr)
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

// doRequestNoAuth performs an HTTP request without authentication (used for login and API info)
func (c *Client) doRequestNoAuth(ctx context.Context, method, path string, params url.Values, result interface{}) error {
	reqURL, err := url.JoinPath(c.baseURL, path)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if len(params) > 0 {
		reqURL = reqURL + "?" + params.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, method, reqURL, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("http %d: %s", resp.StatusCode, string(respBody))
	}

	if result != nil {
		if err := json.Unmarshal(respBody, result); err != nil {
			return fmt.Errorf("parse result: %w", err)
		}
	}

	return nil
}

// doAuthenticatedRequest makes an authenticated API request
func (c *Client) doAuthenticatedRequest(ctx context.Context, method, path string, params url.Values, body io.Reader, result interface{}) error {
	// Ensure we're authenticated
	if !c.IsAuthenticated() {
		if err := c.Login(ctx); err != nil {
			return err
		}
	}

	return c.doRequestWithAuth(ctx, method, path, params, body, result)
}

// doRequestWithAuth makes a request with authentication headers
func (c *Client) doRequestWithAuth(ctx context.Context, method, path string, params url.Values, body io.Reader, result interface{}) error {
	for attempt := 0; attempt < c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(c.retryDelay * time.Duration(attempt)):
			}
		}

		reqURL, err := url.JoinPath(c.baseURL, path)
		if err != nil {
			return fmt.Errorf("invalid URL: %w", err)
		}

		// Clone params and add auth
		reqParams := url.Values{}
		for k, v := range params {
			reqParams[k] = v
		}

		// For POST requests, put params in body; for GET, put in URL
		var reqBody io.Reader
		var contentType string
		if method == http.MethodPost {
			reqBody = strings.NewReader(reqParams.Encode())
			contentType = "application/x-www-form-urlencoded; charset=UTF-8"
		} else {
			if len(reqParams) > 0 {
				reqURL = reqURL + "?" + reqParams.Encode()
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, reqURL, reqBody)
		if err != nil {
			return fmt.Errorf("create request: %w", err)
		}

		req.Header.Set("Accept", "application/json")
		if contentType != "" {
			req.Header.Set("Content-Type", contentType)
		}

			// Add X-SYNO-TOKEN header for CSRF protection
		if c.synotoken != "" {
			req.Header.Set("X-SYNO-TOKEN", c.synotoken)
		}

		// Add Cookie with session ID (required for some APIs)
		if c.sid != "" {
			req.Header.Set("Cookie", fmt.Sprintf("id=%s", c.sid))
		}

		resp, err := c.httpClient.Do(req)
		if err != nil {
			if attempt < c.maxRetries-1 {
				continue
			}
			return fmt.Errorf("http request failed: %w", err)
		}

		respBody, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			if attempt < c.maxRetries-1 {
				continue
			}
			return fmt.Errorf("read response: %w", err)
		}

		if resp.StatusCode != http.StatusOK {
			if attempt < c.maxRetries-1 {
				continue
			}
			return fmt.Errorf("http %d: %s", resp.StatusCode, string(respBody))
		}

		// Debug logging
		if c.logger != nil {
			c.logger.Debug("synology api call", "method", method, "url", reqURL, "response", string(respBody))
		}

		// Parse response
		var baseResp struct {
			Success bool   `json:"success"`
			Error   *Error `json:"error,omitempty"`
		}

		if err := json.Unmarshal(respBody, &baseResp); err != nil {
			if attempt < c.maxRetries-1 {
				continue
			}
			return fmt.Errorf("parse response: %w", err)
		}

		// Handle session expiration - re-login and retry
		if !baseResp.Success && baseResp.Error != nil && baseResp.Error.Code == 119 {
			c.sid = ""
			if err := c.Login(ctx); err != nil {
				if attempt < c.maxRetries-1 {
					continue
				}
				return fmt.Errorf("re-login after session expired: %w", err)
			}
			continue
		}

		// Handle other API errors
		if !baseResp.Success {
			if baseResp.Error != nil {
				return fmt.Errorf("synology api error %d: %s", baseResp.Error.Code, c.getErrorMessage(baseResp.Error.Code))
			}
			return fmt.Errorf("synology api error: success=false")
		}

		// Parse result
		if result != nil {
			if err := json.Unmarshal(respBody, result); err != nil {
				return fmt.Errorf("parse result: %w", err)
			}
		}

		return nil
	}

	return fmt.Errorf("max retries exceeded")
}

