package terminal

import "github.com/simulot/immich-go/internal/ui/core/services"

// Config contains terminal-shell specific options.
type Config struct {
	Theme        services.Theme
	ProfileLabel string
	ServerURL    string
	UserEmail    string
}

// DefaultConfig returns sensible defaults for the terminal shell.
func DefaultConfig() Config {
	return Config{Theme: services.DefaultTheme(), ProfileLabel: "default"}
}
