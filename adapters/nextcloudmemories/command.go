package nextcloudmemories

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	pathpkg "path"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper/osfs"
	"github.com/simulot/immich-go/internal/nextcloud"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

const (
	nextcloudMemoriesSourceName = "nextcloud-memories"
	albumUserRoleEditor         = "editor"
)

// Command holds the CLI state for the Nextcloud Memories source scaffold.
type Command struct {
	NextcloudURL           string
	NextcloudUser          string
	NextcloudPassword      string
	NextcloudLocalDir      string
	NextcloudSkipVerifySSL bool
	NextcloudClientTimeout time.Duration
	DiscoverOnly           bool
	TimelineRoots          []string
	SyncAlbums             bool
	SyncTags               bool
	RequireIndexed         bool
	TagAlbumMembership     bool
	UserMaps               []string

	newRemoteFS           func(context.Context, *nextcloud.Client, string) (fs.FS, error)
	getAlbumCollaborators func(context.Context, *nextcloud.Client, string, string) ([]nextcloud.AlbumCollaborator, error)

	discover func(context.Context, nextcloud.Config) (*nextcloud.MemoriesDiscovery, error)
	app      *app.Application

	discovery     *nextcloud.MemoriesDiscovery
	client        *nextcloud.Client
	metadataIndex *memoriesMetadataIndex
	sourceFS      fs.FS
	selectedRoots []string
	userMappings  map[string]string
	albumUsersMu  sync.Mutex
	albumUsers    map[int][]adapters.AlbumUser
}

func (nc *Command) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVar(&nc.NextcloudURL, "nextcloud-url", "", "Nextcloud base URL")
	flags.StringVar(&nc.NextcloudUser, "nextcloud-user", "", "Nextcloud username")
	flags.StringVar(&nc.NextcloudPassword, "nextcloud-password", "", "Nextcloud password or app password")
	flags.StringVar(&nc.NextcloudLocalDir, "nextcloud-local-dir", "", "Prefer a local directory containing a synced Nextcloud copy to speed up file reads while keeping Memories as the source of truth")
	flags.BoolVar(&nc.NextcloudSkipVerifySSL, "nextcloud-skip-verify-ssl", false, "Skip TLS verification for the source Nextcloud server")
	flags.DurationVar(&nc.NextcloudClientTimeout, "nextcloud-client-timeout", 5*time.Minute, "Timeout for source Nextcloud API calls")
	flags.BoolVar(&nc.DiscoverOnly, "discover-only", false, "Print detected Memories configuration and exit")
	flags.StringSliceVar(&nc.TimelineRoots, "timeline-root", nil, "Limit the import to configured Memories timeline roots. Can be specified multiple times")
	flags.BoolVar(&nc.SyncAlbums, "sync-albums", true, "Recreate owned Memories albums in Immich")
	flags.BoolVar(&nc.SyncTags, "sync-tags", false, "Transfer source Memories system tags to Immich tags")
	flags.BoolVar(&nc.RequireIndexed, "require-indexed", false, "Fail if files are found under the selected Memories roots without matching Memories metadata")
	flags.BoolVar(&nc.TagAlbumMembership, "tag-album-membership", false, "Add synthetic tags encoding source album membership to support later shared-album reconciliation")
	flags.StringArrayVar(&nc.UserMaps, "user-map", nil, "Map a Nextcloud user ID to an Immich user ID for album share restoration (<nextcloud-user>=<immich-user-id>). Can be specified multiple times")
}

