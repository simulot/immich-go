package sync

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/internal/syncstate"
	"github.com/spf13/cobra"
)

func newDownCommand(_ context.Context, opts *syncOptions) *cobra.Command {
	client := &app.Client{}

	cmd := &cobra.Command{
		Use:   "down",
		Short: "Download assets from Immich server to local directory",
		RunE: func(cmd *cobra.Command, args []string) error { //nolint:contextcheck
			return runDown(cmd.Context(), opts, client)
		},
	}

	client.RegisterFlags(cmd.Flags(), "")
	return cmd
}

func runDown(ctx context.Context, opts *syncOptions, client *app.Client) error {
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

	// Ensure local directory exists
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	// Build set of server assets
	log.Message("Fetching server assets...")
	serverAssets := make(map[string]*immich.Asset) // checksum → asset

	err = client.Immich.GetAllAssets(ctx, func(a *immich.Asset) error {
		if a.IsTrashed {
			return nil
		}
		if a.Checksum != "" {
			serverAssets[a.Checksum] = a
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("fetching server assets: %w", err)
	}
	log.Message("Found %d assets on server", len(serverAssets))

	// Download missing assets
	downloaded := 0
	for checksum, sa := range serverAssets {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Already in state and file exists locally?
		if entry, ok := state.Assets[checksum]; ok {
			localPath := filepath.Join(dir, entry.Path)
			if _, err := os.Stat(localPath); err == nil {
				continue // already have it
			}
		}

		// Download the asset
		targetPath := buildLocalPath(sa)
		fullPath := filepath.Join(dir, targetPath)

		if dryRun {
			log.Message("[dry-run] Would download %s → %s", sa.OriginalFileName, targetPath)
		} else {
			log.Message("Downloading %s → %s", sa.OriginalFileName, targetPath)
			if err := downloadAsset(ctx, client.Immich, sa.ID, fullPath); err != nil {
				log.Error("Download failed", "file", sa.OriginalFileName, "err", err.Error())
				continue
			}
		}

		sm.TrackAsset(checksum, syncstate.AssetEntry{
			ID:       sa.ID,
			Filename: sa.OriginalFileName,
			Path:     targetPath,
			Size:     sa.ExifInfo.FileSizeInByte,
		})
		downloaded++

		if err := sm.SaveIfNeeded(50); err != nil {
			log.Error("Saving state", "err", err.Error())
		}
	}

	log.Message("Downloaded %d new assets", downloaded)

	// Handle deletions
	if opts.Delete {
		var toDelete []string
		for checksum, entry := range state.Assets {
			if _, onServer := serverAssets[checksum]; !onServer {
				toDelete = append(toDelete, checksum)
				_ = entry // used below
			}
		}

		if len(toDelete) > 0 {
			if len(toDelete) > 10 && !opts.Force {
				if !confirmDeletion(len(toDelete), "local files") {
					log.Message("Deletion cancelled by user")
					toDelete = nil
				}
			}

			deleted := 0
			for _, checksum := range toDelete {
				entry := state.Assets[checksum]
				localPath := filepath.Join(dir, entry.Path)

				if dryRun {
					log.Message("[dry-run] Would delete local file %s", entry.Path)
				} else {
					if err := os.Remove(localPath); err != nil && !os.IsNotExist(err) {
						log.Error("Deleting file", "path", entry.Path, "err", err.Error())
						continue
					}
					log.Message("Deleted local file %s", entry.Path)
				}
				sm.RemoveAsset(checksum)
				deleted++
			}
			log.Message("Deleted %d local files", deleted)
		}
	}

	return sm.Save()
}

// buildLocalPath creates a YYYY/YYYY-MM/filename path from an immich asset.
func buildLocalPath(a *immich.Asset) string {
	d := a.ExifInfo.DateTimeOriginal.Time
	if d.IsZero() {
		d = a.FileCreatedAt.Time
	}
	if d.IsZero() {
		return filepath.Join("no-date", a.OriginalFileName)
	}
	return filepath.Join(
		fmt.Sprintf("%04d", d.Year()),
		fmt.Sprintf("%04d-%02d", d.Year(), d.Month()),
		a.OriginalFileName,
	)
}

// downloadAsset downloads an asset by ID and writes it to the given local path.
func downloadAsset(ctx context.Context, client immich.ImmichInterface, assetID, localPath string) error {
	if err := os.MkdirAll(filepath.Dir(localPath), 0o755); err != nil {
		return err
	}

	rc, err := client.DownloadAsset(ctx, assetID)
	if err != nil {
		return err
	}
	defer rc.Close()

	// Write atomically via temp file
	tmpPath := localPath + ".tmp"
	f, err := os.Create(tmpPath)
	if err != nil {
		return err
	}
	if _, err := io.Copy(f, rc); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return err
	}
	if err := f.Sync(); err != nil {
		f.Close()
		os.Remove(tmpPath)
		return err
	}
	f.Close()

	return os.Rename(tmpPath, localPath)
}

// confirmDeletion prompts the user for confirmation before mass deletions.
func confirmDeletion(count int, what string) bool {
	fmt.Printf("About to delete %d %s. Continue? [y/N]: ", count, what)
	reader := bufio.NewReader(os.Stdin)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}
