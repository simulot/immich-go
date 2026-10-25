package incremental

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/simulot/immich-go/app"
	importer "github.com/simulot/immich-go/internal/incremental"
	"github.com/spf13/cobra"
)

func NewCommand(application *app.Application, sessionOptions ...importer.Option) *cobra.Command {
	var client app.Client
	var stateFile, exportID, stagingDir, archiveList string
	var maxStagingBytes int64
	var acknowledge bool
	command := &cobra.Command{
		Use:   "experimental-takeout [ZIP ...]",
		Short: "Experimental Takeout import with safe filenames and persistent metadata reconciliation",
		Long:  "Scan Takeout ZIPs for metadata, then upload only media with an unambiguous sidecar filename. Same-byte source occurrences share one asset receipt. Opt in to durable media retention with --staging-dir and a positive --max-staging-bytes. Free-space warnings start at 50 GB, danger at 20 GB; writes preserve a 10 GB reserve and switch to a catalog/second-pass workflow when retention is unavailable. GB means 1,000,000,000 bytes. Existing cached media is preserved. Use a private, initially empty directory bound to this exact state-file location; keep both locations fixed. Persisted staging settings restore when omitted. Supply --archive-list with one ZIP filename per line in export order for exact next-download/replay guidance; the list and progress persist across runs. Input ZIPs are never deleted and downloads remain manual. Capture dates/descriptions can arrive later on any known occurrence. This PoC does not implement albums, grouping, selection filters or fuzzy matching. Run without ZIP arguments to upload ready cached media and retry metadata.",
		RunE: func(cmd *cobra.Command, args []string) (runErr error) {
			options := append([]importer.Option(nil), sessionOptions...)
			var statusPrintErr error
			options = append(options, importer.WithStorageObserver(func(status importer.StorageStatus) {
				statusPrintErr = errors.Join(statusPrintErr, printStorageStatus(cmd.ErrOrStderr(), status))
			}))
			defer func() { runErr = errors.Join(runErr, statusPrintErr) }()
			var expectedArchives []string
			if archiveList != "" {
				var err error
				expectedArchives, err = readArchiveList(archiveList)
				if err != nil {
					return err
				}
			}
			if cmd.Flags().Changed("staging-dir") || cmd.Flags().Changed("max-staging-bytes") {
				if strings.TrimSpace(stagingDir) == "" || maxStagingBytes <= 0 {
					return errors.New("--staging-dir and a positive --max-staging-bytes must be supplied together")
				}
				options = append(options, importer.WithStaging(stagingDir, maxStagingBytes))
			}
			if client.PauseImmichBackgroundJobs {
				return errors.New("experimental importer does not support --pause-immich-jobs; server job races are treated as metadata conflicts")
			}
			if !acknowledge {
				return errors.New("pass --acknowledge-incomplete-import: unresolved filenames defer uploads (retain media ZIPs unless staging succeeds); this PoC does not implement trash/partner selection filters")
			}
			if application.DryRun || client.DryRun {
				return errors.New("experimental importer does not support dry-run; persistent state must describe real uploads")
			}
			if err := client.Open(cmd.Context(), application); err != nil {
				return err
			}
			defer client.Close()
			scope := importer.Scope{Server: strings.TrimSuffix(client.Server, "/"), UserID: client.User.ID, ExportID: exportID}
			session, err := importer.Open(stateFile, scope, client.Immich, options...)
			if err != nil {
				return err
			}
			defer session.Close()
			defer func() {
				summary := session.Summary()
				_, printErr := fmt.Fprintf(cmd.OutOrStdout(), "State: %d logical assets across %d source occurrences, %d upload receipts, %d enriched, %d deferred media, %d unowned, %d unowned metadata, %d awaiting metadata, %d sidecars awaiting media, %d conflicts. Albums, grouping and server job behavior are not fully reconciled.\n", summary.LogicalAssets, summary.Occurrences, summary.Uploaded, summary.Enriched, summary.Deferred, summary.Unowned, summary.UnownedMetadata, summary.PendingMetadata, summary.PendingMedia, summary.Conflicts)
				if summary.MaxStagingBytes > 0 {
					_, stagingErr := fmt.Fprintf(cmd.OutOrStdout(), "Staging: %d actual retained bytes (%d pending bytes across %d cached assets), cap %d bytes. Ready cached media uploads on import/reconcile; managed copies are released after durable upload receipts. Existing cache is retained when storage switches to two-pass mode. Keep the current ZIP on any error. Input ZIPs are never deleted.\n", summary.StagedBytes, summary.PendingStagedBytes, summary.StagedAssets, summary.MaxStagingBytes)
					printErr = errors.Join(printErr, stagingErr)
				} else {
					_, modeErr := fmt.Fprintln(cmd.OutOrStdout(), "Keep/re-download unresolved media ZIPs and re-present them after metadata discovery; no media bytes are retained.")
					printErr = errors.Join(printErr, modeErr)
				}
				for _, issue := range session.Issues() {
					_, issueErr := fmt.Fprintf(cmd.OutOrStdout(), "Attention: source=%q checksum=%q: %s\n", issue.Source, issue.Checksum, issue.Reason)
					printErr = errors.Join(printErr, issueErr)
				}
				printErr = errors.Join(printErr, printArchiveAction(cmd.OutOrStdout(), session.NextArchive()))
				runErr = errors.Join(runErr, printErr)
			}()
			if archiveList != "" {
				if err := session.ConfigureArchives(expectedArchives); err != nil {
					return err
				}
			}
			for _, filename := range args {
				if err = session.ImportArchive(cmd.Context(), filename); err != nil {
					return err
				}
			}
			if err = session.Reconcile(cmd.Context()); err != nil {
				return err
			}
			return nil
		},
	}
	client.RegisterFlags(command.Flags(), "")
	client.PauseImmichBackgroundJobs = false
	command.Flags().Lookup("pause-immich-jobs").DefValue = "false"
	_ = command.Flags().MarkHidden("pause-immich-jobs")
	command.Flags().BoolVar(&acknowledge, "acknowledge-incomplete-import", false, "Acknowledge deferred media, incomplete metadata and absence of trash/partner filters")
	command.Flags().StringVar(&stateFile, "state", "", "Persistent receipt/metadata state file (required)")
	command.Flags().StringVar(&exportID, "export-id", "", "Stable identifier for this Google Takeout export (required)")
	command.Flags().StringVar(&stagingDir, "staging-dir", "", "Private, initially empty durable media directory; bound to this state-file location (unsupported on Windows)")
	command.Flags().Int64Var(&maxStagingBytes, "max-staging-bytes", 0, "Positive retained-media cap, required with --staging-dir; established caps may only increase")
	command.Flags().StringVar(&archiveList, "archive-list", "", "Export ZIP filenames in order, one per line; enables exact next-download/replay guidance")
	_ = command.MarkFlagRequired("state")
	_ = command.MarkFlagRequired("export-id")
	return command
}

