package folder

import (
	"context"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"github.com/simulot/immich-go/adapters"
	"github.com/simulot/immich-go/adapters/shared"
	"github.com/simulot/immich-go/app"
	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/filenames"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/simulot/immich-go/internal/gen"
	"github.com/simulot/immich-go/internal/groups"
	"github.com/simulot/immich-go/internal/namematcher"
	"github.com/simulot/immich-go/internal/worker"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// ImportFolderCmd represents the flags used for importing assets from a file system.
type ImportFolderCmd struct {
	// CLI flags
	UsePathAsAlbumName     AlbumFolderMode
	AlbumNamePathSeparator string
	ImportIntoAlbum        string
	BannedFiles            namematcher.List
	Recursive              bool
	InclusionFlags         cliflags.InclusionFlags
	IgnoreSideCarFiles     bool
	FolderAsTags           bool
	TakeDateFromFilename   bool
	PicasaAlbum            bool
	ICloudTakeout          bool
	ICloudMemoriesAsAlbums bool
	shared.StackOptions

	// Internal fields
	app                     *app.Application
	processor               *fileprocessor.FileProcessor
	fsyss                   []fs.FS
	tz                      *time.Location
	supportedMedia          filetypes.SupportedMedia
	infoCollector           *filenames.InfoCollector
	pool                    *worker.Pool
	wg                      sync.WaitGroup
	groupers                []groups.Grouper
	requiresDateInformation bool                              // true if we need to read the date from the file for the options
	picasaAlbums            *gen.SyncMap[string, PicasaAlbum] // ap[string]PicasaAlbum
	icloudMetas             *gen.SyncMap[string, iCloudMeta]
	icloudMetaPass          bool

	// Pre-scan date tracking
	monthsMu       sync.Mutex
	activeMonthSet map[string]struct{} // set of YYYY-MM strings discovered during pre-scan
	activeMonths   []string            // sorted list of YYYY-MM strings (populated after pre-scan)
	hasNoDateFiles bool                // whether files with no determinable date were found

	// Month filter for BrowseMonth — when set, parseDir skips files
	// not matching this month BEFORE extracting from zip.
	targetMonth  string    // "YYYY-MM", "no-date", or "" (no filter)
	targetAfter  time.Time // inclusive lower bound
	targetBefore time.Time // exclusive upper bound
}

// addMonth extracts the YYYY-MM string from a time.Time and adds it to the
// active months set. It is safe for concurrent use.
func (ifc *ImportFolderCmd) addMonth(t time.Time) {
	if t.IsZero() {
		return
	}
	month := fmt.Sprintf("%04d-%02d", t.Year(), t.Month())
	ifc.monthsMu.Lock()
	defer ifc.monthsMu.Unlock()
	if ifc.activeMonthSet == nil {
		ifc.activeMonthSet = make(map[string]struct{})
	}
	ifc.activeMonthSet[month] = struct{}{}
}

// setNoDateFiles marks that at least one file with no determinable date was found.
func (ifc *ImportFolderCmd) setNoDateFiles() {
	ifc.monthsMu.Lock()
	defer ifc.monthsMu.Unlock()
	ifc.hasNoDateFiles = true
}

// buildSortedMonths converts the activeMonthSet into a sorted slice and stores it in activeMonths.
func (ifc *ImportFolderCmd) buildSortedMonths() {
	ifc.monthsMu.Lock()
	defer ifc.monthsMu.Unlock()
	ifc.activeMonths = make([]string, 0, len(ifc.activeMonthSet))
	for m := range ifc.activeMonthSet {
		ifc.activeMonths = append(ifc.activeMonths, m)
	}
	sort.Strings(ifc.activeMonths)
}

// ActiveMonths returns the sorted list of YYYY-MM strings discovered during pre-scan.
func (ifc *ImportFolderCmd) ActiveMonths() []string {
	ifc.monthsMu.Lock()
	defer ifc.monthsMu.Unlock()
	result := make([]string, len(ifc.activeMonths))
	copy(result, ifc.activeMonths)
	return result
}

// HasNoDateFiles returns whether files with no determinable date were found during pre-scan.
func (ifc *ImportFolderCmd) HasNoDateFiles() bool {
	ifc.monthsMu.Lock()
	defer ifc.monthsMu.Unlock()
	return ifc.hasNoDateFiles
}

// PreScan runs the iCloud CSV first pass (if applicable) and walks local files
// to extract dates from filenames using InfoCollector. It returns a sorted unique
// list of YYYY-MM strings representing months that have assets.
func (ifc *ImportFolderCmd) PreScan(ctx context.Context) ([]string, error) {
	// Initialize month set
	ifc.monthsMu.Lock()
	ifc.activeMonthSet = make(map[string]struct{})
	ifc.hasNoDateFiles = false
	ifc.monthsMu.Unlock()

	// For iCloud takeouts, run the CSV meta pass first
	if ifc.ICloudTakeout && ifc.icloudMetas != nil {
		// Iterate over already-parsed iCloud metas to collect dates
		ifc.icloudMetas.Range(func(_ string, meta iCloudMeta) bool {
			if !meta.originalCreationDate.IsZero() {
				ifc.addMonth(meta.originalCreationDate)
			}
			return true
		})
	}

	// Walk local files to extract dates from filenames
	if ifc.infoCollector == nil {
		ifc.infoCollector = filenames.NewInfoCollector(ifc.tz, ifc.supportedMedia)
	}

	for _, fsys := range ifc.fsyss {
		err := ifc.preScanDir(ctx, fsys, ".")
		if err != nil {
			return nil, err
		}
	}

	ifc.buildSortedMonths()
	return ifc.ActiveMonths(), nil
}

// preScanDir walks a directory to extract dates from filenames for pre-scan.
func (ifc *ImportFolderCmd) preScanDir(ctx context.Context, fsys fs.FS, dir string) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return err
	}

	fsName := ""
	if named, ok := fsys.(interface{ Name() string }); ok {
		fsName = named.Name()
	}

	for _, entry := range entries {
		if entry.IsDir() {
			if ifc.Recursive && entry.Name() != "." {
				subDir := dir + "/" + entry.Name()
				if dir == "." {
					subDir = entry.Name()
				}
				if err := ifc.preScanDir(ctx, fsys, subDir); err != nil {
					return err
				}
			}
			continue
		}

		base := entry.Name()
		ext := filepath.Ext(base)

		// Skip CSV and non-media files
		if ext == icloudMetadataExt {
			continue
		}
		mediaType := ifc.supportedMedia.TypeFromExt(ext)
		if mediaType != filetypes.TypeImage && mediaType != filetypes.TypeVideo {
			continue
		}

		// Build a full path for the info collector
		name := dir + "/" + base
		if dir == "." {
			name = base
		}
		n := name
		if fsName != "" {
			n = fsName + "/" + n
		}

		info := ifc.infoCollector.GetInfo(n)
		if !info.Taken.IsZero() {
			ifc.addMonth(info.Taken)
		} else {
			// Try file modification time as fallback
			fi, err := fs.Stat(fsys, name)
			if err == nil && !fi.ModTime().IsZero() {
				ifc.addMonth(fi.ModTime())
			} else {
				ifc.setNoDateFiles()
			}
		}
	}
	return nil
}

