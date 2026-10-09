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
		Short: "Experimental media-first Takeout import with persistent metadata reconciliation",
		Long:  "Upload one Takeout ZIP at a time and apply capture dates/descriptions when exact-path sidecars arrive. No source deletion, downloads, albums, grouping, or fuzzy matching. Use one state file and export ID per export. Run without ZIP arguments to retry pending metadata.",
		RunE: func(cmd *cobra.Command, args []string) error {
			if client.PauseImmichBackgroundJobs {
				return errors.New("experimental importer does not support --pause-immich-jobs; server job races are treated as metadata conflicts")
			}
			if !acknowledge {
				return errors.New("pass --acknowledge-incomplete-import: media without sidecars may include trashed/partner photos; this PoC does not implement selection filters")
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
			for _, filename := range args {
				if err = session.ImportZIP(cmd.Context(), filename); err != nil {
					return err
				}
			}
			if err = session.Reconcile(cmd.Context()); err != nil {
				return err
			}
			summary := session.Summary()
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "ZIP processing finished: %d media receipts, %d enriched, %d unowned duplicates, %d awaiting metadata, %d sidecars awaiting media. Keep source ZIPs: albums, grouping and server job behavior are not fully reconciled.\n", summary.Uploaded, summary.Enriched, summary.Unowned, summary.PendingMetadata, summary.PendingMedia)
			return err
		},
	}
	client.RegisterFlags(command.Flags(), "")
	client.PauseImmichBackgroundJobs = false
	command.Flags().Lookup("pause-immich-jobs").DefValue = "false"
	_ = command.Flags().MarkHidden("pause-immich-jobs")
	command.Flags().BoolVar(&acknowledge, "acknowledge-incomplete-import", false, "Acknowledge experimental incomplete metadata and absence of trash/partner filters")
	command.Flags().StringVar(&stateFile, "state", "", "Persistent receipt/metadata state file (required)")
	command.Flags().StringVar(&exportID, "export-id", "", "Stable identifier for this Google Takeout export (required)")
	_ = command.MarkFlagRequired("state")
	_ = command.MarkFlagRequired("export-id")
	return command
}
