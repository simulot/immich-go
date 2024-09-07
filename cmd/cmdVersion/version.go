package cmdVersion

import (
	"fmt"

	"github.com/simulot/immich-go/cmd"
	"github.com/spf13/cobra"
)

func AddCommand(root *cmd.RootImmichFlags, version string, commit string, date string) {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number of Immich",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("immich-go  %s, commit %s, built at %s\n", version, commit, date)
		},
	}
	root.Command.AddCommand(versionCmd)
}
