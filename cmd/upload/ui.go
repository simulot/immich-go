package upload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/navidys/tvxwidgets"
	"github.com/rivo/tview"
	"github.com/simulot/immich-go/helpers/fileevent"
	"golang.org/x/sync/errgroup"
)

type uiPage struct {
	// app            *UpCmd
	screen         *tview.Grid
	footer         *tview.Grid
	prepareCounts  *tview.Grid
	uploadCounts   *tview.Grid
	serverJobs     *tvxwidgets.Sparkline
	logView        *tview.TextView
	counts         map[fileevent.Code]*tview.TextView
	prevSlog       *slog.Logger
	serverActivity []float64
	// prevLogFile   io.WriteCloser
	lastTimeServerActive atomic.Int64

	immichReading *tvxwidgets.PercentageModeGauge
	immichPrepare *tvxwidgets.PercentageModeGauge
	immichUpload  *tvxwidgets.PercentageModeGauge

	// page      *tview.Application
	watchJobs bool
	// quitting  chan any
}

func (app *UpCmd) runUI(ctx context.Context) error {
	ctx, cancel := context.WithCancelCause(ctx)

	uiApp := tview.NewApplication()
	ui := newUI(ctx, app)

	defer cancel(nil)
	pages := tview.NewPages()

	var preparationDone atomic.Bool
	var uploadDone atomic.Bool
	var uiGroup errgroup.Group
	var messages strings.Builder

	uiApp.SetRoot(pages, true)

	stopUI := func(err error) {
		cancel(err)
		if uiApp != nil {
			uiApp.Stop()
		}
	}

	pages.AddPage("ui", ui.screen, true, true)

	// handle Ctrl+C and Ctrl+Q
	uiApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlQ, tcell.KeyCtrlC:
			app.Log = ui.prevSlog
			cancel(errors.New("interrupted: Ctrl+C or Ctrl+Q pressed"))
		case tcell.KeyEnter:
			if uploadDone.Load() {
				stopUI(nil)
			}
		}
		return event
	})

	// update server status
	if ui.watchJobs {
		go func() {
			tick := time.NewTicker(250 * time.Millisecond)
			for {
				select {
				case <-ctx.Done():
					tick.Stop()
					return
				case <-tick.C:
					jobs, err := app.Immich.GetJobs(ctx)
					if err == nil {
						jobCount := 0
						jobWaiting := 0
						for _, j := range jobs {
							jobCount += j.JobCounts.Active
							jobWaiting += j.JobCounts.Waiting
						}
						_, _, w, _ := ui.serverJobs.GetInnerRect()
						ui.serverActivity = append(ui.serverActivity, float64(jobCount))
						if len(ui.serverActivity) > w {
							ui.serverActivity = ui.serverActivity[1:]
						}
						ui.serverJobs.SetData(ui.serverActivity)
						ui.serverJobs.SetTitle(fmt.Sprintf("Server's jobs: active: %d, waiting: %d", jobCount, jobWaiting))
						if jobCount > 0 {
							ui.lastTimeServerActive.Store(time.Now().Unix())
						}
					}
				}
			}
		}()
	}

	// force the ui to redraw counters
	go func() {
		tick := time.NewTicker(100 * time.Millisecond)
		for {
			select {
			case <-ctx.Done():
				tick.Stop()
				return
			case <-tick.C:
				uiApp.QueueUpdateDraw(func() {
					counts := app.Jnl.GetCounts()
					for c := range ui.counts {
						ui.getCountView(c, counts[c])
					}
					if app.GooglePhotos {
						ui.immichPrepare.SetMaxValue(int(app.Jnl.TotalAssets()))
						ui.immichPrepare.SetValue(int(app.Jnl.TotalProcessedGP()))

						if preparationDone.Load() {
							ui.immichUpload.SetMaxValue(int(app.Jnl.TotalAssets()))
						}
						ui.immichUpload.SetValue(int(app.Jnl.TotalProcessed()))
					}
				})
			}
		}
	}()

	// start the UI
	uiGroup.Go(func() error {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			err := uiApp.Run()
			cancel(err)
			return err
		}
	})

	// start the processes
	uiGroup.Go(func() error {
		processGrp := errgroup.Group{}
		processGrp.Go(func() error {
			// Get immich asset
			err := app.getImmichAssets(ctx, ui.updateImmichReading)
			if err != nil {
				stopUI(err)
			}
			return err
		})
		processGrp.Go(func() error {
			err := app.getImmichAlbums(ctx)
			if err != nil {
				stopUI(err)
			}
			return err
		})
		processGrp.Go(func() error {
			// Run Prepare
			err := app.browser.Prepare(ctx)
			if err != nil {
				stopUI(err)
			}
			return err
		})

		// Wait the end of the preparation: immich assets, albums and first browsing
		err := processGrp.Wait()
		if err != nil {
			return context.Cause(ctx)
		}
		preparationDone.Store(true)

		// we can upload assets
		err = app.uploadLoop(ctx)
		if err != nil {
			return context.Cause(ctx)
		}
		uploadDone.Store(true)
		counts := app.Jnl.GetCounts()
		if counts[fileevent.Error]+counts[fileevent.UploadServerError] > 0 {
			messages.WriteString("Some errors have occurred. Look at the log file for details\n")
		}
		if app.GooglePhotos && counts[fileevent.AnalysisMissingAssociatedMetadata] > 0 {
			messages.WriteString(fmt.Sprintf("\n%d JSON files are missing.\n", counts[fileevent.AnalysisMissingAssociatedMetadata]))
			messages.WriteString("- have you processed all takeout parts together?\n")
			messages.WriteString("- ask for another takeout, or ask one year at a time.\n")
		}

		modal := newModal(messages.String())
		pages.AddPage("modal", modal, true, false)
		// upload is done!
		pages.ShowPage("modal")

		return err
	})

	// Wait for termination of UI processes
	err := uiGroup.Wait()
	if err != nil {
		err = context.Cause(ctx)
	}

	// Time to leave
	app.Jnl.Report()
	if messages.Len() > 0 {
		return (errors.New(messages.String()))
	}
	return err
}

