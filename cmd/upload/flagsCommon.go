package upload

import (
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/spf13/cobra"
)

// CommonFlags represents a set of common flags used for filtering assets.
type CommonFlags struct {
	NoUI             bool // Disable UI
	StackJpgWithRaw  bool // Stack jpg/raw (Default: TRUE)
	StackBurstPhotos bool // Stack burst (Default: TRUE)

	Jnl *fileevent.Recorder // Log file events
}

// addCommonFlags adds the common flags to a Cobra command.
func addCommonFlags(cmd *cobra.Command) *CommonFlags {
	flags := CommonFlags{}

	cmd.Flags().BoolVar(&flags.NoUI, "no-ui", false, "Disable the user interface")
	cmd.Flags().BoolVar(&flags.StackJpgWithRaw, "stack-jpg-with-raw", false, "Stack JPG images with their corresponding raw images in Immich")
	cmd.Flags().BoolVar(&flags.StackBurstPhotos, "stack-burst-photos", false, "Stack bursts of photos in Immich")
	return &flags
}
