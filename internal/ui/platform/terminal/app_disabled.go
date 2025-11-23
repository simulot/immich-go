//go:build !ui_terminal

package terminal

import (
	"context"

	"github.com/simulot/immich-go/internal/ui/core/messages"
)

// Run is compiled when the terminal UI is disabled.
func Run(context.Context, Config, messages.Stream) error {
	return ErrUnavailable
}
