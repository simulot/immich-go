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
	if envURL := os.Getenv("E2E_SERVER"); envURL != "" {
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

func AdminSetup(email, password, name string) error {
	type SignUpDto struct {
		Email    string `json:"email"`
		Name     string `json:"name"`
		Password string `json:"password"`
	}

	type UserAdminResponseDto struct {
		Email   string `json:"email"`
		ID      string `json:"id"`
		IsAdmin bool   `json:"isAdmin"`
		Name    string `json:"name"`
		OauthID string `json:"oauthId"`

		// AvatarColor          UserAvatarColor `json:"avatarColor"`
		// CreatedAt            string          `json:"createdAt"`
		// DeletedAt            *string         `json:"deletedAt"`
		// License              *UserLicense    `json:"license"`
		// ProfileChangedAt     string          `json:"profileChangedAt"`
		// ProfileImagePath     string          `json:"profileImagePath"`
		// QuotaSizeInBytes     *int64          `json:"quotaSizeInBytes"`
		// QuotaUsageInBytes    *int64          `json:"quotaUsageInBytes"`
		// ShouldChangePassword bool            `json:"shouldChangePassword"`
		// Status               UserStatus      `json:"status"`
		// StorageLabel         *string         `json:"storageLabel"`
		// UpdatedAt            string          `json:"updatedAt"`
	}
	s := SignUpDto{
		Email:    email,
		Name:     name,
		Password: password,
	}
	resp, err := post(getAPIURL()+"/auth/admin-sign-up", &s, "")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("can't create admin user: %s", resp.Status)
	}

	r := UserAdminResponseDto{}
	err = json.NewDecoder(resp.Body).Decode(&r)
	return err
}

func UserLogin(email, password string) (Token, error) {
	type LoginCredentialDto struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	type LoginResponseDto struct {
		AccessToken          string `json:"accessToken"`
		IsAdmin              bool   `json:"isAdmin"`
		IsOnboarded          bool   `json:"isOnboarded"`
		Name                 string `json:"name"`
		ProfileImagePath     string `json:"profileImagePath"`
		ShouldChangePassword bool   `json:"shouldChangePassword"`
		UserEmail            string `json:"userEmail"`
		UserID               string `json:"userId"`
	}

	login := LoginCredentialDto{
		Email:    email,
		Password: password,
	}

	resp, err := post(getAPIURL()+"/auth/login", &login, "")
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// if resp.StatusCode != http.StatusOK {
	// 	return "", fmt.Errorf("login failed: %s", resp.Status)
	// }

	var loginResponse LoginResponseDto
	err = json.NewDecoder(resp.Body).Decode(&loginResponse)
	if err != nil {
		return "", fmt.Errorf("login failed: %s", resp.Status)
	}
	return Token(loginResponse.AccessToken), nil
}

func SetUserOnboarding(token Token, onboarding bool) error {
	type OnboardingDto struct {
		IsOnboarded bool `json:"isOnboarded"`
	}
	o := OnboardingDto{
		IsOnboarded: onboarding,
	}

	resp, err := put(getAPIURL()+"/users/me/onboarding", o, token)
	if err != nil {
		return err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("setUserOnboarding failed: %s", resp.Status)
	}
	return nil
}

func CreateApiKey(token Token, name string, permissions []string) (string, error) {
	type APIKeyCreateDto struct {
		Name        string   `json:"name,omitempty"`
		Permissions []string `json:"permissions"`
	}
	type APIKeyResponseDto struct {
		CreatedAt   string   `json:"createdAt"`
		ID          string   `json:"id"`
		Name        string   `json:"name"`
		Permissions []string `json:"permissions"`
		UpdatedAt   string   `json:"updatedAt"`
	}
	type APIKeyCreateResponseDto struct {
		APIKey APIKeyResponseDto `json:"apiKey"`
		Secret string            `json:"secret"`
	}
	resp, err := post(getAPIURL()+"/api-keys", APIKeyCreateDto{
		Name:        name,
		Permissions: permissions,
	}, token)
	if err != nil {
		return "", err
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("CreateApiKey failed: %s", resp.Status)
	}

	r := APIKeyCreateResponseDto{}
	err = json.NewDecoder(resp.Body).Decode(&r)
	if err != nil {
		return "", err
	}
	return r.Secret, nil
}

func CreateUser(adminToken Token, email string, password string, name string) error {
	type UserAdminCreateDto struct {
		Email                string `json:"email" validate:"required,email"`
		Name                 string `json:"name" validate:"required"`
		Password             string `json:"password" validate:"required"`
		ShouldChangePassword bool   `json:"shouldChangePassword"`
	}
	u := UserAdminCreateDto{
		Email:                email,
		Password:             password,
		Name:                 name,
		ShouldChangePassword: false,
	}

	// uResp := map[string]any
	resp, err := post(getAPIURL()+"/admin/users", u, adminToken)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
