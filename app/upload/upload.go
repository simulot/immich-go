package upload

import (
	"context"
	"fmt"
	"time"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/adapters/folder"
	"github.com/simulot/immich-go/adapters/fromimmich"
	gp "github.com/simulot/immich-go/adapters/googlePhotos"
	"github.com/simulot/immich-go/adapters/shared"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assets/cache"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/simulot/immich-go/internal/gen/syncset"
	"github.com/simulot/immich-go/internal/groups/burst"
	"github.com/simulot/immich-go/internal/groups/epsonfastfoto"
	"github.com/simulot/immich-go/internal/groups/series"
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

type UpCmd struct {
	// Cli flags

	shared.StackOptions
	client     app.Client
	NoUI       bool // Disable UI
	Overwrite  bool // Always overwrite files on the server with local versions
	Tags       []string
	SessionTag bool
	session    string // Session tag value

	// Upload command state
	Filters           []filters.Filter
	TZ                *time.Location
	Mode              UpLoadMode
	app               *app.Application
	assetIndex        *immichIndex                         // List of assets present on the server
	localAssets       *syncset.Set[string]                 // List of assets present on the local input by name+size
	immichAssetsReady chan struct{}                        // Signal that the asset index is ready
	deleteServerList  []*immich.Asset                      // List of server assets to remove
	adapter           adapters.Reader                      // the source of assets
	DebugCounters     bool                                 // Enable CSV action counters per file
	albumsCache       *cache.CollectionCache[assets.Album] // List of albums present on the server
	tagsCache         *cache.CollectionCache[assets.Tag]   // List of tags present on the server
	finished          bool                                 // the finish task has been run
	SupportedMedia    filetypes.SupportedMedia             // List of supported media types
	InfoCollector     *filenames.InfoCollector             // Collects information about the files being processed
}

func (uc *UpCmd) RegisterFlags(flags *pflag.FlagSet) {
	uc.client.RegisterFlags(flags, "")
	flags.BoolVar(&uc.NoUI, "no-ui", false, "Disable the user interface")
	flags.BoolVar(&uc.Overwrite, "overwrite", false, "Always overwrite files on the server with local versions")
	flags.StringSliceVar(&uc.Tags, "tag", nil, "Add tags to the imported assets. Can be specified multiple times. Hierarchy is supported using a / separator (e.g. 'tag1/subtag1')")
	flags.BoolVar(&uc.SessionTag, "session-tag", false, "Tag uploaded photos with a tag \"{immich-go}/YYYY-MM-DD HH-MM-SS\"")

	uc.StackOptions.RegisterFlags(flags)
}

// NewUploadCommand creates the root "upload" command and adds subcommands for each supported source.
// It registers flags and initializes the UpCmd struct, which holds the state for uploads.
func NewUploadCommand(ctx context.Context, app *app.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "upload [flags]",
		Short: "Upload photos to an Immich server from various sources",
		Args:  cobra.NoArgs, // This command does not accept any arguments, only subcommands
	}
	uc := &UpCmd{
		app:               app,
		localAssets:       syncset.New[string](),
		immichAssetsReady: make(chan struct{}),
	}

	// Register CLI flags for the upload command
	uc.RegisterFlags(cmd.PersistentFlags())

	// Add subcommands for each supported upload source
	cmd.AddCommand(folder.NewFromFolderCommand(ctx, cmd, app, uc))
	cmd.AddCommand(folder.NewFromICloudCommand(ctx, cmd, app, uc))
	cmd.AddCommand(folder.NewFromPicasaCommand(ctx, cmd, app, uc))
	cmd.AddCommand(gp.NewFromGooglePhotosCommand(ctx, cmd, app, uc))
	cmd.AddCommand(fromimmich.NewFromImmichCommand(ctx, cmd, app, uc))

	cmd.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		// Initialize the Journal
		if app.Jnl() == nil {
			app.SetJnl(fileevent.NewRecorder(app.Log().Logger))
		}
		app.SetTZ(time.Local)
		if tz, err := cmd.Flags().GetString("time-zone"); err == nil && tz != "" {
			if loc, err := time.LoadLocation(tz); err == nil {
				app.SetTZ(loc)
			}
		} else {
			return err
		}
		return nil
	}

	return cmd
}

// Run is called back by the actual asset reader
func (uc *UpCmd) Run(cmd *cobra.Command, adapter adapters.Reader) error {
	uc.Mode = UpModeFolder // TODO

	// ready to run
	ctx := cmd.Context()
	err := uc.client.Open(ctx, uc.app)
	if err != nil {
		return nil
	}
	uc.TZ = uc.app.GetTZ()

	// Initialize the Journal
	if uc.app.Jnl() == nil {
		uc.app.SetJnl(fileevent.NewRecorder(uc.app.Log().Logger))
	}

	if uc.SessionTag {
		uc.session = fmt.Sprintf("{immich-go}/%s", time.Now().Format("2006-01-02 15:04:05"))
	}

	if uc.ManageEpsonFastFoto {
		g := epsonfastfoto.Group{}
		uc.Groupers = append(uc.Groupers, g.Group)
	}
	if uc.ManageBurst != filters.BurstNothing {
		uc.Groupers = append(uc.Groupers, burst.Group)
	}
	uc.Groupers = append(uc.Groupers, series.Group)

	// create the adapter for folders
	uc.SupportedMedia = uc.client.Immich.SupportedMedia()
	uc.Filters = append(uc.Filters, uc.ManageBurst.GroupFilter(), uc.ManageRawJPG.GroupFilter(), uc.ManageHEICJPG.GroupFilter())

	uc.InfoCollector = filenames.NewInfoCollector(uc.TZ, uc.SupportedMedia)

	return uc.upload(ctx, adapter)
}
