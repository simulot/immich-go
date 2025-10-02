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
	"github.com/spf13/pflag"
)

type (
	RunE        func(cmd *cobra.Command, args []string) error
	RunEAdaptor func(ctx context.Context, cmd *cobra.Command, app *Application) error
)

// Application holds configuration used by commands
type Application struct {
	client Client
	log    *Log
	jnl    *fileevent.Recorder
	tz     *time.Location
	Config *config.ConfigurationManager
	CommonFlags

	numErrors atomic.Int64 // count the errors occurred during the run
}

type CommonFlags struct {
	DryRun   bool                  `mapstructure:"dry-run" yaml:"dry-run" json:"dry-run" toml:"dry-run"`
	OnErrors cliflags.OnErrorsFlag `mapstructure:"on-errors" yaml:"on-errors" json:"on-errors" toml:"on-errors"`
}

func (cf *CommonFlags) RegisterFlags(flags *pflag.FlagSet) {
	_ = cf.OnErrors.Set("stop")
	cf.OnErrors.RegisterFlags(flags, "")
	flags.BoolVar(&cf.DryRun, "dry-run", false, "dry run")
}

func New(ctx context.Context, cmd *cobra.Command, cm *config.ConfigurationManager) *Application {
	// application's context
	app := &Application{
		log:    &Log{},
		tz:     time.Local,
		Config: cm,
	}
	app.RegisterFlags(cmd.PersistentFlags())
	AddLogFlags(ctx, cmd, app)
	return app
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

func (app *Application) Client() *Client {
	return &app.client
}

func (app *Application) Jnl() *fileevent.Recorder {
	return app.jnl
}

func (app *Application) SetJnl(jnl *fileevent.Recorder) {
	app.jnl = jnl
}

func (app *Application) Log() *Log {
	return app.log
}

func (app *Application) SetLog(log *Log) {
	app.log = log
}

func ChainRunEFunctions(prev RunE, fn RunEAdaptor, ctx context.Context, cmd *cobra.Command, app *Application) RunE {
	if prev == nil {
		return func(cmd *cobra.Command, args []string) error {
			return fn(ctx, cmd, app)
		}
	}
	return func(cmd *cobra.Command, args []string) error {
		if prev != nil {
			err := prev(cmd, args)
			if err != nil {
				return err
			}
		}
		return fn(ctx, cmd, app)
	}
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
