package app

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"time"

	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/config"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/filetypes"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// Application holds configuration used by all commands
// It manages global settings like:
// - the log and the log-level
// - application counters
// - the concurrency
// - the configuration file

type Application struct {
	// CLI flags
	DryRun         bool
	OnErrors       cliflags.OnErrorsFlag
	SaveConfig     bool
	ConcurrentTask int
	CfgFile        string

	// Internal state
	log    *Log
	jnl    *fileevent.Recorder
	tz     *time.Location
	Config *config.ConfigurationManager

	sm filetypes.SupportedMedia

	numErrors atomic.Int64 // count the errors occurred during the run
}

func (app *Application) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVar(&app.CfgFile, "config", "", "config file (default is ./immich-go.yaml)")
	flags.BoolVar(&app.DryRun, "dry-run", false, "dry run")
	flags.BoolVar(&app.SaveConfig, "save-config", false, "Save the configuration to immich-go.yaml")
	flags.Var(&app.OnErrors, "on-errors", "What to do when an error occurs (stop, continue, accept N errors at max)")
	flags.IntVar(&app.ConcurrentTask, "concurrent-tasks", runtime.NumCPU(), "Number of concurrent tasks (1-20)")
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

func (app *Application) GetSupportedMedia() filetypes.SupportedMedia {
	if app.sm == nil {
		return filetypes.DefaultSupportedMedia
	}
	return app.sm
}

func (app *Application) SetSupportedMedia(sm filetypes.SupportedMedia) {
	app.sm = sm
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
// Test comment
// Test comment
// Test comment
