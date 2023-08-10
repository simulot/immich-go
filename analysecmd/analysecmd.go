// This package analyses the content of an immich server
// It reports:
// - missing GPS info
// - missing or wrong date of capture
// - camera maker
// - file type
package analysecmd

import (
	"context"
	"immich-go/immich"
	"immich-go/immich/logger"
)

type AnalyseCmd struct {
	Immich *immich.ImmichClient // Immich client
	logger *logger.Logger
}

func AnalyseCommand(ctx context.Context, ic *immich.ImmichClient, logger *logger.Logger, args []string) error {
	app := &AnalyseCmd{
		logger: logger,
		Immich: ic,
	}
	logger.Info("Get server's assets...")
	assets, err := app.Immich.GetAllAssets(nil)
	if err != nil {
		return err
	}
	logger.OK("%d assets on the server.", app.AssetIndex.Len())
	return nil

}
