//go:build ui_terminal

package terminal

import (
	"context"

	"github.com/simulot/immich-go/internal/ui/core/messages"
)

// Run will host the Bubble Tea program once implemented. For now it reports unavailability.
func Run(context.Context, Config, messages.Stream) error {
	return ErrUnavailable
}
