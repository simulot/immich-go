package incremental

import (
	"errors"
	"fmt"
	"strings"

	"github.com/simulot/immich-go/app"
	importer "github.com/simulot/immich-go/internal/incremental"
	"github.com/spf13/cobra"
)

func NewCommand(application *app.Application) *cobra.Command {
	var client app.Client
	var stateFile, exportID string
	var acknowledge bool
	command := &cobra.Command{
		Use:   "experimental-takeout [ZIP ...]",
		Short: "Experimental Takeout import with safe filenames and persistent metadata reconciliation",
		Long:  "Scan Takeout ZIPs for metadata, then upload only media with an unambiguous sidecar filename. Same-byte source occurrences share one asset receipt. Re-present/re-download deferred media ZIPs after metadata discovery; media bytes are not retained. Capture dates/descriptions can arrive later on any known occurrence. No source deletion, downloads, albums, grouping, or fuzzy matching. Use one state file and export ID per export. Run without ZIP arguments to retry pending metadata.",
		RunE: func(cmd *cobra.Command, args []string) (runErr error) {
			if client.PauseImmichBackgroundJobs {
				return errors.New("experimental importer does not support --pause-immich-jobs; server job races are treated as metadata conflicts")
			}
			if !acknowledge {
				return errors.New("pass --acknowledge-incomplete-import: unresolved filenames defer uploads and require media ZIPs again; this PoC does not implement trash/partner selection filters")
			}
			if application.DryRun || client.DryRun {
				return errors.New("experimental importer does not support dry-run; persistent state must describe real uploads")
			}
			if err := client.Open(cmd.Context(), application); err != nil {
				return err
			}
			defer client.Close()
			scope := importer.Scope{Server: strings.TrimSuffix(client.Server, "/"), UserID: client.User.ID, ExportID: exportID}
			session, err := importer.Open(stateFile, scope, client.Immich)
			if err != nil {
				return err
			}
			defer session.Close()
			defer func() {
				summary := session.Summary()
				_, printErr := fmt.Fprintf(cmd.OutOrStdout(), "State: %d logical assets across %d source occurrences, %d upload receipts, %d enriched, %d deferred media, %d unowned, %d unowned metadata, %d awaiting metadata, %d sidecars awaiting media, %d conflicts. Keep/re-download unresolved media ZIPs and re-present them after metadata discovery; no media bytes are retained. Albums, grouping and server job behavior are not fully reconciled.\n", summary.LogicalAssets, summary.Occurrences, summary.Uploaded, summary.Enriched, summary.Deferred, summary.Unowned, summary.UnownedMetadata, summary.PendingMetadata, summary.PendingMedia, summary.Conflicts)
				for _, issue := range session.Issues() {
					_, issueErr := fmt.Fprintf(cmd.OutOrStdout(), "Attention: source=%q checksum=%q: %s\n", issue.Source, issue.Checksum, issue.Reason)
					printErr = errors.Join(printErr, issueErr)
				}
				runErr = errors.Join(runErr, printErr)
			}()
			for _, filename := range args {
				if err = session.ImportZIP(cmd.Context(), filename); err != nil {
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
	_ = command.MarkFlagRequired("state")
	_ = command.MarkFlagRequired("export-id")
	return command
}
