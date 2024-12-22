package cmd

import (
	"context"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/app/cmd/archive"
	"github.com/simulot/immich-go/app/cmd/stack"
	"github.com/simulot/immich-go/app/cmd/upload"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// Run immich-go
func RootImmichGoCommand(ctx context.Context) (*cobra.Command, *app.Application) {
	viper.SetEnvPrefix("IMMICHGO")

	// Create the application context

	// Add the root command
	c := &cobra.Command{
		Use:     "immich-go",
		Short:   "Immich-go is a command line application to interact with the Immich application using its API",
		Long:    `An alternative to the immich-CLI command that doesn't depend on nodejs installation. It tries its best for importing google photos takeout archives.`,
		Version: app.Version,
	}
	cobra.EnableTraverseRunHooks = true // doc: cobra/site/content/user_guide.md
	a := app.New(ctx, c)

	// add immich-go commands
	c.AddCommand(
		app.NewVersionCommand(ctx, a),
		upload.NewUploadCommand(ctx, a),
		archive.NewArchiveCommand(ctx, a),
		stack.NewStackCommand(ctx, a),
	)

	return c, a
}
