package native

import (
	"context"
	"errors"

	"github.com/simulot/immich-go/internal/ui/core/messages"
)

// Config contains knobs for native shells (placeholder for now).
type Config struct{}

// ErrUnavailable indicates that the native shell has not been implemented yet.
var ErrUnavailable = errors.New("native ui unavailable")

// Run drains the event stream until a real native implementation exists.
func Run(context.Context, Config, messages.Stream) error {
	return ErrUnavailable
}
