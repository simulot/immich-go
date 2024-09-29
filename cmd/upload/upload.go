// Command Upload

package upload

import (
	"errors"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/adapters/folder"
	gp "github.com/simulot/immich-go/adapters/googlePhotos"
	"github.com/simulot/immich-go/cmd"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/spf13/cobra"
)

type UpCmd struct {
	UploadCmd         *cobra.Command         // The import command
	Jnl               *fileevent.Recorder    // File event recorder
	Root              *cmd.RootImmichFlags   // global flags
	Server            *cmd.ImmichServerFlags // server flags attached to the import command
	*CommonFlags                             // Common flags between import sub-commands
	UploadFolderFlags *folder.ImportFlags    // Folder import flags
	GooglePhotosFlags *gp.ImportFlags        // Google Photos import flags

	AssetIndex       *AssetIndex     // List of assets present on the server
	deleteServerList []*immich.Asset // List of server assets to remove
	// deleteLocalList  []*adapters.LocalAssetFile // List of local assets to remove
	// stacks        *stacking.StackBuilder
	browser       adapters.Adapter
	DebugCounters bool // Enable CSV action counters per file

	// fsyss  []fs.FS                            // pseudo file system to browse
	Paths  []string                          // Path to explore
	albums map[string]immich.AlbumSimplified // Albums by title
}

func AddCommand(root *cmd.RootImmichFlags) {
	upCommand := &cobra.Command{
		Use:   "upload",
		Short: "upload photos and videos on the immich sever",
	}

	upCommand.RunE = func(cmd *cobra.Command, args []string) error {
		return errors.New("the upload command need a valid sub command")
	}
	root.Command.AddCommand(upCommand)
	addFromFolderCommand(upCommand, root)
	addFromGooglePhotosCommand(upCommand, root)
}

