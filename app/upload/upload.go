package upload

import (
	"context"
	"runtime"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/adapters/folder"
	gp "github.com/simulot/immich-go/adapters/googlePhotos"
	"github.com/simulot/immich-go/adapters/shared"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/simulot/immich-go/internal/gen/syncset"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type UpLoadMode int

const (
	UpModeGoogleTakeout UpLoadMode = iota
	UpModeFolder
	UpModeICloud
	UpModePicasa
)

func (m UpLoadMode) String() string {
	switch m {
	case UpModeGoogleTakeout:
		return "Google Takeout"
	case UpModeFolder:
		return "Folder"
	case UpModeICloud:
		return "iCloud"
	case UpModePicasa:
		return "Picasa"
	default:
		return "Unknown"
	}
}

// UploadOptions represents a set of common flags used for filtering assets.
type UploadOptions struct {
	shared.StackOptions

	// TODO place this option at the top
	NoUI bool // Disable UI

	// Add Overwrite flag to UploadOptions
	Overwrite bool // Always overwrite files on the server with local versions

	// Concurrent upload configuration
	ConcurrentUploads int // Number of concurrent upload workers

	// Tags is a list of tags to be added to the imported assets.
	Tags []string `mapstructure:"tag" yaml:"tag" json:"tag" toml:"tag"`

	SessionTag bool   `mapstructure:"session-tag" yaml:"session-tag" json:"session-tag" toml:"session-tag"`
	session    string // Session tag value

	folder.ImportFolderOptions

	// fsyss   []fs.FS
	Filters []filters.Filter
}

func (o *UploadOptions) RegisterFlags(flags *pflag.FlagSet) {
	flags.BoolVar(&o.NoUI, "no-ui", false, "Disable the user interface")
	flags.BoolVar(&o.Overwrite, "overwrite", false, "Always overwrite files on the server with local versions")
	flags.IntVar(&o.ConcurrentUploads, "concurrent-uploads", runtime.NumCPU(), "Number of concurrent upload workers (1-20)")
	flags.StringSliceVar(&o.Tags, "tag", nil, "Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1')")
	flags.BoolVar(&o.SessionTag, "session-tag", false, "Tag uploaded photos with a tag \"{immich-go}/YYYY-MM-DD HH-MM-SS\"")

	o.StackOptions.RegisterFlags(flags)
}

// NewUploadCommand adds the Upload command
func NewUploadCommand(ctx context.Context, app *app.Application) *cobra.Command {
	options := &UploadOptions{}
	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload photos to an Immich server from various sources",
		Args:  cobra.NoArgs, // This command does not accept any arguments, only subcommands

	}
	upCmd := &UpCmd{
		UploadOptions: options,
		app:           app,
		// Mode:              mode,
		localAssets:       syncset.New[string](),
		immichAssetsReady: make(chan struct{}),
	}

	options.RegisterFlags(cmd.PersistentFlags())

	cmd.AddCommand(NewFromFolderCommand(ctx, cmd, app, options))
	cmd.AddCommand(NewFromICloudCommand(ctx, cmd, app, options))
	cmd.AddCommand(NewFromPicasaCommand(ctx, cmd, app, options))
	cmd.AddCommand(gp.NewFromGooglePhotosCommand(ctx, cmd, app, upCmd))
	cmd.AddCommand(NewFromImmichCommand(ctx, cmd, app, options))
	return cmd
}

func (uc *UpCmd) Run(cmd *cobra.Command, adapter adapters.Reader) error {
	// ready to run
	ctx := cmd.Context()
	err := uc.Client.Open(ctx, uc.app)
	if err != nil {
		return nil
	}
	uc.TZ = uc.app.GetTZ()

	// Initialize the Journal
	if uc.app.Jnl() == nil {
		uc.app.SetJnl(fileevent.NewRecorder(uc.app.Log().Logger))
	}

	// Validate concurrent uploads range
	uc.ConcurrentUploads = min(max(uc.ConcurrentUploads, 1), 20)

	// create the adapter for folders
	uc.SupportedMedia = uc.Client.Immich.SupportedMedia()
	uc.Filters = append(uc.Filters, uc.ManageBurst.GroupFilter(), uc.ManageRawJPG.GroupFilter(), uc.ManageHEICJPG.GroupFilter())

	uc.InfoCollector = filenames.NewInfoCollector(uc.TZ, uc.SupportedMedia)

	return uc.upload(ctx, adapter)
}
