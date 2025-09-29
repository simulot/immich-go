package tool

import (
	"context"

	"github.com/simulot/immich-go/app"
	"github.com/spf13/cobra"
)

func NewToolCommand(ctx context.Context, a *app.Application) *cobra.Command {
	c := &cobra.Command{
		Use:   "tool",
		Short: "Miscellaneous tools",
	}
	return c
}