/*

func UploadCommand(ctx context.Context, common *cmd.RootImmichFlags, args []string) error {
	app, err := newCommand(ctx, common, args, nil)
	if err != nil {
		return err
	}
	if len(app.fsyss) == 0 {
		return nil
	}
	return app.run(ctx)
}

type fsOpener func() ([]fs.FS, error)

func newCommand(ctx context.Context, common *cmd.RootImmichFlags, args []string, fsOpener fsOpener) (*UpCmd, error) {
	var err error
	cmd := flag.NewFlagSet("upload", flag.ExitOnError)

	app := UpCmd{
		RootImmichFlags: common,
	}
	app.BannedFiles, err = namematcher.New(
		`@eaDir/`,
		`@__thumb/`,          // QNAP
		`SYNOFILE_THUMB_*.*`, // SYNOLOGY
		`Lightroom Catalog/`, // LR
		`thumbnails/`,        // Android photo
		`.DS_Store/`,         // Mac OS custom attributes
	)
	if err != nil {
		return nil, err
	}

	// app.RootImmichFlags.SetFlags(cmd)
	cmd.BoolFunc(
		"dry-run",
		"display actions but don't touch source or destination",
		myflag.BoolFlagFn(&app.DryRun, false))
	cmd.Var(&app.DateRange,
		"date",
		"Date of capture range.")
	cmd.StringVar(&app.ImportIntoAlbum,
		"album",
		"",
		"All assets will be added to this album.")
	cmd.BoolFunc(
		"create-album-folder",
		" folder import only: Create albums for assets based on the parent folder",
		myflag.BoolFlagFn(&app.CreateAlbumAfterFolder, false))
	cmd.BoolFunc(
		"use-full-path-album-name",
		" folder import only: Use the full path towards the asset for determining the Album name",
		myflag.BoolFlagFn(&app.UseFullPathAsAlbumName, false))
	cmd.StringVar(&app.AlbumNamePathSeparator,
		"album-name-path-separator",
		" ",
		" when use-full-path-album-name = true, determines how multiple (sub) folders, if any, will be joined")
	cmd.BoolFunc(
		"google-photos",
		"Import GooglePhotos takeout zip files",
		myflag.BoolFlagFn(&app.GooglePhotos, false))
	cmd.BoolFunc(
		"create-albums",
		" google-photos only: Create albums like there were in the source (default: TRUE)",
		myflag.BoolFlagFn(&app.CreateAlbums, true))
	cmd.StringVar(&app.PartnerAlbum,
		"partner-album",
		"",
		" google-photos only: Assets from partner will be added to this album. (ImportIntoAlbum, must already exist)")
	cmd.BoolFunc(
		"keep-partner",
		" google-photos only: Import also partner's items (default: TRUE)", myflag.BoolFlagFn(&app.KeepPartner, true))
	cmd.StringVar(&app.ImportFromAlbum,
		"from-album",
		"",
		" google-photos only: Import only from this album")

	cmd.BoolFunc(
		"keep-untitled-albums",
		" google-photos only: Keep Untitled albums and imports their contain (default: FALSE)", myflag.BoolFlagFn(&app.KeepUntitled, false))

	cmd.BoolFunc(
		"use-album-folder-as-name",
		" google-photos only: Use folder name and ignore albums' title (default:FALSE)", myflag.BoolFlagFn(&app.UseFolderAsAlbumName, false))

	cmd.BoolFunc(
		"discard-archived",
		" google-photos only: Do not import archived photos (default FALSE)", myflag.BoolFlagFn(&app.DiscardArchived, false))

	cmd.BoolFunc(
		"auto-archive",
		" google-photos only: Automatically archive photos that are also archived in google photos (default TRUE)", myflag.BoolFlagFn(&app.AutoArchive, true))

	cmd.BoolFunc(
		"create-stacks",
		"Stack jpg/raw or bursts  (default FALSE)", myflag.BoolFlagFn(&app.CreateStacks, false))

	cmd.BoolFunc(
		"stack-jpg-raw",
		"Control the stacking of jpg/raw photos (default TRUE)", myflag.BoolFlagFn(&app.StackJpgRaws, false))
	cmd.BoolFunc(
		"stack-burst",
		"Control the stacking bursts (default TRUE)", myflag.BoolFlagFn(&app.StackBurst, false))

	// cmd.BoolVar(&app.Delete, "delete", false, "Delete local assets after upload")

	cmd.Var(&app.BrowserConfig.SelectExtensions, "select-types", "list of selected extensions separated by a comma")
	cmd.Var(&app.BrowserConfig.ExcludeExtensions, "exclude-types", "list of excluded extensions separated by a comma")

	cmd.StringVar(&app.WhenNoDate,
		"when-no-date",
		"FILE",
		" When the date of take can't be determined, use the FILE's date or the current time NOW. (default: FILE)")

	cmd.Var(&app.BannedFiles, "exclude-files", "Ignore files based on a pattern. Case insensitive. Add one option for each pattern do you need.")

	cmd.BoolVar(&app.ForceUploadWhenNoJSON, "upload-when-missing-JSON", app.ForceUploadWhenNoJSON, "when true, photos are upload even without associated JSON file.")
	cmd.BoolVar(&app.DebugFileList, "debug-file-list", app.DebugFileList, "Check how the your file list would be processed")

	err = cmd.Parse(args)
	if err != nil {
		return nil, err
	}

	if app.DebugFileList {
		if len(cmd.Args()) < 2 {
			return nil, fmt.Errorf("the option -debug-file-list requires a file name and a date format")
		}
		app.LogFile = strings.TrimSuffix(cmd.Arg(0), filepath.Ext(cmd.Arg(0))) + ".log"
		_ = os.Remove(app.LogFile)

		fsOpener = func() ([]fs.FS, error) {
			return fakefs.ScanFileList(cmd.Arg(0), cmd.Arg(1))
		}
	} else {
	}

	app.WhenNoDate = strings.ToUpper(app.WhenNoDate)
	switch app.WhenNoDate {
	case "FILE", "NOW":
	default:
		return nil, fmt.Errorf("the -when-no-date accepts FILE or NOW")
	}

	app.BrowserConfig.Validate()
	err = app.RootImmichFlags.Start(ctx)
	if err != nil {
		return nil, err
	}

	if fsOpener == nil {
		fsOpener = func() ([]fs.FS, error) {
			return fshelper.ParsePath(cmd.Args())
		}
	}
	app.fsyss, err = fsOpener()
	if err != nil {
		return nil, err
	}
	if len(app.fsyss) == 0 {
		fmt.Println("No file found matching the pattern: ", strings.Join(cmd.Args(), ","))
		app.Log.Info("No file found matching the pattern: " + strings.Join(cmd.Args(), ","))
	}
	return &app, nil
}
*/
