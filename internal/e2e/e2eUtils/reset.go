package e2eutils

/*
// getSSHHostFromURL extracts the hostname from E2E_SERVER for SSH connections
func getSSHHostFromURL() string {
	immichURL := os.Getenv("E2E_SERVER")
	if immichURL == "" {
		return "" // Local execution
	}

	parsed, err := url.Parse(immichURL)
	if err != nil {
		slog.Warn("failed to parse E2E_SERVER", "url", immichURL, "error", err)
		return ""
	}

	// Return just the hostname (without port)
	return parsed.Hostname()
}

func (ictlr *ImmichController) ResetImmich(ctx context.Context) error {
	// Reset immich's database
	// https://github.com/immich-app/immich/blob/main/e2e/src/utils.ts

	if ictlr.PingAPI(ctx) == nil {
		err := ictlr.PauseImmichServer(ctx)
		if err != nil {
			return fmt.Errorf("can't stop immich: %w", err)
		}
	}

	sqlCmd := `
        delete from stack CASCADE;
        delete from library CASCADE;
        delete from shared_link CASCADE;
        delete from person CASCADE;
        delete from album CASCADE;
        delete from asset CASCADE;
        delete from asset_face CASCADE;
        delete from activity CASCADE;
        delete from tag CASCADE;
        -- delete from session CASCADE;
        -- delete from api_key CASCADE;
        -- delete from user CASCADE;
        -- delete from system_metadata where "key" NOT IN ('reverse-geocoding-state', 'system-flags');
	`

	args := []string{
		"exec", "-i", "immich_postgres", "psql", "--dbname=immich", "--username=postgres", "-c",
		sqlCmd,
	}

	err := ictlr.execCommand(ctx, timeout, "docker", args...)
	if err != nil {
		return fmt.Errorf("can't reset immich: %w", err)
	}

	if err = ictlr.ResumeImmichServer(ctx); err != nil {
		return fmt.Errorf("can't resume immich: %w", err)
	}
	return nil
}

*/
