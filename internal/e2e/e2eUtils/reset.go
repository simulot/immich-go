package e2eutils

import (
	"context"
	"fmt"
	"log/slog"
	"os/exec"
)

func (ictlr *ImmichController) ResetImmich(ctx context.Context) error {
	// Reset immich's database
	// https://github.com/immich-app/immich/blob/main/e2e/src/utils.ts
	//

	if ictlr.PingAPI(ctx) == nil {
		err := ictlr.PauseImmichServer(ctx)
		if err != nil {
			return fmt.Errorf("can't stop immich: %w", err)
		}
	}
	args := []string{
		"exec", "-i", "immich_postgres", "psql", "--dbname=immich", "--username=postgres", "-c",
		`
        delete from stack CASACDE;
        delete from library CASACDE;
        delete from shared_link CASACDE;
        delete from person CASACDE;
        delete from album CASACDE;
        delete from asset CASACDE;
        delete from asset_face CASACDE;
        delete from activity CASACDE;
        delete from tag CASACDE;
        -- delete from session CASACDE;
        -- delete from api_key CASACDE;
        -- delete from user CASACDE;
        -- delete from system_metadata where "key" NOT IN ('reverse-geocoding-state', 'system-flags');
		`,
	}
	slog.Info("exec", "command", "docker", "args", args[:4])
	c := exec.CommandContext(ctx, "docker", args...)
	out, err := c.CombinedOutput()
	if err != nil {
		return fmt.Errorf("can't reset immich: %w\n%s", err, string(out))
	}

	if err := ictlr.ResumeImmichServer(ctx); err != nil {
		err = fmt.Errorf("can't reset immich: %w\n%s", err, string(out))
		return err
	}
	return err
}