// NewFromNextcloudMemoriesCommand creates the command scaffold for the planned
// Nextcloud Memories source importer.
func NewFromNextcloudMemoriesCommand(ctx context.Context, parent *cobra.Command, app *app.Application, runner adapters.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-nextcloud-memories [flags]",
		Short: "Upload photos from a Nextcloud Memories library",
		Long: strings.TrimSpace(`Import photos and videos from a Nextcloud instance that uses the Memories app.

This source is intended to migrate the configured Memories library for a user account,
not arbitrary Nextcloud storage paths. The planned workflow is:

1. Authenticate to Nextcloud.
2. Discover the user's effective Memories configuration.
3. Resolve the configured timeline roots.
4. Import assets and supported metadata from that scope.

This command is used to iterate on the UX and flag contract while the source implementation continues to mature.`),
		Example: strings.TrimSpace(`  immich-go upload from-nextcloud-memories \
    --nextcloud-url=https://cloud.example.com \
    --nextcloud-user=alice \
    --nextcloud-password="$NEXTCLOUD_APP_PASSWORD" \
    --discover-only

  immich-go upload from-nextcloud-memories \
    --nextcloud-url=https://cloud.example.com \
    --nextcloud-user=alice \
    --nextcloud-password="$NEXTCLOUD_APP_PASSWORD" \
		--nextcloud-local-dir="$HOME/Nextcloud" \
    --timeline-root=/Photos \
    --server=http://immich.example.com:2283 \
    --api-key="$IMMICH_API_KEY"`),
		Args: cobra.NoArgs,
	}
	cmd.SetContext(ctx)

	nc := &Command{app: app}
	nc.RegisterFlags(cmd.Flags())

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return nc.Run(ctx, cmd, runner)
	}

	return cmd
}

// Run validates the source configuration, prepares the selected Memories scope,
// and delegates the actual upload lifecycle to the shared upload runner.
func (nc *Command) Run(ctx context.Context, cmd *cobra.Command, runner adapters.Runner) error {
	if err := nc.validate(); err != nil {
		return err
	}
	if cmd != nil {
		cmd.Println(nc.intentSummary())
	}

	if nc.app != nil {
		nc.app.Log().Message("Nextcloud Memories command scaffold invoked (discover-only=%t)", nc.DiscoverOnly)
	}

	if nc.DiscoverOnly {
		discovery, err := nc.runDiscovery(ctx)
		if err != nil {
			return err
		}
		if cmd != nil {
			cmd.Println(nc.discoverySummary(discovery))
		}
		return nil
	}

	if runner == nil {
		return errors.New("nextcloud Memories import runner is not configured")
	}
	if err := nc.prepareImport(ctx); err != nil {
		return err
	}
	return runner.Run(cmd, nc)
}

func (nc *Command) runDiscovery(ctx context.Context) (*nextcloud.MemoriesDiscovery, error) {
	discover := nc.discover
	if discover == nil {
		discover = discoverMemories
	}

	discovery, err := discover(ctx, nextcloud.Config{
		BaseURL:       nc.NextcloudURL,
		Username:      nc.NextcloudUser,
		Password:      nc.NextcloudPassword,
		SkipVerifySSL: nc.NextcloudSkipVerifySSL,
		Timeout:       nc.NextcloudClientTimeout,
	})
	if err != nil {
		return nil, err
	}

	if _, err := nc.resolveSelectedRoots(discovery); err != nil {
		return nil, err
	}
	return discovery, nil
}

func (nc *Command) prepareImport(ctx context.Context) error {
	if nc.sourceFS != nil {
		if len(nc.selectedRoots) == 0 {
			nc.selectedRoots = slices.Clone(nc.TimelineRoots)
		}
		if len(nc.selectedRoots) == 0 {
			return errors.New("nextcloud Memories import has no selected timeline roots")
		}
		return nil
	}

	discovery, err := nc.runDiscovery(ctx)
	if err != nil {
		return err
	}
	selectedRoots, err := nc.resolveSelectedRoots(discovery)
	if err != nil {
		return err
	}

	client, err := nextcloud.NewClient(nextcloud.Config{
		BaseURL:       nc.NextcloudURL,
		Username:      nc.NextcloudUser,
		Password:      nc.NextcloudPassword,
		SkipVerifySSL: nc.NextcloudSkipVerifySSL,
		Timeout:       nc.NextcloudClientTimeout,
	})
	if err != nil {
		return err
	}
	if err := client.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect to Nextcloud DAV endpoint: %w", err)
	}

	uid := nc.NextcloudUser
	if discovery.Describe.UID != nil && strings.TrimSpace(*discovery.Describe.UID) != "" {
		uid = strings.TrimSpace(*discovery.Describe.UID)
	}
	newRemoteFS := nc.newRemoteFS
	if newRemoteFS == nil {
		newRemoteFS = func(ctx context.Context, client *nextcloud.Client, uid string) (fs.FS, error) {
			return nextcloud.NewWebDAVFS(ctx, client, uid)
		}
	}
	remoteFS, err := newRemoteFS(ctx, client, uid)
	if err != nil {
		return err
	}

	sourceFS := remoteFS
	if nc.NextcloudLocalDir != "" {
		sourceFS = nextcloud.NewPreferredLocalFS(osfs.DirFS(nc.NextcloudLocalDir), remoteFS)
	}

	nc.discovery = discovery
	nc.client = client
	nc.selectedRoots = selectedRoots
	nc.sourceFS = sourceFS
	return nil
}

