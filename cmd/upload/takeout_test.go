package upload

// func simulate_upload() {
// 	ic := &icCatchUploadsAssets{
// 		albums: map[string][]string{},
// 	}
// 	ctx := context.Background()

// 	log := slog.New(slog.NewTextHandler(io.Discard, nil))
// 	serv := cmd.SharedFlags{
// 		Immich: ic,
// 		Jnl:    fileevent.NewRecorder(log, false),
// 		Log:    log,
// 	}

// 	args := append([]string{"-no-ui"}, tc.args...)

// 	err := UploadCommand(ctx, &serv, args)
// 	if err != nil {
// 		t.Errorf("can't instantiate the UploadCmd: %s", err)
// 		return
// 	}
// }
