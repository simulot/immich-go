package cmd

import (
	"context"
	"fmt"
	"os"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/app/cmd/archive"
	"github.com/simulot/immich-go/app/cmd/stack"
	"github.com/simulot/immich-go/app/cmd/upload"
	"github.com/simulot/immich-go/internal/config"
	"github.com/spf13/cobra"
)

// Run immich-go
func RootImmichGoCommand(ctx context.Context) (*cobra.Command, *app.Application) {
	var cfgFile string
	cm := config.New()

	// Create the application context

	// Add the root command
	c := &cobra.Command{
		Use:     "immich-go",
		Short:   "Immich-go is a command line application to interact with the Immich application using its API",
		Long:    `An alternative to the immich-CLI command that doesn't depend on nodejs installation. It tries its best for importing google photos takeout archives.`,
		Version: app.Version,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			err := cm.Init(cfgFile)
			if err != nil {
				return err
			}
			err = cm.ProcessCommand(cmd)
			if err != nil {
				return err
			}
			if save, _ := cmd.Flags().GetBool("save-config"); save {
				if err := cm.Save("immich-go.yaml"); err != nil {
					fmt.Fprintln(os.Stderr, "Can't save the configuration: ", err.Error())
				}
			}
			return nil
		},
	}
	c.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is ./immich-go.yaml)")
	c.PersistentFlags().Bool("save-config", false, "Save the configuration to immich-go.yaml")

	cobra.EnableTraverseRunHooks = true // doc: cobra/site/content/user_guide.md
	a := app.New(ctx, c, cm)

	// add immich-go commands
	c.AddCommand(
		app.NewVersionCommand(ctx, a),
		upload.NewUploadCommand(ctx, a),
		archive.NewArchiveCommand(ctx, a),
		stack.NewStackCommand(ctx, a),
	)

	return c, a
}

func NewToolCommand(ctx context.Context, a *app.Application) *cobra.Command {
	c := &cobra.Command{
		Use:   "tool",
		Short: "Miscellaneous tools",
	}
	return c
}