func (nc *Command) resolveSelectedRoots(discovery *nextcloud.MemoriesDiscovery) ([]string, error) {
	if len(nc.TimelineRoots) == 0 {
		return slices.Clone(discovery.TimelineRoots), nil
	}

	for _, requestedRoot := range nc.TimelineRoots {
		if !slices.Contains(discovery.TimelineRoots, requestedRoot) {
			return nil, fmt.Errorf("requested --timeline-root %q is not part of the configured Memories timeline roots: %s", requestedRoot, strings.Join(discovery.TimelineRoots, ", "))
		}
	}

	return slices.Clone(nc.TimelineRoots), nil
}

func (nc *Command) DesiredAlbumUsers(ctx context.Context, album assets.Album) ([]adapters.AlbumUser, error) {
	if len(nc.userMappings) == 0 || nc.client == nil {
		return nil, nil
	}
	_, state, err := parseManagedAlbumState(album.Description)
	if err != nil {
		return nil, err
	}
	if state == nil || state.Source != nextcloudMemoriesSourceName || state.AlbumID <= 0 || state.OwnerUID == "" || state.AlbumName == "" {
		return nil, nil
	}

	nc.albumUsersMu.Lock()
	if users, ok := nc.albumUsers[state.AlbumID]; ok {
		cached := slices.Clone(users)
		nc.albumUsersMu.Unlock()
		return cached, nil
	}
	nc.albumUsersMu.Unlock()

	getAlbumCollaborators := nc.getAlbumCollaborators
	if getAlbumCollaborators == nil {
		getAlbumCollaborators = nextcloud.GetAlbumCollaborators
	}
	collaborators, err := getAlbumCollaborators(ctx, nc.client, state.OwnerUID, state.AlbumName)
	if err != nil {
		return nil, err
	}

	desiredUsers := make([]adapters.AlbumUser, 0, len(collaborators))
	seen := make(map[string]struct{}, len(collaborators))
	for _, collaborator := range collaborators {
		if collaborator.Type != nextcloud.AlbumCollaboratorTypeUser {
			continue
		}
		mappedUserID, ok := nc.userMappings[strings.TrimSpace(collaborator.ID)]
		if !ok || mappedUserID == "" {
			if nc.app != nil {
				nc.app.Log().Warn("skipping album collaborator without user mapping", "album", album.Title, "nextcloudUser", collaborator.ID)
			}
			continue
		}
		if _, ok := seen[mappedUserID]; ok {
			continue
		}
		seen[mappedUserID] = struct{}{}
		desiredUsers = append(desiredUsers, adapters.AlbumUser{
			UserID: mappedUserID,
			Role:   albumUserRoleEditor,
		})
	}

	nc.albumUsersMu.Lock()
	if nc.albumUsers == nil {
		nc.albumUsers = map[int][]adapters.AlbumUser{}
	}
	nc.albumUsers[state.AlbumID] = slices.Clone(desiredUsers)
	nc.albumUsersMu.Unlock()

	return desiredUsers, nil
}

