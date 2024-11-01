// Tag uploaded assets
package tag

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/simulot/immich-go/browser"
	"github.com/simulot/immich-go/browser/files"
	"github.com/simulot/immich-go/cmd"
	"github.com/simulot/immich-go/helpers/asset"
	"github.com/simulot/immich-go/helpers/datatype"
	"github.com/simulot/immich-go/helpers/fileevent"
	"github.com/simulot/immich-go/helpers/fshelper"
	"github.com/simulot/immich-go/helpers/myflag"
	"github.com/simulot/immich-go/helpers/namematcher"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/fakefs"
)

type TagCmd struct {
	*cmd.SharedFlags // shared flags and immich client

	fsyss []fs.FS // pseudo file system to browse

	DryRun         bool                // Display actions but don't change anything
	BannedFiles    namematcher.List    // List of banned file name patterns
	Tags           datatype.StringList // List of tags to apply to assets. Can use forwards slashes to create tag hierarchy (ex. "Holiday/Groundhog's Day")
	TagWithSession bool                // Tag uploaded assets according to the format immich-go/YYYY-MM-DD/HH-MI-SS
	TagWithPath    bool                // Hierarchically tag uploaded assets using path to assets
	RemoveTags     datatype.StringList // List of tags to remove from assets. If list is empty all tags are removed.
	TagCleanup     bool                // Trigger job to delete unused tags.

	BrowserConfig asset.Configuration

	AssetIndex       *asset.AssetIndex // List of assets present on the server
	browser          browser.Browser
	uploadedAssetIDs []string
	tagToAssetIDs    map[string][]string
	sessionTag       string
}

func TagCommand(ctx context.Context, common *cmd.SharedFlags, args []string) error {
	app, err := newCommand(ctx, common, args, nil)
	if err != nil {
		return err
	}
	return app.run(ctx)
}

type fsOpener func() ([]fs.FS, error)

