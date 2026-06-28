package version

import (
	"context"
	"fmt"

	"github.com/simulot/immich-go/app"
	"github.com/spf13/cobra"
)

// NewVersionCommand creates the version command.
func NewVersionCommand(ctx context.Context, a *app.Application) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "version",
		Short: "Give immich-go version",
	}

	cmd.RunE = func(cmd *cobra.Command, args []string) error {
		fmt.Println(app.GetVersion())
		return nil
	}
	return cmd
}
