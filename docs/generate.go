package main

import (
	"context"
	"fmt"

	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/app/cmd"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

/* Generate documentation for the command */

func main() {
	ctx := context.Background()
	c := &cobra.Command{
		Use:     "immich-go",
		Short:   "Immich-go is a command line application to interact with the Immich application using its API",
		Long:    `An alternative to the immich-CLI command that doesn't depend on nodejs installation. It tries its best for importing google photos takeout archives.`,
		Version: app.Version,
	}
	cobra.EnableTraverseRunHooks = true // doc: cobra/site/content/user_guide.md
	a := app.New(ctx, c)

	// add immich-go commands
	c.AddCommand(app.NewVersionCommand(ctx, a))
	cmd.AddCommands(c, ctx, a)
	fmt.Println(doc.GenMarkdownTree(c, "../docs"))
}
