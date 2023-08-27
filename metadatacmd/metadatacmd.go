package metadatacmd

import (
	"context"
	"fmt"
	"immich-go/immich"
	"immich-go/immich/logger"
)

type App struct {
	Immich *immich.ImmichClient // Immich client

}

func MetadataCommand(ctx context.Context, ic *immich.ImmichClient, log *logger.Logger, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("missing metadata sub command: check|fix")
	}
	app := App{}
	cmd := args[0]
	switch cmd {
	case "check":
		app.CheckCommand(ctx, ic, log, args[1:])
	default:
		return fmt.Errorf("unknown metadata sub command: %s", cmd)
	}
	return nil
}

func (app *App) CheckCommand(ctx context.Context, ic *immich.ImmichClient, log *logger.Logger, args []string) error {

	log.MessageContinue(logger.OK, "Get server's assets...")
	list, err := app.Immich.GetAllAssets(ctx, nil)
	if err != nil {
		return err
	}
	log.MessageTerminate(logger.OK, " %d received", len(list))

	app.AssetIndex = &assterAssetIndex{
		assets: list,
	}

	app.AssetIndex.ReIndex()

	return nil
}
