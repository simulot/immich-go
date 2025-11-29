//go:build ui_web

package web

import (
	"context"

	"github.com/simulot/immich-go/internal/ui/core/messages"
)

// Run is compiled when the web UI shell is enabled.
func Run(context.Context, Config, messages.Stream) error {
	return ErrUnavailable
}
