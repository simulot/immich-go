//go:build ui_native

package native

import (
	"context"

	"github.com/simulot/immich-go/internal/ui/core/messages"
)

// Run is compiled when the native UI shell is enabled.
func Run(context.Context, Config, messages.Stream) error {
	return ErrUnavailable
}
