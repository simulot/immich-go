package immich

import "strings"

// DocsPermissionsURL points to the API permissions documentation
const DocsPermissionsURL = "https://github.com/simulot/immich-go/blob/main/docs/installation.md#api-permissions"

// RequiredUserPermissions lists all permissions needed for a user API key
var RequiredUserPermissions = []string{
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

// RequiredAdminPermissions lists permissions that require an admin API key
var RequiredAdminPermissions = []string{
	"job.create",
	"job.read",
}

// AdminOnlyEndpoints lists endpoints that require admin permissions
var AdminOnlyEndpoints = []string{
	EndPointGetJobs,
	EndPointSendJobCommand,
	EndPointCreateJob,
}

// isPermissionError detects if an error is permission-related based on status code and message content
func isPermissionError(status int, message *ServerErrorMessage) bool {
	// Check HTTP status codes
	if status == 401 || status == 403 {
		return true
	}

	// Check message content for permission-related keywords
	if message != nil && message.Message != "" {
		msgLower := strings.ToLower(message.Message)
		permissionKeywords := []string{
			"permission",
			"unauthorized",
			"forbidden",
			"not allowed",
			"insufficient privileges",
			"no_permission",
		}
		for _, keyword := range permissionKeywords {
			if strings.Contains(msgLower, keyword) {
				return true
			}
		}
	}

	return false
}

// isAdminEndpoint checks if an endpoint requires admin permissions
func isAdminEndpoint(endpoint string) bool {
	for _, adminEndpoint := range AdminOnlyEndpoints {
		if endpoint == adminEndpoint {
			return true
		}
	}
	return false
}

// formatPermissionList formats a slice of permissions as a comma-separated string
func formatPermissionList(permissions []string) string {
	return strings.Join(permissions, ", ")
}
