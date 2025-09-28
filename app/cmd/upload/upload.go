package upload

import (
	"context"
	"fmt"
	"runtime"
	"time"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/configuration"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filters"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
	// TODO place this option at the top
	NoUI bool // Disable UI

	// Add Overwrite flag to UploadOptions
	Overwrite bool // Always overwrite files on the server with local versions

	// Concurrent upload configuration
	ConcurrentUploads int // Number of concurrent upload workers

	Filters []filters.Filter
}

// NewUploadCommand adds the Upload command
func NewUploadCommand(ctx context.Context, a *app.Application) *cobra.Command {
	options := &UploadOptions{}
	cmd := &cobra.Command{
		Use:   "upload",
		Short: "Upload photos to an Immich server from various sources",
		Args:  cobra.NoArgs, // This command does not accept any arguments, only subcommands
	}

	app.AddClientFlags(ctx, cmd, a, false)
	cmd.TraverseChildren = true

	// Add upload-specific flags
	cmd.PersistentFlags().BoolVar(&options.NoUI, "no-ui", false, "Disable the user interface")
	cmd.PersistentFlags().BoolVar(&options.Overwrite, "overwrite", false, "Always overwrite files on the server with local versions")
	cmd.PersistentFlags().IntVar(&options.ConcurrentUploads, "concurrent-uploads", runtime.NumCPU(), "Number of concurrent upload workers (1-20)")

	// Bind upload flags to Viper
	_ = viper.BindPFlag("ui.no_ui", cmd.PersistentFlags().Lookup("no-ui"))
	_ = viper.BindPFlag("upload.overwrite", cmd.PersistentFlags().Lookup("overwrite"))
	_ = viper.BindPFlag("upload.concurrent_uploads", cmd.PersistentFlags().Lookup("concurrent-uploads"))

	cmd.PersistentPreRunE = app.ChainRunEFunctions(cmd.PersistentPreRunE, options.LoadConfiguration, ctx, cmd, a)
	cmd.PersistentPreRunE = app.ChainRunEFunctions(cmd.PersistentPreRunE, options.Open, ctx, cmd, a)

	cmd.AddCommand(NewFromFolderCommand(ctx, cmd, a, options))
	cmd.AddCommand(NewFromICloudCommand(ctx, cmd, a, options))
	cmd.AddCommand(NewFromPicasaCommand(ctx, cmd, a, options))
	cmd.AddCommand(NewFromGooglePhotosCommand(ctx, cmd, a, options))
	cmd.AddCommand(NewFromImmichCommand(ctx, cmd, a, options))
	return cmd
}

func (options *UploadOptions) LoadConfiguration(ctx context.Context, cmd *cobra.Command, app *app.Application) error {
	// Load configuration values from Viper into upload options
	config, err := configuration.GetConfiguration()
	if err != nil {
		return err
	}

	var uploadConfigSources []string

	// Apply configuration values (only if not set by flags)
	if !cmd.PersistentFlags().Changed("no-ui") {
		options.NoUI = config.UI.NoUI
		if config.UI.NoUI {
			uploadConfigSources = append(uploadConfigSources, "no-ui=true from config")
		}
	} else {
		uploadConfigSources = append(uploadConfigSources, "no-ui from CLI flag")
	}

	if !cmd.PersistentFlags().Changed("overwrite") {
		options.Overwrite = config.Upload.Overwrite
		if config.Upload.Overwrite {
			uploadConfigSources = append(uploadConfigSources, "overwrite=true from config")
		}
	} else {
		uploadConfigSources = append(uploadConfigSources, "overwrite from CLI flag")
	}

	if !cmd.PersistentFlags().Changed("concurrent-uploads") {
		options.ConcurrentUploads = config.Upload.ConcurrentUploads
		if config.Upload.ConcurrentUploads > 0 {
			uploadConfigSources = append(uploadConfigSources, fmt.Sprintf("concurrent-uploads=%d from config", config.Upload.ConcurrentUploads))
		}
	} else {
		uploadConfigSources = append(uploadConfigSources, "concurrent-uploads from CLI flag")
	}

	// Log upload-specific configuration
	if len(uploadConfigSources) > 0 {
		app.Log().Message("Upload configuration sources:")
		for _, source := range uploadConfigSources {
			app.Log().Message("  - %s", source)
		}
	}

	return nil
}

func (options *UploadOptions) Open(ctx context.Context, cmd *cobra.Command, app *app.Application) error {
	// Initialize the Journal
	if app.Jnl() == nil {
		app.SetJnl(fileevent.NewRecorder(app.Log().Logger))
	}
	app.SetTZ(time.Local)
	if tz, err := cmd.Flags().GetString("time-zone"); err == nil && tz != "" {
		if loc, err := time.LoadLocation(tz); err == nil {
			app.SetTZ(loc)
		}
	}

	// Validate concurrent uploads range
	options.ConcurrentUploads = min(max(options.ConcurrentUploads, 1), 20)
	return nil
}
