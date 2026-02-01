package sync

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/fshelper"
	"github.com/simulot/immich-go/internal/fshelper/hash"
	"github.com/simulot/immich-go/internal/syncstate"
	"github.com/spf13/cobra"
)

func newUpCommand(_ context.Context, opts *syncOptions) *cobra.Command {
	client := &app.Client{}

	cmd := &cobra.Command{
		Use:   "up",
		Short: "Upload local assets to Immich server",
		RunE: func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
			return runUp(cmd.Context(), opts, client)
		},
	}

	client.RegisterFlags(cmd.Flags(), "")
	return cmd
}

func runUp(ctx context.Context, opts *syncOptions, client *app.Client) error {
	log := opts.app.Log()

	dir := opts.Directory
	if dir == "" {
		return fmt.Errorf("--directory (-d) is required")
	}
	dir, err := filepath.Abs(dir)
	if err != nil {
		return fmt.Errorf("resolving directory: %w", err)
	}

	// Open client connection
	if err := client.Open(ctx, opts.app); err != nil {
		return err
	}
	defer client.Close()

	dryRun := opts.app.DryRun || client.DryRun

	// State management
	sm := syncstate.NewManager(dir, dryRun)
	if err := sm.Lock(); err != nil {
		return err
	}
	defer func() {
		if sm.NeedsSave() {
			log.Message("Saving state before exit...")
			_ = sm.Save()
		}
		_ = sm.Unlock()
	}()

	state, err := sm.Load()
	if err != nil {
		return fmt.Errorf("loading state: %w", err)
	}

	if err := sm.Validate(client.Server, client.User.ID, opts.Force); err != nil {
		return err
	}

	// Set up graceful shutdown: Ctrl+C saves state and unlocks
	ctx, stopSignal := withGracefulShutdown(ctx, sm, log)
	defer stopSignal()

	// Build server checksum index
	log.Message("Fetching server assets...")
	serverChecksums := make(map[string]string) // checksum → assetID

	err = client.Immich.GetAllAssets(ctx, func(a *immich.Asset) error {
		if a.IsTrashed {
			return nil
		}
		if a.Checksum != "" {
			serverChecksums[a.Checksum] = a.ID
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("fetching server assets: %w", err)
	}
	log.Message("Found %d assets on server", len(serverChecksums))

	// Scan local directory
	log.Message("Scanning local directory...")
	localFiles := make(map[string]string) // checksum → relative path
	uploaded := 0

	walkFn := func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return nil // skip errors
		}
		if d.IsDir() {
			base := filepath.Base(path)
			if base == ".immich-sync" || strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			if opts.NoRecursive && path != dir {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip hidden files
		if strings.HasPrefix(d.Name(), ".") {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		relPath, _ := filepath.Rel(dir, path)

		// Compute checksum
		checksum, err := fileSHA1(path)
		if err != nil {
			log.Error("Computing checksum", "file", relPath, "err", err.Error())
			return nil
		}

		localFiles[checksum] = relPath

		// Already on server?
		if _, ok := serverChecksums[checksum]; ok {
			// Update state to track it even if we didn't upload it
			info, _ := d.Info()
			var size int64
			if info != nil {
				size = info.Size()
			}
			sm.TrackAsset(checksum, syncstate.AssetEntry{
				ID:       serverChecksums[checksum],
				Filename: d.Name(),
				Path:     relPath,
				Size:     size,
			})
			return nil
		}

		// Upload
		info, _ := d.Info()
		var size int64
		if info != nil {
			size = info.Size()
		}

		if dryRun {
			log.Message("[dry-run] Would upload %s", relPath)
		} else {
			log.Message("Uploading %s", relPath)
			assetID, uploadErr := uploadFile(ctx, client.Immich, path, d.Name(), checksum)
			if uploadErr != nil {
				log.Error("Upload failed", "file", relPath, "err", uploadErr.Error())
				return nil
			}
			serverChecksums[checksum] = assetID

			sm.TrackAsset(checksum, syncstate.AssetEntry{
				ID:       assetID,
				Filename: d.Name(),
				Path:     relPath,
				Size:     size,
			})
			uploaded++

			if err := sm.SaveIfNeeded(50); err != nil {
				log.Error("Saving state", "err", err.Error())
			}
		}

		return nil
	}

	if err := filepath.WalkDir(dir, walkFn); err != nil {
		return fmt.Errorf("scanning directory: %w", err)
	}
	log.Message("Uploaded %d new assets", uploaded)

	// Handle deletions (delete server assets that are tracked in state but no longer local)
	if opts.Delete {
		var toDelete []struct {
			checksum string
			assetID  string
		}
		for checksum, entry := range state.Assets {
			if _, stillLocal := localFiles[checksum]; !stillLocal {
				toDelete = append(toDelete, struct {
					checksum string
					assetID  string
				}{checksum: checksum, assetID: entry.ID})
			}
		}

		if len(toDelete) > 0 {
			if len(toDelete) > 10 && !opts.Force {
				if !confirmDeletion(len(toDelete), "server assets") {
					log.Message("Deletion cancelled by user")
					toDelete = nil
				}
			}

			deleted := 0
			for _, item := range toDelete {
				if dryRun {
					entry := state.Assets[item.checksum]
					log.Message("[dry-run] Would delete server asset %s (%s)", entry.Filename, item.assetID)
				} else {
					if err := client.Immich.DeleteAssets(ctx, []string{item.assetID}, false); err != nil {
						log.Error("Deleting server asset", "id", item.assetID, "err", err.Error())
						continue
					}
					entry := state.Assets[item.checksum]
					log.Message("Deleted server asset %s (%s)", entry.Filename, item.assetID)
				}
				sm.RemoveAsset(item.checksum)
				deleted++
			}
			log.Message("Deleted %d server assets", deleted)
		}
	}

	return sm.Save()
}

// fileSHA1 computes the SHA1 checksum of a file, returned as a base64 string
// matching the format Immich uses.
func fileSHA1(path string) (string, error) {
	fsys := os.DirFS(filepath.Dir(path))
	return hash.Base64Encode(hash.FileSHA1Hash(fsys, filepath.Base(path)))
}

// uploadFile uploads a local file to the Immich server and returns the asset ID.
func uploadFile(ctx context.Context, client immich.ImmichInterface, localPath, filename, checksum string) (string, error) {
	fsys := os.DirFS(filepath.Dir(localPath))

	a := &assets.Asset{
		OriginalFileName: filename,
		File:             fshelper.FSName(fsys, filepath.Base(localPath)),
		Checksum:         checksum,
	}

	// Get file info for size
	info, err := os.Stat(localPath)
	if err != nil {
		return "", err
	}
	a.FileSize = int(info.Size())
	a.FileDate = info.ModTime()

	resp, err := client.AssetUpload(ctx, a)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}