func newCommand(
	ctx context.Context,
	common *cmd.SharedFlags,
	args []string,
	fsOpener fsOpener,
) (*TagCmd, error) {
	var err error
	cmd := flag.NewFlagSet("tag", flag.ExitOnError)

	app := TagCmd{
		SharedFlags: common,
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

	app.SharedFlags.SetFlags(cmd)
	cmd.BoolFunc(
		"dry-run",
		"display actions but don't touch source or destination",
		myflag.BoolFlagFn(&app.DryRun, false))
	cmd.Var(
		&app.BrowserConfig.SelectExtensions,
		"select-types",
		"list of selected extensions separated by a comma",
	)
	cmd.Var(
		&app.BrowserConfig.ExcludeExtensions,
		"exclude-types",
		"list of excluded extensions separated by a comma",
	)
	cmd.Var(
		&app.BannedFiles,
		"exclude-files",
		"Ignore files based on a pattern. Case insensitive. Add one option for each pattern do you need.",
	)
	cmd.BoolVar(
		&app.DebugFileList,
		"debug-file-list",
		app.DebugFileList,
		"Check how the your file list would be processed",
	)

	cmd.Var(
		&app.Tags,
		"tags",
		"Comma separated tags to apply to assets. Use forwards slashes to create hierarchal tags (ex. \"Holiday/Groundhog's Day\").",
	)
	cmd.Var(
		&app.RemoveTags,
		"remove-tags",
		"Comma separated tags to remove from assets before applying new tags. If option is provided without tags, all tags are removed.",
	)
	cmd.BoolFunc(
		"tag-with-session",
		"Tag uploaded assets according to the format immich-go/YYYY-MM-DD/HH-MI-SS",
		myflag.BoolFlagFn(&app.TagWithSession, false),
	)
	cmd.BoolFunc(
		"tag-with-path",
		"Hierarchically tag uploaded assets using path to assets",
		myflag.BoolFlagFn(&app.TagWithPath, false),
	)
	cmd.BoolFunc(
		"cleanup",
		"Delete unused tags",
		myflag.BoolFlagFn(&app.TagCleanup, false),
	)

	err = cmd.Parse(args)
	if err != nil {
		return nil, err
	}

	if len(app.Tags) == 0 && !app.TagWithPath && !app.TagWithSession && !app.TagCleanup &&
		app.RemoveTags == nil {
		return nil, errors.New(
			`provide at least one of the following flags: -tags, -tag-with-path, -tag-with-session, -tag-cleanup, -remove-tags`,
		)
	}

	if app.TagWithSession {
		app.sessionTag = time.Now().Format("immich-go/2006-01-02/15-04-05")
	}

	app.tagToAssetIDs = make(map[string][]string)

	if app.DebugFileList {
		if len(cmd.Args()) < 2 {
			return nil, fmt.Errorf(
				"the option -debug-file-list requires a file name and a date format",
			)
		}
		app.LogFile = strings.TrimSuffix(cmd.Arg(0), filepath.Ext(cmd.Arg(0))) + ".log"
		_ = os.Remove(app.LogFile)

		fsOpener = func() ([]fs.FS, error) {
			return fakefs.ScanFileList(cmd.Arg(0), cmd.Arg(1))
		}
	}

	app.BrowserConfig.Validate()
	err = app.SharedFlags.Start(ctx)
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

	if len(app.Tags) > 0 || app.TagWithPath || app.TagWithSession || app.RemoveTags != nil {
		if len(app.fsyss) == 0 {
			return nil, errors.New(
				"No file found matching the pattern: " + strings.Join(cmd.Args(), ","),
			)
		}
	}

	return &app, nil
}

func (app *TagCmd) run(ctx context.Context) error {
	defer func() {
		_ = fshelper.CloseFSs(app.fsyss)
	}()

	var err error
	app.Log.Info("Browsing folder(s)...")
	app.browser, err = app.exploreLocalFolder(ctx, app.fsyss)

	if err != nil {
		return err
	}

	defer func() {
		if app.DebugCounters {
			fn := strings.TrimSuffix(app.LogFile, filepath.Ext(app.LogFile)) + ".csv"
			f, err := os.Create(fn)
			if err == nil {
				_ = app.Jnl.WriteFileCounts(f)
				fmt.Println("\nCheck the counters file: ", f.Name())
				f.Close()
			}
		}
	}()

	return app.runNoUI(ctx)
}

func (app *TagCmd) getImmichAssets(ctx context.Context, updateFn func(value, maxValue int)) error {
	statistics, err := app.Immich.GetAssetStatistics(ctx)
	if err != nil {
		return err
	}
	totalOnImmich := statistics.Total
	received := 0

	var list []*immich.Asset

	err = app.Immich.GetAllAssetsWithFilter(ctx, func(a *immich.Asset) error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			received++
			list = append(list, a)
			if updateFn != nil {
				updateFn(received, totalOnImmich)
			}
			return nil
		}
	})
	if err != nil {
		return err
	}
	if updateFn != nil {
		updateFn(totalOnImmich, totalOnImmich)
	}
	app.AssetIndex = asset.NewAssetIndex(list)
	return nil
}

func (app *TagCmd) uploadLoop(ctx context.Context) error {
	/*
		https://github.com/immich-app/immich/discussions/13637#discussioncomment-11017586
		This hack resolve possible race condition where asset is moved out of upload dir
		by pausing the storageTemplateMigration job
	*/
	app.Log.Info(fmt.Sprintf("Pausing %s job", immich.StorageTemplateMigration))
	app.Immich.SendJobCommand(ctx, immich.StorageTemplateMigration, immich.Pause, false)

	var err error
	assetChan := app.browser.Browse(ctx)
assetLoop:
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()

		case a, ok := <-assetChan:
			if !ok {
				break assetLoop
			}
			if a.Err != nil {
				app.Jnl.Record(ctx, fileevent.Error, a, a.FileName, a.Err.Error())
			} else {
				err = app.handleAsset(ctx, a)
				if err != nil {
					app.Jnl.Record(ctx, fileevent.Error, a, a.FileName, a.Err.Error())
				}
			}
		}
	}

	if app.TagWithSession {
		app.Tags = append(app.Tags, time.Now().Format("immich-go/2006-01-02/15-04-05"))
	}

	if len(app.Tags) > 0 || app.TagWithPath {
		tagAssets(ctx, app, app.Tags, app.uploadedAssetIDs)

		if app.TagWithPath {
			for k, v := range app.tagToAssetIDs {
				if k == "." {
					continue
				}
				tagAssets(ctx, app, []string{k}, v)
			}
		}
	}

	if app.TagCleanup {
		fmt.Println("Deleting unused tags")
		app.Log.Info("Deleting unused tags")
		app.Immich.CreateJob(ctx, immich.TagCleanup)
	}

	app.Log.Info(fmt.Sprintf("Resuming %s job", string(immich.StorageTemplateMigration)))
	app.Immich.SendJobCommand(ctx, immich.StorageTemplateMigration, immich.Resume, false)

	return err
}

