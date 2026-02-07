package immich

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type testServer struct {
	// endpoint       string
	responseStatus int
	responseBody   string
}

func (ts *testServer) ServeHTTP(resp http.ResponseWriter, req *http.Request) {
	resp.WriteHeader(ts.responseStatus)
	_, _ = resp.Write([]byte(ts.responseBody))
}

func TestCall(t *testing.T) {
	tt := []struct {
		name        string
		requestFn   requestFunction
		expectedErr bool
		server      testServer
	}{
		{
			name:        "happy path",
			requestFn:   getRequest("/assets", setAcceptJSON()),
			expectedErr: false,
			server: testServer{
				responseStatus: http.StatusOK,
				responseBody:   `{"status": "All correct"}`,
			},
		},
		{
			name:        "bad url",
			requestFn:   getRequest("/ass\nets", setAcceptJSON()),
			expectedErr: true,
			server: testServer{
				responseStatus: http.StatusOK,
				responseBody:   `{"status": "All correct"}`,
			},
		},
		{
			name:        "post / ok",
			requestFn:   postRequest("/albums", "application/json", setAcceptJSON(), setJSONBody(struct{ Name string }{Name: "test"})),
			expectedErr: false,
			server: testServer{
				responseStatus: http.StatusOK,
				responseBody:   `{"Name": "test"}`,
			},
		},
		{
			name:        "bad request / post",
			requestFn:   postRequest("/albums", "application/json", setAcceptJSON(), setJSONBody(struct{ Name string }{Name: "test"})),
			expectedErr: true,
			server: testServer{
				responseStatus: http.StatusBadRequest,
				responseBody:   `{"error": "Bad request", "statusCode": "400", "message": ["String1","String2"]}`,
			},
		},
	}

	for _, tst := range tt {
		t.Run(tst.name, func(t *testing.T) {
			server := httptest.NewServer(&tst.server)
			defer server.Close()
			ctx := context.Background()
			ic, err := NewImmichClient(server.URL, "1234")
			if err != nil {
				t.Fail()
				return
			}
			// ic.EnableAppTrace(true)
			r := map[string]string{}
			err = ic.newServerCall(ctx, tst.name).do(tst.requestFn, responseJSON(&r))
			if tst.expectedErr && err == nil {
				t.Errorf("expected error, but no error")
			}
			if !tst.expectedErr && err != nil {
				t.Errorf("no error expected, but error: %s", err.Error())
			}
			if err != nil {
				t.Logf("error received: %s", err.Error())
			}
			t.Logf("response received: %#v", r)
		})
	}
}

