package app

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/config"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/spf13/cobra"
)

// Application holds configuration used by all commands
// It manages global settings like:
// - the log and the log-level
// - application counters
// - the concurrency
// - the configuration file

type Application struct {
	DryRun     bool                  `mapstructure:"dry-run" yaml:"dry-run" json:"dry-run" toml:"dry-run"`
	OnErrors   cliflags.OnErrorsFlag `mapstructure:"on-errors" yaml:"on-errors" json:"on-errors" toml:"on-errors"`
	SaveConfig bool                  `mapstructure:"sauve-config" yaml:"sauve-config" json:"sauve-config" toml:"sauve-config"`

	CfgFile string
	log     *Log
	jnl     *fileevent.Recorder
	tz      *time.Location
	Config  *config.ConfigurationManager

	numErrors atomic.Int64 // count the errors occurred during the run
}

func New(ctx context.Context, cmd *cobra.Command) *Application {
	// application's context
	a := &Application{
		log:    &Log{},
		tz:     time.Local,
		Config: config.New(),
	}
	return a
}

func (app *Application) Log() *Log {
	return app.log
}

func (app *Application) GetTZ() *time.Location {
	if app.tz == nil {
		app.tz = time.Local
	}
	return app.tz
}

func (app *Application) SetTZ(tz *time.Location) {
	app.tz = tz
}

func (app *Application) Jnl() *fileevent.Recorder {
	return app.jnl
}

func (app *Application) SetJnl(jnl *fileevent.Recorder) {
	app.jnl = jnl
}

func (app *Application) SetLog(log *Log) {
	app.log = log
}

func (app *Application) ProcessError(err error) error {
	if err == nil {
		return nil
	}
	// we don't count context.Canceled as an error
	// but we want to return it to the caller
	if errors.Is(err, context.Canceled) {
		return err
	}

	nErr := app.numErrors.Add(1)
	if app.OnErrors == cliflags.OnErrorsStop {
		app.Log().Error("Error", "err", err.Error())
		return err
	} else if app.OnErrors == cliflags.OnErrorsNeverStop {
		app.Log().Error("Error", "err", err.Error())
		return nil
	} else if nErr > int64(app.OnErrors) {
		app.Log().Error("Too many errors, stopping", "err", err.Error())
		return err
	}
	return nil
}