func (app *TagCmd) handleAsset(ctx context.Context, a *browser.LocalAssetFile) error {
	defer func() {
		a.Close()
	}()
	ext := path.Ext(a.FileName)
	if app.BrowserConfig.ExcludeExtensions.Exclude(ext) {
		app.Jnl.Record(
			ctx,
			fileevent.UploadNotSelected,
			a,
			a.FileName,
			"reason",
			"extension in rejection list",
		)
		return nil
	}
	if !app.BrowserConfig.SelectExtensions.Include(ext) {
		app.Jnl.Record(
			ctx,
			fileevent.UploadNotSelected,
			a,
			a.FileName,
			"reason",
			"extension not in selection list",
		)
		return nil
	}

	advice := app.AssetIndex.GetAdvice(a)
	switch advice.Advice {
	case asset.NotOnServer:
		app.Jnl.Record(ctx, fileevent.TagNotOnServer, a, a.FileName)
		return nil

	default:
		app.uploadedAssetIDs = append(app.uploadedAssetIDs, advice.ServerAsset.ID)
		if len(app.Tags) > 0 {
			app.Jnl.Record(
				ctx,
				fileevent.Tagged,
				a,
				a.FileName,
				"tags",
				app.Tags.String(),
				"reason",
				"option -tags",
			)
		}

		if app.TagWithSession {
			app.Jnl.Record(
				ctx,
				fileevent.Tagged,
				a,
				a.FileName,
				"tag",
				app.sessionTag,
				"reason",
				"option -tag-with-session",
			)
		}

		if app.TagWithPath {
			pathTag := filepath.ToSlash(filepath.Dir(a.FileName))
			if assetIDs := app.tagToAssetIDs[pathTag]; assetIDs != nil {
				app.tagToAssetIDs[pathTag] = append(assetIDs, advice.ServerAsset.ID)
			} else {
				app.tagToAssetIDs[pathTag] = []string{advice.ServerAsset.ID}
			}

			app.Jnl.Record(
				ctx,
				fileevent.Tagged,
				a,
				a.FileName,
				"tag",
				pathTag,
				"reason",
				"option -tag-with-path",
			)
		}

		fmt.Println(advice.ServerAsset.Tags...)
		// TODO bnguyen if app.RemoveTags
		// if app.RemoveTags != nil {
		// 	advice.ServerAsset.Tags
		// }

		return nil
	}
}

func (app *TagCmd) exploreLocalFolder(ctx context.Context, fsyss []fs.FS) (browser.Browser, error) {
	b, err := files.NewLocalFiles(ctx, app.Jnl, fsyss...)
	if err != nil {
		return nil, err
	}
	b.SetSupportedMedia(app.Immich.SupportedMedia())
	b.SetWhenNoDate("FILE")
	b.SetBannedFiles(app.BannedFiles)
	return b, nil
}

func tagAssets(
	ctx context.Context,
	app *TagCmd,
	tags []string,
	assetIDs []string,
) {
	if len(tags) == 0 || len(assetIDs) == 0 {
		return
	}

	if app.DryRun {
		return
	}

	app.Log.Info(fmt.Sprintf("Upserting tags: %s", strings.Join(tags, ", ")))
	tagResponses, err := app.Immich.UpsertTags(ctx, tags)
	if err != nil {
		app.Log.Error(fmt.Sprintf("Failed to UpsertTags: %s", err))
	} else {
		app.Log.Info(fmt.Sprintf("Tagging %d assets", len(assetIDs)))
		var tagIDs []string
		for _, tag := range tagResponses {
			tagIDs = append(tagIDs, tag.ID)
		}
		_, err := app.Immich.BulkTagAssets(ctx, tagIDs, assetIDs)
		if err != nil {
			app.Log.Error(fmt.Sprintf("Failed to BulkTagAssets for tagIDs %+v: %s", tagIDs, err))
		}
	}
}
