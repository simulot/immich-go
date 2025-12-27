//go:build ui_terminal

package terminal

import (
	"context"

	"github.com/simulot/immich-go/internal/ui/core/messages"
)

// Run starts the Immich upload dashboard Bubble Tea program.
func Run(ctx context.Context, cfg Config, stream messages.Stream) error {
	return runUploadDashboard(ctx, cfg, stream)
}
