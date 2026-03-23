package synology

import (
	"context"
	"fmt"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/app"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// FromSynologyCmd holds the configuration for the from-synology command
type FromSynologyCmd struct {
	// Synology connection settings
	SynologyURL    string
	SynologyUser   string
	SynologyPass   string
	IncludeShared  bool
	InsecureSSL    bool

	// Filters
	Albums        []string
	Tags          []string
	People        []string
	SkipFaceData  bool

	// Internal
	app       *app.Application
	adapter   *Adapter
}

// RegisterFlags registers CLI flags for the command
func (cmd *FromSynologyCmd) RegisterFlags(flags *pflag.FlagSet) {
	// Connection flags
	flags.StringVar(&cmd.SynologyURL, "synology-url", "", "Synology Photos URL (e.g., https://nas:5001 or https://nas/photo)")
	flags.StringVar(&cmd.SynologyUser, "synology-user", "", "Synology username")
	flags.StringVar(&cmd.SynologyPass, "synology-pass", "", "Synology password")
	flags.BoolVar(&cmd.IncludeShared, "include-shared-space", false, "Include shared space (FotoTeam)")
	flags.BoolVar(&cmd.InsecureSSL, "insecure-ssl", true, "Skip SSL certificate verification")

	// Filter flags
	flags.StringSliceVar(&cmd.Albums, "from-albums", nil, "Import only from these albums (comma-separated)")
	flags.StringSliceVar(&cmd.Tags, "from-tags", nil, "Import only items with these tags (comma-separated)")
	flags.StringSliceVar(&cmd.People, "from-people", nil, "Import only items with these people (comma-separated)")
	flags.BoolVar(&cmd.SkipFaceData, "skip-face-data", false, "Skip face recognition data")
}

// NewFromSynologyCommand creates a new Cobra command for importing from Synology Photos
func NewFromSynologyCommand(ctx context.Context, parent *cobra.Command, app *app.Application, runner adapters.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-synology [flags]",
		Short: "Import photos from Synology Photos",
		Long: `Import photos, videos, albums, and tags from a Synology Photos server.

This command connects to a Synology NAS running Synology Photos and imports
your media library into Immich. It supports:

- Photos and videos from Personal Space
- Albums (normal albums, not conditional albums)
- Tags/labels
- Face recognition data (stored as tags)
- GPS coordinates and descriptions

Examples:
  # Import all photos from Synology
  immich-go upload from-synology --server=http://immich:2283 --api-key=xxx \\
    --synology-url=https://nas:5001 --synology-user=admin --synology-pass=secret

  # Import specific albums only
  immich-go upload from-synology --server=http://immich:2283 --api-key=xxx \\
    --synology-url=https://nas:5001 --synology-user=admin --synology-pass=secret \\
    --from-albums="Vacation 2023,Birthday Party"

  # Import items with specific tags
  immich-go upload from-synology --server=http://immich:2283 --api-key=xxx \\
    --synology-url=https://nas:5001 --synology-user=admin --synology-pass=secret \\
    --from-tags="family,vacation"

  # Import without face recognition data
  immich-go upload from-synology --server=http://immich:2283 --api-key=xxx \\
    --synology-url=https://nas:5001 --synology-user=admin --synology-pass=secret \\
    --skip-face-data`,
		Args: cobra.NoArgs,
	}

	cmd.SetContext(ctx)
	fsc := &FromSynologyCmd{}
	fsc.RegisterFlags(cmd.Flags())

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return fsc.Run(ctx, cmd, app, runner)
	}

	return cmd
}

// Run executes the from-synology command
func (fsc *FromSynologyCmd) Run(ctx context.Context, cmd *cobra.Command, app *app.Application, runner adapters.Runner) error {
	// Validate required flags
	if fsc.SynologyURL == "" {
		return fmt.Errorf("--synology-url is required")
	}
	if fsc.SynologyUser == "" {
		return fmt.Errorf("--synology-user is required")
	}
	if fsc.SynologyPass == "" {
		return fmt.Errorf("--synology-pass is required")
	}

	fsc.app = app

	// Create and configure the adapter
	adapter := NewAdapter(app, fsc.SynologyURL, fsc.SynologyUser, fsc.SynologyPass)
	adapter.IncludeShared = fsc.IncludeShared
	adapter.Albums = fsc.Albums
	adapter.Tags = fsc.Tags
	adapter.People = fsc.People
	adapter.SkipFaceData = fsc.SkipFaceData
	fsc.adapter = adapter

	// Open connection to Synology
	app.Log().Info("Connecting to Synology Photos", "url", fsc.SynologyURL)
	if err := adapter.Open(ctx); err != nil {
		return fmt.Errorf("failed to connect to Synology Photos: %w", err)
	}
	defer adapter.Close(ctx)

	app.Log().Info("Successfully connected to Synology Photos")

	// Run the import
	return runner.Run(cmd, adapter)
}