func (ifc *ImportFolderCmd) RegisterFlags(flags *pflag.FlagSet, cmd *cobra.Command) {
	ifc.Recursive = true
	ifc.supportedMedia = filetypes.DefaultSupportedMedia
	ifc.UsePathAsAlbumName = FolderModeNone
	ifc.BannedFiles, _ = namematcher.New(shared.DefaultBannedFiles...)

	flags.Var(&ifc.BannedFiles, "ban-file", "Exclude a file based on a pattern (case-insensitive). Can be specified multiple times.")
	flags.StringVar(&ifc.ImportIntoAlbum, "into-album", "", "Specify an album to import all files into")
	flags.Var(&ifc.UsePathAsAlbumName, "folder-as-album", "Import all files in albums defined by the folder structure. Can be set to 'FOLDER' to use the folder name as the album name, or 'PATH' to use the full path as the album name")
	flags.StringVar(&ifc.AlbumNamePathSeparator, "album-path-joiner", " / ", "Specify a string to use when joining multiple folder names to create an album name (e.g. ' ',' - ')")
	flags.BoolVar(&ifc.Recursive, "recursive", true, "Explore the folder and all its sub-folders")
	flags.BoolVar(&ifc.IgnoreSideCarFiles, "ignore-sidecar-files", false, "Don't upload sidecar with the photo.")
	flags.BoolVar(&ifc.FolderAsTags, "folder-as-tags", false, "Use the folder structure as tags, (ex: the file  holiday/summer 2024/file.jpg will have the tag holiday/summer 2024)")
	flags.BoolVar(&ifc.TakeDateFromFilename, "date-from-name", true, "Use the date from the filename if the date isn't available in the metadata (Only for jpg, mp4, heic, dng, cr2, cr3, arw, raf, nef, mov)")

	if cmd.Parent() != nil && cmd.Parent().Name() == "upload" {
		ifc.StackOptions.RegisterFlags(flags)
	}

	ifc.InclusionFlags.RegisterFlags(flags, "") // selection per extension
	ifc.ICloudTakeout = false
	ifc.PicasaAlbum = false
	switch cmd.Name() {
	case "from-picasa":
		flags.BoolVar(&ifc.PicasaAlbum, "album-picasa", true, "Use Picasa album name found in .picasa.ini file")
	case "from-icloud":
		ifc.ICloudTakeout = true
		ifc.PicasaAlbum = false
		cmd.Flags().BoolVar(&ifc.ICloudMemoriesAsAlbums, "memories", false, "Import icloud memories as albums")
	}
}

func NewFromFolderCommand(ctx context.Context, parent *cobra.Command, app *app.Application, runner adapters.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-folder [flags] <path>...",
		Short: "Upload photos from a folder",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.SetContext(ctx)
	flags := cmd.Flags()
	o := ImportFolderCmd{}
	o.RegisterFlags(flags, cmd)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return o.run(cmd, args, app, runner)
	}

	return cmd
}

func NewFromICloudCommand(ctx context.Context, parent *cobra.Command, app *app.Application, runner adapters.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-icloud [flags] <path>...",
		Short: "Upload photos from an iCloud takeout folder or zip file",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.SetContext(ctx)
	flags := cmd.Flags()
	o := ImportFolderCmd{}
	o.RegisterFlags(flags, cmd)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return o.run(cmd, args, app, runner)
	}
	return cmd
}

func NewFromPicasaCommand(ctx context.Context, parent *cobra.Command, app *app.Application, runner adapters.Runner) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "from-picasa [flags] <path>...",
		Short: "Upload photos from a Picasa folder or zip file",
		Args:  cobra.MinimumNArgs(1),
	}
	cmd.SetContext(ctx)
	flags := cmd.Flags()
	o := ImportFolderCmd{}
	o.RegisterFlags(flags, cmd)
	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		return o.run(cmd, args, app, runner)
	}
	return cmd
}
