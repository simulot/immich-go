package app

import (
	"context"
	"errors"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/simulot/immich-go/internal/ui/runner"

	cliflags "github.com/simulot/immich-go/internal/cliFlags"
	"github.com/simulot/immich-go/internal/config"
	"github.com/simulot/immich-go/internal/fileprocessor"
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
	DryRun             bool
	OnErrors           cliflags.OnErrorsFlag
	SaveConfig         bool
	ConcurrentTask     int
	CfgFile            string
	UIMode             runner.Mode
	UIExperimental     bool
	UILegacy           bool
	UIEventBuffer      int
	UIJobsPollInterval time.Duration
	UIDumpEvents       bool

	// Internal state
	log       *Log
	processor *fileprocessor.FileProcessor // Unified file processing tracker
	tz        *time.Location
	Config    *config.ConfigurationManager

	sm filetypes.SupportedMedia

	numErrors atomic.Int64 // count the errors occurred during the run
}

func (app *Application) RegisterFlags(flags *pflag.FlagSet) {
	flags.StringVar(&app.CfgFile, "config", "", "config file (default is ./immich-go.yaml)")
	flags.BoolVar(&app.DryRun, "dry-run", false, "dry run")
	flags.BoolVar(&app.SaveConfig, "save-config", false, "Save the configuration to immich-go.yaml")
	flags.Var(&app.OnErrors, "on-errors", "What to do when an error occurs (stop, continue, accept N errors at max)")
	flags.IntVar(&app.ConcurrentTask, "concurrent-tasks", runtime.NumCPU(), "Number of concurrent tasks (1-20)")
	flags.StringVar((*string)(&app.UIMode), "ui", string(runner.ModeAuto), "UI mode for experimental interface (auto, terminal, web, native, off)")
	_ = flags.MarkHidden("ui")
	flags.BoolVar(&app.UIExperimental, "tui-experimental", false, "Enable the experimental Bubble Tea interface")
	flags.BoolVar(&app.UILegacy, "tui-legacy", false, "Force the legacy tcell UI even when new UI becomes default")
	flags.IntVar(&app.UIEventBuffer, "ui-event-buffer", 256, "Size of the buffered channel used to stream UI events")
	flags.BoolVar(&app.UIDumpEvents, "ui-dump-events", false, "Log every experimental UI event for debugging")
	_ = flags.MarkHidden("ui-dump-events")
}

func New(ctx context.Context, cmd *cobra.Command) *Application {
	// application's context
	a := &Application{
		log:                &Log{},
		tz:                 time.Local,
		Config:             config.New(),
		UIMode:             runner.ModeAuto,
		UIEventBuffer:      256,
		UIJobsPollInterval: 250 * time.Millisecond,
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

// FileProcessor returns the file processor for coordinated asset tracking and event logging
func (app *Application) FileProcessor() *fileprocessor.FileProcessor {
	return app.processor
}

// SetFileProcessor sets the file processor
func (app *Application) SetFileProcessor(processor *fileprocessor.FileProcessor) {
	app.processor = processor
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