// Test permission error detection
func TestIsPermissionError(t *testing.T) {
	tests := []struct {
		name     string
		status   int
		message  *ServerErrorMessage
		expected bool
	}{
		{
			name:     "401 status code",
			status:   401,
			message:  nil,
			expected: true,
		},
		{
			name:     "403 status code",
			status:   403,
			message:  nil,
			expected: true,
		},
		{
			name:   "permission keyword in message",
			status: 400,
			message: &ServerErrorMessage{
				Message: "You don't have permission to access this resource",
			},
			expected: true,
		},
		{
			name:   "unauthorized keyword in message",
			status: 400,
			message: &ServerErrorMessage{
				Message: "Unauthorized access",
			},
			expected: true,
		},
		{
			name:   "forbidden keyword in message",
			status: 400,
			message: &ServerErrorMessage{
				Message: "Forbidden",
			},
			expected: true,
		},
		{
			name:   "not allowed keyword in message",
			status: 400,
			message: &ServerErrorMessage{
				Message: "This action is not allowed",
			},
			expected: true,
		},
		{
			name:   "insufficient privileges keyword in message",
			status: 400,
			message: &ServerErrorMessage{
				Message: "Insufficient privileges",
			},
			expected: true,
		},
		{
			name:   "no_permission error string",
			status: 400,
			message: &ServerErrorMessage{
				Message: "no_permission",
			},
			expected: true,
		},
		{
			name:   "regular error not permission related",
			status: 500,
			message: &ServerErrorMessage{
				Message: "Internal server error",
			},
			expected: false,
		},
		{
			name:   "regular 400 error",
			status: 400,
			message: &ServerErrorMessage{
				Message: "Bad request",
			},
			expected: false,
		},
		{
			name:     "empty message",
			status:   500,
			message:  &ServerErrorMessage{},
			expected: false,
		},
		{
			name:     "nil message",
			status:   500,
			message:  nil,
			expected: false,
		},
		{
			name:   "false positive - word permission in non-error context",
			status: 500,
			message: &ServerErrorMessage{
				Message: "Please check your file permissions on disk",
			},
			expected: true, // This is intentionally detected - better false positive than false negative
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isPermissionError(tt.status, tt.message)
			if result != tt.expected {
				t.Errorf("isPermissionError() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// Test admin endpoint detection
func TestIsAdminEndpoint(t *testing.T) {
	tests := []struct {
		name     string
		endpoint string
		expected bool
	}{
		{
			name:     "GetJobs is admin endpoint",
			endpoint: EndPointGetJobs,
			expected: true,
		},
		{
			name:     "SendJobCommand is admin endpoint",
			endpoint: EndPointSendJobCommand,
			expected: true,
		},
		{
			name:     "CreateJob is admin endpoint",
			endpoint: EndPointCreateJob,
			expected: true,
		},
		{
			name:     "GetAllAlbums is not admin endpoint",
			endpoint: EndPointGetAllAlbums,
			expected: false,
		},
		{
			name:     "AssetUpload is not admin endpoint",
			endpoint: EndPointAssetUpload,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAdminEndpoint(tt.endpoint)
			if result != tt.expected {
				t.Errorf("isAdminEndpoint() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// Test permission error messages include enhanced information
func TestCallErrorWithPermissions(t *testing.T) {
	tests := []struct {
		name               string
		endpoint           string
		status             int
		message            *ServerErrorMessage
		expectPermissions  bool
		expectAdmin        bool
		expectDocsURL      bool
		expectOriginalMsg  bool
	}{
		{
			name:     "401 error includes permission info",
			endpoint: EndPointGetAllAssets,
			status:   401,
			message: &ServerErrorMessage{
				Message: "Unauthorized",
			},
			expectPermissions: true,
			expectAdmin:       false,
			expectDocsURL:     true,
			expectOriginalMsg: true,
		},
		{
			name:     "403 error includes permission info",
			endpoint: EndPointCreateAlbum,
			status:   403,
			message: &ServerErrorMessage{
				Message: "Forbidden",
			},
			expectPermissions: true,
			expectAdmin:       false,
			expectDocsURL:     true,
			expectOriginalMsg: true,
		},
		{
			name:     "admin endpoint shows admin requirements",
			endpoint: EndPointGetJobs,
			status:   403,
			message: &ServerErrorMessage{
				Message: "Forbidden",
			},
			expectPermissions: true,
			expectAdmin:       true,
			expectDocsURL:     true,
			expectOriginalMsg: true,
		},
		{
			name:     "regular error does not include permission info",
			endpoint: EndPointGetAllAssets,
			status:   500,
			message: &ServerErrorMessage{
				Message: "Internal server error",
			},
			expectPermissions: false,
			expectAdmin:       false,
			expectDocsURL:     false,
			expectOriginalMsg: true,
		},
		{
			name:     "no_permission error from tag operations",
			endpoint: EndPointTagAssets,
			status:   400,
			message: &ServerErrorMessage{
				Message: "no_permission",
			},
			expectPermissions: true,
			expectAdmin:       false,
			expectDocsURL:     true,
			expectOriginalMsg: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ce := callError{
				endPoint: tt.endpoint,
				method:   "GET",
				url:      "http://example.com/api/test",
				status:   tt.status,
				message:  tt.message,
			}

			errorMsg := ce.Error()

			// Check original message is preserved
			if tt.expectOriginalMsg && tt.message != nil {
				if !strings.Contains(errorMsg, tt.message.Message) {
					t.Errorf("Error message should contain original message '%s', got: %s", tt.message.Message, errorMsg)
				}
			}

			// Check permission list is included
			if tt.expectPermissions {
				if !strings.Contains(errorMsg, "asset.read") {
					t.Errorf("Error message should contain permission list, got: %s", errorMsg)
				}
				if !strings.Contains(errorMsg, "Your API key must have the following permissions") {
					t.Errorf("Error message should contain permission header, got: %s", errorMsg)
				}
			} else {
				if strings.Contains(errorMsg, "Your API key must have the following permissions") {
					t.Errorf("Error message should NOT contain permission info for non-permission errors, got: %s", errorMsg)
				}
			}

			// Check admin-specific information
			if tt.expectAdmin {
				if !strings.Contains(errorMsg, "This endpoint requires an admin API key") {
					t.Errorf("Error message should indicate admin requirement, got: %s", errorMsg)
				}
				if !strings.Contains(errorMsg, "job.create") {
					t.Errorf("Error message should contain admin permissions, got: %s", errorMsg)
				}
			}

			// Check documentation URL
			if tt.expectDocsURL {
				if !strings.Contains(errorMsg, DocsPermissionsURL) {
					t.Errorf("Error message should contain documentation URL, got: %s", errorMsg)
				}
			} else {
				if strings.Contains(errorMsg, DocsPermissionsURL) {
					t.Errorf("Error message should NOT contain documentation URL for non-permission errors, got: %s", errorMsg)
				}
			}
		})
	}
}

// Test format permission list helper
func TestFormatPermissionList(t *testing.T) {
	tests := []struct {
		name        string
		permissions []string
		expected    string
	}{
		{
			name:        "single permission",
			permissions: []string{"asset.read"},
			expected:    "asset.read",
		},
		{
			name:        "multiple permissions",
			permissions: []string{"asset.read", "asset.write", "album.create"},
			expected:    "asset.read, asset.write, album.create",
		},
		{
			name:        "empty list",
			permissions: []string{},
			expected:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatPermissionList(tt.permissions)
			if result != tt.expected {
				t.Errorf("formatPermissionList() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// Test that all required permissions are present in the constants
func TestRequiredPermissions(t *testing.T) {
	expectedUserPermissions := []string{
		"asset.read",
		"asset.statistics",
		"asset.update",
		"asset.upload",
		"asset.copy",
		"asset.replace",
		"asset.delete",
		"asset.download",
		"album.create",
		"album.read",
		"albumAsset.create",
		"server.about",
		"stack.create",
		"tag.asset",
		"tag.create",
		"user.read",
	}

	if len(RequiredUserPermissions) != len(expectedUserPermissions) {
		t.Errorf("RequiredUserPermissions has %d permissions, expected %d", len(RequiredUserPermissions), len(expectedUserPermissions))
	}

	permMap := make(map[string]bool)
	for _, perm := range RequiredUserPermissions {
		permMap[perm] = true
	}

	for _, expected := range expectedUserPermissions {
		if !permMap[expected] {
			t.Errorf("RequiredUserPermissions missing permission: %s", expected)
		}
	}

	expectedAdminPermissions := []string{"job.create", "job.read"}
	if len(RequiredAdminPermissions) != len(expectedAdminPermissions) {
		t.Errorf("RequiredAdminPermissions has %d permissions, expected %d", len(RequiredAdminPermissions), len(expectedAdminPermissions))
	}

	for i, perm := range RequiredAdminPermissions {
		if perm != expectedAdminPermissions[i] {
			t.Errorf("RequiredAdminPermissions[%d] = %s, expected %s", i, perm, expectedAdminPermissions[i])
		}
	}
}
