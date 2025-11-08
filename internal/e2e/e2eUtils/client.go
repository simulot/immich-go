package e2eutils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
)

type Token string

// getAPIURL returns the Immich API URL, checking E2E_SERVER environment variable first
func getAPIURL() string {
	// Check for environment variable (set by CI workflow)
	if envURL := os.Getenv("e2e_url"); envURL != "" {
		return strings.TrimSuffix(envURL, "/") + "/api"
	}
	// Default for local development
	return "http://localhost:2283/api"
}

func do(method string, url string, body any, token Token) (*http.Response, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("can't post %s: %w", url, err)
	}
	req, err := http.NewRequest(method, url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("can't post %s: %w", url, err)
	}
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+string(token))
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("can't post %s: %w", url, err)
	}
	if resp.StatusCode > 299 {
		defer resp.Body.Close()
		var er ErrorResponse
		err = json.NewDecoder(resp.Body).Decode(&er)
		if err != nil {
			return nil, fmt.Errorf("can't post %s: %s", url, resp.Status)
		}
		return nil, fmt.Errorf("can't post %s: %s,%s", url, resp.Status, er.GetMessage())
	}
	return resp, nil
}

func post(url string, body any, token Token) (*http.Response, error) {
	return do(http.MethodPost, url, body, token)
}

func put(url string, body any, token Token) (*http.Response, error) {
	return do(http.MethodPut, url, body, token)
}

type ErrorResponse struct {
	Message       any    `json:"message"`
	Error         string `json:"error"`
	StatusCode    int    `json:"statusCode"`
	CorrelationID string `json:"correlationId"`
}

// GetMessage concatenates all messages into a single string, handling both string and []string formats
func (e ErrorResponse) GetMessage() string {
	switch m := e.Message.(type) {
	case string:
		return m
	case []interface{}:
		var msgs []string
		for _, v := range m {
			if s, ok := v.(string); ok {
				msgs = append(msgs, s)
			}
		}
		return strings.Join(msgs, "; ")
	default:
		return fmt.Sprintf("%v", m)
	}
}