func (nc *Command) intentSummary() string {
	mode := "import"
	if nc.DiscoverOnly {
		mode = "discover-only"
	}

	rootScope := "all configured Memories timeline roots"
	if len(nc.TimelineRoots) > 0 {
		rootScope = strings.Join(nc.TimelineRoots, ", ")
	}
	sourceMode := "webdav"
	if nc.NextcloudLocalDir != "" {
		sourceMode = fmt.Sprintf("local-first (%s), fallback webdav", nc.NextcloudLocalDir)
	}

	return strings.Join([]string{
		"Nextcloud Memories scaffold",
		fmt.Sprintf("  mode: %s", mode),
		fmt.Sprintf("  nextcloud-url: %s", nc.NextcloudURL),
		fmt.Sprintf("  nextcloud-user: %s", nc.NextcloudUser),
		fmt.Sprintf("  source-files: %s", sourceMode),
		fmt.Sprintf("  timeline-roots: %s", rootScope),
		fmt.Sprintf("  sync-albums: %t", nc.SyncAlbums),
		fmt.Sprintf("  sync-tags: %t", nc.SyncTags),
		fmt.Sprintf("  require-indexed: %t", nc.RequireIndexed),
		fmt.Sprintf("  tag-album-membership: %t", nc.TagAlbumMembership),
		fmt.Sprintf("  user-maps: %d", len(nc.userMappings)),
		fmt.Sprintf("  skip-verify-ssl: %t", nc.NextcloudSkipVerifySSL),
	}, "\n")
}

func (nc *Command) discoverySummary(discovery *nextcloud.MemoriesDiscovery) string {
	uid := "unknown"
	if discovery.Describe.UID != nil && *discovery.Describe.UID != "" {
		uid = *discovery.Describe.UID
	}

	selectedRoots := discovery.TimelineRoots
	if len(nc.TimelineRoots) > 0 {
		selectedRoots = nc.TimelineRoots
	}

	lines := make([]string, 0, 9+len(discovery.TimelineRoots)+1+len(selectedRoots)+6)
	lines = append(lines,
		"Nextcloud Memories discovery",
		fmt.Sprintf("  nextcloud-base-url: %s", discovery.BaseURL),
		fmt.Sprintf("  dav-root: %s", discovery.DAVRoot),
		fmt.Sprintf("  nextcloud-version: %s", discovery.Capabilities.VersionString),
		fmt.Sprintf("  nextcloud-product: %s", strings.TrimSpace(strings.Join([]string{discovery.Capabilities.ProductName, discovery.Capabilities.Edition}, " "))),
		fmt.Sprintf("  memories-version: %s", discovery.Describe.Version),
		fmt.Sprintf("  authenticated-user: %s", uid),
		fmt.Sprintf("  memories-base-url: %s", discovery.Describe.BaseURL),
		"  configured-timeline-roots:",
	)
	for _, root := range discovery.TimelineRoots {
		lines = append(lines, fmt.Sprintf("    - %s", root))
	}
	lines = append(lines,
		"  selected-timeline-roots:",
	)
	for _, root := range selectedRoots {
		lines = append(lines, fmt.Sprintf("    - %s", root))
	}
	lines = append(lines,
		fmt.Sprintf("  folders-path: %s", discovery.Config.FoldersPath),
		fmt.Sprintf("  albums-enabled: %t", discovery.Config.AlbumsEnabled),
		fmt.Sprintf("  system-tags-enabled: %t", discovery.Config.SystemTagsEnabled),
		fmt.Sprintf("  preview-generator-enabled: %t", discovery.Config.PreviewGeneratorEnabled),
		fmt.Sprintf("  recognize-enabled: %t", discovery.Config.RecognizeEnabled),
		fmt.Sprintf("  facerecognition-enabled: %t", discovery.Config.FaceRecognitionEnabled),
	)

	return strings.Join(lines, "\n")
}

func discoverMemories(ctx context.Context, cfg nextcloud.Config) (*nextcloud.MemoriesDiscovery, error) {
	client, err := nextcloud.NewClient(cfg)
	if err != nil {
		return nil, err
	}
	if err := client.Connect(ctx); err != nil {
		return nil, fmt.Errorf("failed to connect to Nextcloud DAV endpoint: %w", err)
	}
	return nextcloud.DiscoverMemories(ctx, client)
}

