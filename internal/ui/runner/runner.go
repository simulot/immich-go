package runner

import (
	"context"
	"errors"
	"fmt"

	"github.com/simulot/immich-go/internal/ui/core/messages"
	"github.com/simulot/immich-go/internal/ui/platform/native"
	"github.com/simulot/immich-go/internal/ui/platform/terminal"
	"github.com/simulot/immich-go/internal/ui/platform/web"
)

// Mode enumerates supported UI shells.
type Mode string

const (
	ModeAuto     Mode = "auto"
	ModeOff      Mode = "off"
	ModeTerminal Mode = "terminal"
	ModeWeb      Mode = "web"
	ModeNative   Mode = "native"
)

// Config controls which UI shell should be launched.
type Config struct {
	Mode          Mode
	Experimental  bool
	LegacyEnabled bool
}

// ErrShellUnavailable is returned when the requested UI shell is not compiled in.
var ErrShellUnavailable = errors.New("ui shell not available")

// ErrNoShellSelected indicates that no UI shell could be selected.
var ErrNoShellSelected = errors.New("no ui shell selected")

// Run selects the best available UI shell or drains the stream when none are available.
func Run(ctx context.Context, cfg Config, stream messages.Stream) error {
	if cfg.Mode == ModeOff {
		drainStream(ctx, stream)
		return ErrNoShellSelected
	}

	if cfg.Mode == ModeAuto || cfg.Mode == ModeTerminal || cfg.Mode == "" {
		if err := terminal.Run(ctx, terminal.DefaultConfig(), stream); err == nil {
			return nil
		} else if !errors.Is(err, terminal.ErrUnavailable) {
			return err
		}
		if cfg.Mode == ModeTerminal {
			return fmt.Errorf("terminal ui unavailable: %w", terminal.ErrUnavailable)
		}
	}

	if cfg.Mode == ModeWeb || cfg.Mode == ModeAuto {
		if err := web.Run(ctx, web.Config{}, stream); err == nil {
			return nil
		} else if !errors.Is(err, web.ErrUnavailable) {
			return err
		}
		if cfg.Mode == ModeWeb {
			return fmt.Errorf("web ui unavailable: %w", web.ErrUnavailable)
		}
	}

	if cfg.Mode == ModeNative || cfg.Mode == ModeAuto {
		if err := native.Run(ctx, native.Config{}, stream); err == nil {
			return nil
		} else if !errors.Is(err, native.ErrUnavailable) {
			return err
		}
		if cfg.Mode == ModeNative {
			return fmt.Errorf("native ui unavailable: %w", native.ErrUnavailable)
		}
	}

	drainStream(ctx, stream)

	if cfg.LegacyEnabled {
		return ErrShellUnavailable
	}

	return ErrNoShellSelected
}

func drainStream(ctx context.Context, stream messages.Stream) {
	if stream == nil {
		return
	}
	for {
		select {
		case <-ctx.Done():
			return
		case _, ok := <-stream:
			if !ok {
				return
			}
		}
	}
}