func newModal(message string) tview.Primitive {
	message += "\nYou can quit the program safely.\n\nPress the [enter] key to exit."
	lines := strings.Count(message, "\n")
	// Returns a new primitive which puts the provided primitive in the center and
	// sets its size to the given width and height.
	modal := func(p tview.Primitive, width, height int) tview.Primitive {
		return tview.NewFlex().
			AddItem(nil, 0, 1, false).
			AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
				AddItem(nil, 0, 1, false).
				AddItem(p, height, 1, true).
				AddItem(nil, 0, 1, false), width, 1, true).
			AddItem(nil, 0, 1, false)
	}
	text := tview.NewTextView().SetText(message)
	box := tview.NewBox().
		SetBorder(true).
		SetTitle("Upload completed")
	text.Box = box
	return modal(text, 80, 2+lines)
}

func newUI(ctx context.Context, app *UpCmd) *uiPage {
	ui := &uiPage{
		counts: map[fileevent.Code]*tview.TextView{},
	}

	ui.screen = tview.NewGrid()

	ui.screen.AddItem(tview.NewTextView().SetText(app.Banner.String()), 0, 0, 1, 1, 0, 0, false)

	ui.prepareCounts = tview.NewGrid()
	ui.prepareCounts.SetBorder(true).SetTitle("Input analysis")

	ui.addCounter(ui.prepareCounts, 0, "Images", fileevent.DiscoveredImage)
	ui.addCounter(ui.prepareCounts, 1, "Videos", fileevent.DiscoveredVideo)
	ui.addCounter(ui.prepareCounts, 2, "Metadata files", fileevent.DiscoveredSidecar)
	ui.addCounter(ui.prepareCounts, 3, "Discarded files", fileevent.DiscoveredDiscarded)
	ui.addCounter(ui.prepareCounts, 4, "Unsupported files", fileevent.DiscoveredUnsupported)
	ui.addCounter(ui.prepareCounts, 5, "Duplicates in the input", fileevent.AnalysisLocalDuplicate)
	ui.addCounter(ui.prepareCounts, 6, "Files with a sidecar", fileevent.AnalysisAssociatedMetadata)
	ui.addCounter(ui.prepareCounts, 7, "Files without sidecar", fileevent.AnalysisMissingAssociatedMetadata)

	ui.prepareCounts.SetSize(8, 2, 1, 1).SetColumns(30, 10)

	ui.uploadCounts = tview.NewGrid()
	ui.uploadCounts.SetBorder(true).SetTitle("Uploading")

	ui.addCounter(ui.uploadCounts, 0, "Files uploaded", fileevent.Uploaded)
	ui.addCounter(ui.uploadCounts, 1, "Errors during upload", fileevent.UploadServerError)
	ui.addCounter(ui.uploadCounts, 2, "Files not selected", fileevent.UploadNotSelected)
	ui.addCounter(ui.uploadCounts, 3, "Server's asset upgraded", fileevent.UploadUpgraded)
	ui.addCounter(ui.uploadCounts, 4, "Server has same quality", fileevent.UploadServerDuplicate)
	ui.addCounter(ui.uploadCounts, 5, "Server has better quality", fileevent.UploadServerBetter)
	ui.uploadCounts.SetSize(6, 2, 1, 1).SetColumns(30, 10)

	if _, err := app.Immich.GetJobs(ctx); err == nil {
		ui.watchJobs = true

		ui.serverJobs = tvxwidgets.NewSparkline()
		ui.serverJobs.SetBorder(true).SetTitle("Server pending jobs")
		ui.serverJobs.SetData(ui.serverActivity)
		ui.serverJobs.SetDataTitleColor(tcell.ColorDarkOrange)
		ui.serverJobs.SetLineColor(tcell.ColorSteelBlue)
	}

	counts := tview.NewGrid()
	counts.Box = tview.NewBox()
	counts.AddItem(ui.prepareCounts, 0, 0, 1, 1, 0, 0, false)
	counts.AddItem(ui.uploadCounts, 0, 1, 1, 1, 0, 0, false)
	if ui.watchJobs {
		counts.AddItem(ui.serverJobs, 0, 2, 1, 1, 0, 0, false)
	}
	counts.SetSize(1, 3, 15, 40)
	counts.SetColumns(40, 40, 0)

	ui.screen.AddItem(counts, 1, 0, 1, 1, 0, 0, false)

	// Hijack the log
	ui.logView = tview.NewTextView().SetMaxLines(100).ScrollToEnd()
	ui.prevSlog = app.SharedFlags.Log

	if app.SharedFlags.LogWriterCloser != nil {
		w := io.MultiWriter(app.SharedFlags.LogWriterCloser, ui.logView)
		app.SetLogWriter(w)
	} else {
		app.SetLogWriter(ui.logView)
	}
	app.SharedFlags.Jnl.SetLogger(app.SharedFlags.Log)
	ui.logView.SetBorder(true).SetTitle("Log")
	ui.screen.AddItem(ui.logView, 2, 0, 1, 1, 0, 0, false)

	ui.immichReading = tvxwidgets.NewPercentageModeGauge()
	ui.immichReading.SetRect(0, 0, 50, 1)
	ui.immichReading.SetMaxValue(0)
	ui.immichReading.SetValue(0)

	ui.immichPrepare = tvxwidgets.NewPercentageModeGauge()
	ui.immichPrepare.SetRect(0, 0, 50, 1)
	ui.immichPrepare.SetMaxValue(0)
	ui.immichPrepare.SetValue(0)

	ui.immichUpload = tvxwidgets.NewPercentageModeGauge()
	ui.immichUpload.SetRect(0, 0, 50, 1)
	ui.immichUpload.SetMaxValue(0)
	ui.immichUpload.SetValue(0)

	ui.footer = tview.NewGrid()
	ui.footer.AddItem(tview.NewTextView().SetText("Immich content:").SetTextAlign(tview.AlignCenter), 0, 0, 1, 1, 0, 0, false).AddItem(ui.immichReading, 0, 1, 1, 1, 0, 0, false)
	if app.GooglePhotos {
		ui.footer.AddItem(tview.NewTextView().SetText("Google Photo puzzle:").SetTextAlign(tview.AlignCenter), 0, 2, 1, 1, 0, 0, false).AddItem(ui.immichPrepare, 0, 3, 1, 1, 0, 0, false)
		ui.footer.AddItem(tview.NewTextView().SetText("Uploading:").SetTextAlign(tview.AlignCenter), 0, 4, 1, 1, 0, 0, false).AddItem(ui.immichUpload, 0, 5, 1, 1, 0, 0, false)
		ui.footer.SetColumns(25, 0, 25, 0, 25, 0)
	} else {
		ui.footer.SetColumns(25, 0)
	}
	ui.screen.AddItem(ui.footer, 3, 0, 1, 1, 0, 0, false)

	// Adjust section's height
	ui.screen.SetRows(4, 10, 0, 1)
	return ui
}

type progressUpdate func(value, max int)

// call back to get the progression
func (p *uiPage) updateImmichReading(value, total int) {
	if value == 0 && total == 0 {
		total, value = 100, 100
	}
	p.immichReading.SetMaxValue(total)
	p.immichReading.SetValue(value)
}

func (p *uiPage) getCountView(c fileevent.Code, count int64) *tview.TextView {
	v, ok := p.counts[c]
	if !ok {
		v = tview.NewTextView()
		p.counts[c] = v
	}
	v.SetText(fmt.Sprintf("%6d", count))
	return v
}

func (p *uiPage) addCounter(g *tview.Grid, row int, label string, counter fileevent.Code) {
	g.AddItem(tview.NewTextView().SetText(label), row, 0, 1, 1, 0, 0, false)
	g.AddItem(p.getCountView(counter, 0), row, 1, 1, 1, 0, 0, false)
}