func (nc *Command) validate() error {
	var joinedErr error

	nc.NextcloudURL = strings.TrimSpace(nc.NextcloudURL)
	nc.NextcloudUser = strings.TrimSpace(nc.NextcloudUser)
	nc.NextcloudPassword = strings.TrimSpace(nc.NextcloudPassword)
	nc.NextcloudLocalDir = strings.TrimSpace(nc.NextcloudLocalDir)
	nc.TimelineRoots = normalizeTimelineRoots(nc.TimelineRoots)
	nc.UserMaps = normalizeUserMaps(nc.UserMaps)
	nc.userMappings = map[string]string{}
	nc.albumUsers = map[int][]adapters.AlbumUser{}

	if nc.NextcloudURL == "" {
		joinedErr = errors.Join(joinedErr, errors.New("missing the parameter --nextcloud-url, Nextcloud base URL"))
	} else {
		parsedURL, err := url.Parse(nc.NextcloudURL)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			joinedErr = errors.Join(joinedErr, fmt.Errorf("invalid --nextcloud-url %q: must include scheme and host", nc.NextcloudURL))
		} else if parsedURL.User != nil {
			joinedErr = errors.Join(joinedErr, errors.New("invalid --nextcloud-url: must not include credentials; use --nextcloud-user/--nextcloud-password"))
		} else if parsedURL.RawQuery != "" || parsedURL.Fragment != "" {
			joinedErr = errors.Join(joinedErr, errors.New("invalid --nextcloud-url: must not include a query string or fragment"))
		} else {
			nc.NextcloudURL = strings.TrimRight(parsedURL.String(), "/")
		}
	}
	if nc.NextcloudUser == "" {
		joinedErr = errors.Join(joinedErr, errors.New("missing the parameter --nextcloud-user, Nextcloud username"))
	}
	if nc.NextcloudPassword == "" {
		joinedErr = errors.Join(joinedErr, errors.New("missing the parameter --nextcloud-password, Nextcloud password or app password"))
	}
	if nc.NextcloudClientTimeout <= 0 {
		joinedErr = errors.Join(joinedErr, errors.New("invalid --nextcloud-client-timeout: must be greater than 0"))
	}
	if nc.NextcloudLocalDir != "" {
		info, err := os.Stat(nc.NextcloudLocalDir)
		if err != nil {
			joinedErr = errors.Join(joinedErr, fmt.Errorf("invalid --nextcloud-local-dir %q: %w", nc.NextcloudLocalDir, err))
		} else if !info.IsDir() {
			joinedErr = errors.Join(joinedErr, fmt.Errorf("invalid --nextcloud-local-dir %q: not a directory", nc.NextcloudLocalDir))
		}
	}
	for _, root := range nc.TimelineRoots {
		if !strings.HasPrefix(root, "/") {
			joinedErr = errors.Join(joinedErr, fmt.Errorf("invalid --timeline-root %q: must start with /", root))
		}
	}
	for _, userMap := range nc.UserMaps {
		sourceUser, targetUser, err := splitUserMap(userMap)
		if err != nil {
			joinedErr = errors.Join(joinedErr, err)
			continue
		}
		if existingTargetUser, ok := nc.userMappings[sourceUser]; ok && existingTargetUser != targetUser {
			joinedErr = errors.Join(joinedErr, fmt.Errorf("conflicting --user-map for %q: %q and %q", sourceUser, existingTargetUser, targetUser))
			continue
		}
		nc.userMappings[sourceUser] = targetUser
	}

	return joinedErr
}

func normalizeTimelineRoots(roots []string) []string {
	normalized := make([]string, 0, len(roots))
	seen := map[string]struct{}{}

	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		if !strings.HasPrefix(root, "/") {
			root = "/" + root
		}
		root = pathpkg.Clean(root)
		if root == "." || root == "" {
			continue
		}
		if !strings.HasPrefix(root, "/") {
			root = "/" + root
		}
		if _, ok := seen[root]; ok {
			continue
		}
		seen[root] = struct{}{}
		normalized = append(normalized, root)
	}

	return normalized
}

func normalizeUserMaps(values []string) []string {
	normalized := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		normalized = append(normalized, value)
	}
	return normalized
}

func splitUserMap(value string) (string, string, error) {
	sourceUser, targetUser, ok := strings.Cut(value, "=")
	if !ok {
		return "", "", fmt.Errorf("invalid --user-map %q: expected <nextcloud-user>=<immich-user-id>", value)
	}
	sourceUser = strings.TrimSpace(sourceUser)
	targetUser = strings.TrimSpace(targetUser)
	if sourceUser == "" || targetUser == "" {
		return "", "", fmt.Errorf("invalid --user-map %q: expected <nextcloud-user>=<immich-user-id>", value)
	}
	return sourceUser, targetUser, nil
}