func readArchiveList(filename string) ([]string, error) {
	f, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("read archive list: %w", err)
	}
	defer f.Close()
	var names []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if name := strings.TrimSpace(scanner.Text()); name != "" {
			names = append(names, name)
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read archive list: %w", err)
	}
	if len(names) == 0 {
		return nil, errors.New("archive list is empty; supply the export's ZIP filenames in order")
	}
	return names, nil
}

func printStorageStatus(w io.Writer, status importer.StorageStatus) error {
	var printErr error
	if status.Level != "ok" {
		reason := "Media writes preserve a 10 GB reserve."
		switch status.Level {
		case "warning":
			reason = "Free space is at or below 50 GB; danger starts at 20 GB and media writes preserve a 10 GB reserve."
		case "danger":
			reason = "Free space is at or below 20 GB; media writes preserve a 10 GB reserve."
		}
		_, printErr = fmt.Fprintf(w, "%s: %.2f GB free. %s\n", strings.ToUpper(status.Level), float64(status.FreeBytes)/1e9, reason)
	}
	if status.Fallback {
		_, fallbackErr := fmt.Fprintf(w, "FALLBACK: %.2f GB currently free. Persistent two-pass mode was selected because: %s. New media retention has stopped; existing cache is preserved. Continue the catalog pass, then re-present only the media archives requested below. A 10 GB reserve is required for importer media writes.\n", float64(status.FreeBytes)/1e9, status.Reason)
		printErr = errors.Join(printErr, fallbackErr)
	}
	return printErr
}

func printArchiveAction(w io.Writer, action importer.ArchiveAction) error {
	if action.Archive == "" {
		_, err := fmt.Fprintf(w, "Next: %s\n", action.Reason)
		return err
	}
	_, err := fmt.Fprintf(w, "Next: %s %q. %s\n", action.Action, action.Archive, action.Reason)
	return err
}
