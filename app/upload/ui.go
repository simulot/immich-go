package upload

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/navidys/tvxwidgets"
	"github.com/rivo/tview"
	"github.com/simulot/immich-go/app"
	"github.com/simulot/immich-go/internal/assets"
	"github.com/simulot/immich-go/internal/assettracker"
	"github.com/simulot/immich-go/internal/fileevent"
	"github.com/simulot/immich-go/internal/fileprocessor"
	"github.com/simulot/immich-go/internal/ui/core/messages"
	"github.com/simulot/immich-go/internal/ui/core/state"
	"golang.org/x/sync/errgroup"
)

type uiPage struct {
	screen         *tview.Grid
	footer         *tview.Grid
	discoveryZone  *tview.Grid // NEW: Discovery zone
	processingZone *tview.Grid // NEW: Processing events zone
	statusZone     *tview.Grid // NEW: Asset processing status zone
	serverJobs     *tvxwidgets.Sparkline
	logView        *tview.TextView
	counts         map[fileevent.Code]*tview.TextView
	sizes          map[fileevent.Code]*tview.TextView // Size views for discovery events

	// Status zone views (separate from fileevent counters)
	statusViews map[string]*tview.TextView

	// Discovery zone views for total row
	discoveryViews map[string]*tview.TextView

	// File processor reference for event sizes
	fileProcessor *fileprocessor.FileProcessor

	// Asset tracker reference for status updates
	tracker *assettracker.AssetTracker

	// server's activity history
	serverActivity []float64

	// detect when the server is idling
	lastTimeServerActive atomic.Int64

	// gauges
	immichReading *tvxwidgets.PercentageModeGauge
	immichPrepare *tvxwidgets.PercentageModeGauge
	immichUpload  *tvxwidgets.PercentageModeGauge

	watchJobs bool
}

func (ui *uiPage) highJackLogger(app *app.Application) {
	ui.logView.SetDynamicColors(true)
	app.FileProcessor().Logger().SetLogger(app.Log().SetLogWriter(tview.ANSIWriter(ui.logView)))
}

func (ui *uiPage) restoreLogger(app *app.Application) {
	app.FileProcessor().Logger().SetLogger(app.Log().SetLogWriter(nil))
}

func (uc *UpCmd) runUI(ctx context.Context, app *app.Application) error {
	ctx, cancel := context.WithCancelCause(ctx)
	uiApp := tview.NewApplication()
	ui := uc.newUI(ctx, app)

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
			ui.restoreLogger(app)
			cancel(errors.New("interrupted: Ctrl+C or Ctrl+Q pressed"))
		case tcell.KeyEnter:
			if uploadDone.Load() {
				stopUI(nil)
			}
		}
		return event
	})

	uc.startLegacyUIEventConsumer(ctx, uiApp, ui)

	// update server status via legacy poller when stream not available
	if ui.watchJobs && uc.uiStream == nil {
		go func() {
			tick := time.NewTicker(250 * time.Millisecond)
			for {
				select {
				case <-ctx.Done():
					tick.Stop()
					return
				case <-tick.C:
					jobs, err := uc.client.AdminImmich.GetJobs(ctx)
					if err == nil {
						jobCount := 0
						jobWaiting := 0
						for _, j := range jobs {
							jobCount += j.JobCounts.Active
							jobWaiting += j.JobCounts.Waiting
						}
						ui.updateJobSparkline(jobCount, jobWaiting)
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
					counts := app.FileProcessor().Logger().GetCounts()
					sizes := app.FileProcessor().Logger().GetEventSizes()
					for c := range ui.counts {
						ui.getCountView(c, counts[c])
						ui.updateSizeView(c, sizes[c])
					}
					// Update the processing status zone
					ui.updateStatusZone()
					if uc.Mode == UpModeGoogleTakeout {
						ui.immichPrepare.SetMaxValue(int(app.FileProcessor().Logger().TotalAssets()))
						// Calculate processed items for Google Takeout progress
						counts := app.FileProcessor().Logger().GetCounts()
						processedGP := counts[fileevent.ProcessedAssociatedMetadata] +
							counts[fileevent.ProcessedMissingMetadata]
						ui.immichPrepare.SetValue(int(processedGP))

						if preparationDone.Load() {
							ui.immichUpload.SetMaxValue(int(app.FileProcessor().Logger().TotalAssets()))
						}
						// ui.immichUpload.SetValue(int(app.Jnl().TotalProcessed(uc.takeoutOptions.KeepJSONLess)))
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
		var groupChan chan *assets.Group
		var err error

		processGrp := errgroup.Group{}
		processGrp.Go(func() error {
			// Get immich asset
			err = uc.getImmichAssets(ctx, ui.updateImmichReading)
			if err != nil {
				stopUI(err)
			}
			return err
		})
		processGrp.Go(func() error {
			err = uc.getImmichAlbums(ctx)
			if err != nil {
				stopUI(err)
			}
			return err
		})
		processGrp.Go(func() error {
			// Run Prepare
			groupChan = uc.adapter.Browse(ctx)
			return nil
		})

		// Wait the end of the preparation: immich assets, albums and first browsing
		err = processGrp.Wait()
		if err != nil {
			return context.Cause(ctx)
		}
		preparationDone.Store(true)

		// we can upload assets
		err = uc.uploadLoop(ctx, groupChan)
		// if err != nil {
		// 	return context.Cause(ctx)
		// }

		err = errors.Join(err, uc.finishing(ctx))

		uploadDone.Store(true)
		counts := app.FileProcessor().Logger().GetCounts()
		if counts[fileevent.ErrorUploadFailed]+counts[fileevent.ErrorServerError]+counts[fileevent.ErrorFileAccess]+counts[fileevent.ErrorIncomplete] > 0 {
			messages.WriteString("Some errors have occurred. Look at the log file for details\n")
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

func (uc *UpCmd) newUI(ctx context.Context, a *app.Application) *uiPage {
	ui := &uiPage{
		counts: map[fileevent.Code]*tview.TextView{},
		sizes:  map[fileevent.Code]*tview.TextView{},
	}

	ui.screen = tview.NewGrid()

	ui.screen.AddItem(tview.NewTextView().SetText(app.Banner()), 0, 0, 1, 1, 0, 0, false)

	ui.discoveryZone = ui.createDiscoveryZone()

	// Create the processing zone (shows processing events)
	ui.processingZone = ui.createProcessingZone()

	// Create the processing status zone (replaces upload counts in layout)
	ui.statusZone = ui.createStatusZone()

	// Set tracker reference for status updates
	if a.FileProcessor() != nil {
		ui.tracker = a.FileProcessor().Tracker()
		ui.fileProcessor = a.FileProcessor()
	}

	canWatchJobs := false
	if uc.client.AdminImmich != nil {
		if uc.uiStream != nil {
			canWatchJobs = true
		} else if _, err := uc.client.AdminImmich.GetJobs(ctx); err == nil {
			canWatchJobs = true
		}
	}
	if canWatchJobs {
		ui.watchJobs = true
		ui.initServerJobsView()
	}

	counts := tview.NewGrid()
	counts.Box = tview.NewBox()
	// Single row: Discovery, Processing, Upload Progress, Server Jobs
	counts.AddItem(ui.discoveryZone, 0, 0, 1, 1, 0, 0, false)
	counts.AddItem(ui.processingZone, 0, 1, 1, 1, 0, 0, false)
	counts.AddItem(ui.statusZone, 0, 2, 1, 1, 0, 0, false)
	if ui.watchJobs {
		counts.AddItem(ui.serverJobs, 0, 3, 1, 1, 0, 0, false)
	}
	counts.SetSize(1, 4, 15, 40)
	counts.SetColumns(40, 30, 40, 0)

	ui.screen.AddItem(counts, 1, 0, 1, 1, 0, 0, false)

	// Hijack the log
	ui.logView = tview.NewTextView().SetMaxLines(100).ScrollToEnd()
	ui.highJackLogger(a)

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

	if uc.Mode == UpModeGoogleTakeout {
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

type progressUpdate func(value, maxValue int)

// call back to get the progression
func (ui *uiPage) updateImmichReading(value, total int) {
	if value == 0 && total == 0 {
		total, value = 100, 100
	}
	ui.immichReading.SetMaxValue(total)
	ui.immichReading.SetValue(value)
}

func (ui *uiPage) getCountView(c fileevent.Code, count int64) *tview.TextView {
	v, ok := ui.counts[c]
	if !ok {
		v = tview.NewTextView()
		ui.counts[c] = v
	}
	v.SetText(fmt.Sprintf("%6d", count))
	return v
}

func (ui *uiPage) updateSizeView(c fileevent.Code, size int64) {
	v, ok := ui.sizes[c]
	if !ok {
		return
	}
	if size == 0 {
		v.SetText("0 B")
	} else {
		v.SetText(ui.formatBytes(size))
	}
}

func (ui *uiPage) addCounter(g *tview.Grid, row int, label string, counter fileevent.Code) {
	g.AddItem(tview.NewTextView().SetText(label), row, 0, 1, 1, 0, 0, false)
	g.AddItem(ui.getCountView(counter, 0), row, 1, 1, 1, 0, 0, false)
	g.AddItem(tview.NewTextView().SetText(""), row, 2, 1, 1, 0, 0, false) // Spacer

	// Create size view for discovery events
	sizeView := tview.NewTextView().SetText("0 B").SetTextAlign(tview.AlignRight)
	if ui.sizes == nil {
		ui.sizes = make(map[fileevent.Code]*tview.TextView)
	}
	ui.sizes[counter] = sizeView
	g.AddItem(sizeView, row, 3, 1, 1, 0, 0, false)
}

func (ui *uiPage) addProcessingCounter(g *tview.Grid, row int, label string, counter fileevent.Code) {
	g.AddItem(tview.NewTextView().SetText(label), row, 0, 1, 1, 0, 0, false)
	g.AddItem(ui.getCountView(counter, 0), row, 1, 1, 1, 0, 0, false)
}

func (ui *uiPage) initServerJobsView() {
	if ui.serverJobs != nil {
		return
	}
	ui.serverJobs = tvxwidgets.NewSparkline()
	ui.serverJobs.SetBorder(true).SetTitle("Server pending jobs")
	ui.serverJobs.SetData(ui.serverActivity)
	ui.serverJobs.SetDataTitleColor(tcell.ColorDarkOrange)
	ui.serverJobs.SetLineColor(tcell.ColorSteelBlue)
}

// createDiscoveryZone creates the discovery zone showing asset discovery events
func (ui *uiPage) createDiscoveryZone() *tview.Grid {
	discovery := tview.NewGrid()
	discovery.SetBorder(true).SetTitle("Discovery")

	// Row 0: Images
	ui.addCounter(discovery, 0, "Images", fileevent.DiscoveredImage)
	// Row 1: Videos
	ui.addCounter(discovery, 1, "Videos", fileevent.DiscoveredVideo)
	// Row 2: Empty row for spacing
	discovery.AddItem(tview.NewTextView().SetText(""), 2, 0, 1, 4, 0, 0, false)
	// Row 3: Duplicates (local)
	ui.addCounter(discovery, 3, "Duplicates (local)", fileevent.DiscardedLocalDuplicate)
	// Row 4: Already on server
	ui.addCounter(discovery, 4, "Already on server", fileevent.DiscardedServerDuplicate)
	// Row 5: Filtered (rules)
	ui.addCounter(discovery, 5, "Filtered (rules)", fileevent.DiscardedFiltered)
	// Row 6: Banned
	ui.addCounter(discovery, 6, "Banned", fileevent.DiscardedBanned)
	// Row 7: Missing sidecar
	ui.addCounter(discovery, 7, "Missing sidecar", fileevent.ProcessedMissingMetadata)
	// Row 8: Total discovered
	discovery.AddItem(tview.NewTextView().SetText("Total discovered"), 8, 0, 1, 1, 0, 0, false)
	ui.addDiscoveryCounter(discovery, 8, "discoveredCount", "discoveredSize")

	discovery.SetSize(9, 4, 1, 1).SetColumns(20, 8, 2, 10)
	return discovery
}

// createProcessingZone creates the processing zone showing processing events
func (ui *uiPage) createProcessingZone() *tview.Grid {
	processing := tview.NewGrid()
	processing.SetBorder(true).SetTitle("Processing")

	// Row 0: Sidecars associated
	ui.addProcessingCounter(processing, 0, "Sidecars associated", fileevent.ProcessedAssociatedMetadata)
	// Row 1: Added to albums
	ui.addProcessingCounter(processing, 1, "Added to albums", fileevent.ProcessedAlbumAdded)
	// Row 2: Stacked (bursts, raw+jpg)
	ui.addProcessingCounter(processing, 2, "Stacked", fileevent.ProcessedStacked)
	// Row 3: Tagged
	ui.addProcessingCounter(processing, 3, "Tagged", fileevent.ProcessedTagged)
	// Row 4: Metadata updated
	ui.addProcessingCounter(processing, 4, "Metadata updated", fileevent.ProcessedMetadataUpdated)

	processing.SetSize(5, 2, 1, 1).SetColumns(20, 10)
	return processing
}

// createStatusZone creates the asset processing status zone
func (ui *uiPage) createStatusZone() *tview.Grid {
	status := tview.NewGrid()
	status.SetBorder(true).SetTitle("Progress")

	// Row 0: Pending assets
	status.AddItem(tview.NewTextView().SetText("Pending"), 0, 0, 1, 1, 0, 0, false)
	ui.addStatusCounter(status, 0, "pendingCount", "pendingSize")

	// Row 1: Uploaded assets
	status.AddItem(tview.NewTextView().SetText("Processed"), 1, 0, 1, 1, 0, 0, false)
	ui.addStatusCounter(status, 1, "uploadedCount", "uploadedSize")

	// Row 2: Discarded assets
	status.AddItem(tview.NewTextView().SetText("Discarded"), 2, 0, 1, 1, 0, 0, false)
	ui.addStatusCounter(status, 2, "discardedCount", "discardedSize")

	// Row 3: Error assets
	status.AddItem(tview.NewTextView().SetText("Errors"), 3, 0, 1, 1, 0, 0, false)
	ui.addStatusCounter(status, 3, "errorCount", "errorSize")

	// Row 4: Total
	status.AddItem(tview.NewTextView().SetText("Total"), 4, 0, 1, 1, 0, 0, false)
	ui.addStatusCounter(status, 4, "totalCount", "totalSize")

	status.SetSize(5, 4, 1, 1).SetColumns(20, 8, 2, 10)
	return status
}

// addStatusCounter adds count and size views for a status category
func (ui *uiPage) addStatusCounter(g *tview.Grid, row int, countKey, sizeKey string) {
	countView := tview.NewTextView().SetText("0").SetTextAlign(tview.AlignRight)
	sizeView := tview.NewTextView().SetText("0 B").SetTextAlign(tview.AlignRight)

	// Store references for updates
	if ui.statusViews == nil {
		ui.statusViews = make(map[string]*tview.TextView)
	}
	ui.statusViews[countKey] = countView
	ui.statusViews[sizeKey] = sizeView

	g.AddItem(countView, row, 1, 1, 1, 0, 0, false)
	g.AddItem(tview.NewTextView().SetText(""), row, 2, 1, 1, 0, 0, false) // Spacer
	g.AddItem(sizeView, row, 3, 1, 1, 0, 0, false)
}

// addDiscoveryCounter adds count and size views for discovery zone total
func (ui *uiPage) addDiscoveryCounter(g *tview.Grid, row int, countKey, sizeKey string) {
	countView := tview.NewTextView().SetText("0").SetTextAlign(tview.AlignRight)
	sizeView := tview.NewTextView().SetText("0 B").SetTextAlign(tview.AlignRight)

	// Store references for updates
	if ui.discoveryViews == nil {
		ui.discoveryViews = make(map[string]*tview.TextView)
	}
	ui.discoveryViews[countKey] = countView
	ui.discoveryViews[sizeKey] = sizeView

	g.AddItem(countView, row, 1, 1, 1, 0, 0, false)
	g.AddItem(tview.NewTextView().SetText(""), row, 2, 1, 1, 0, 0, false) // Spacer
	g.AddItem(sizeView, row, 3, 1, 1, 0, 0, false)
}

// updateStatusZone updates the status zone with current asset tracker data
func (ui *uiPage) updateStatusZone() {
	if ui.tracker == nil {
		return
	}

	// Get current counters
	pendingCount := ui.tracker.GetPendingCount()
	pendingSize := ui.tracker.GetPendingSize()
	processedCount := ui.tracker.GetProcessedCount()
	processedSize := ui.tracker.GetProcessedSize()
	discardedCount := ui.tracker.GetDiscardedCount()
	discardedSize := ui.tracker.GetDiscardedSize()
	errorCount := ui.tracker.GetErrorCount()
	errorSize := ui.tracker.GetErrorSize()

	// Calculate totals
	totalCount := pendingCount + processedCount + discardedCount + errorCount
	totalSize := pendingSize + processedSize + discardedSize + errorSize

	// Update the status views
	ui.statusViews["pendingCount"].SetText(fmt.Sprintf("%6d", pendingCount))
	ui.statusViews["pendingSize"].SetText(ui.formatBytes(pendingSize))
	ui.statusViews["uploadedCount"].SetText(fmt.Sprintf("%6d", processedCount))
	ui.statusViews["uploadedSize"].SetText(ui.formatBytes(processedSize))
	ui.statusViews["discardedCount"].SetText(fmt.Sprintf("%6d", discardedCount))
	ui.statusViews["discardedSize"].SetText(ui.formatBytes(discardedSize))
	ui.statusViews["errorCount"].SetText(fmt.Sprintf("%6d", errorCount))
	ui.statusViews["errorSize"].SetText(ui.formatBytes(errorSize))
	ui.statusViews["totalCount"].SetText(fmt.Sprintf("%6d", totalCount))
	ui.statusViews["totalSize"].SetText(ui.formatBytes(totalSize))

	// Update discovery zone total
	if ui.discoveryViews != nil {
		ui.discoveryViews["discoveredCount"].SetText(fmt.Sprintf("%6d", totalCount))
		ui.discoveryViews["discoveredSize"].SetText(ui.formatBytes(totalSize))
	}
}

// formatBytes formats byte count as human-readable string
func (ui *uiPage) formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d  B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func (uc *UpCmd) startLegacyUIEventConsumer(ctx context.Context, uiApp *tview.Application, ui *uiPage) {
	if uc.uiStream == nil {
		return
	}
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case evt, ok := <-uc.uiStream:
				if !ok {
					return
				}
				uc.dispatchLegacyUIEvent(uiApp, ui, evt)
			}
		}
	}()
}

func (uc *UpCmd) dispatchLegacyUIEvent(uiApp *tview.Application, ui *uiPage, evt messages.Event) {
	uiApp.QueueUpdateDraw(func() {
		switch evt.Type {
		case messages.EventLogLine:
			if entry, ok := evt.Payload.(state.LogEvent); ok {
				ui.appendUILogEntry(entry)
			}
		case messages.EventJobsUpdated:
			if summaries, ok := evt.Payload.([]state.JobSummary); ok {
				ui.applyJobSummaries(summaries)
			}
		}
	})
}

func (ui *uiPage) appendUILogEntry(entry state.LogEvent) {
	if ui.logView == nil {
		return
	}
	timestamp := entry.Timestamp.Format("15:04:05")
	line := fmt.Sprintf("[%s] %-5s %s", timestamp, strings.ToUpper(entry.Level), entry.Message)
	if len(entry.Details) > 0 {
		keys := make([]string, 0, len(entry.Details))
		for k := range entry.Details {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		pairs := make([]string, 0, len(keys))
		for _, k := range keys {
			pairs = append(pairs, fmt.Sprintf("%s=%s", k, entry.Details[k]))
		}
		line = fmt.Sprintf("%s (%s)", line, strings.Join(pairs, " "))
	}
	fmt.Fprintln(ui.logView, line)
}

func (ui *uiPage) applyJobSummaries(jobs []state.JobSummary) {
	if len(jobs) == 0 {
		return
	}
	active := 0
	waiting := 0
	for _, job := range jobs {
		active += job.Active
		waiting += job.Waiting
	}
	ui.updateJobSparkline(active, waiting)
}

func (ui *uiPage) updateJobSparkline(active, waiting int) {
	if ui.serverJobs == nil {
		return
	}
	_, _, w, _ := ui.serverJobs.GetInnerRect()
	ui.serverActivity = append(ui.serverActivity, float64(active))
	if w > 0 && len(ui.serverActivity) > w {
		ui.serverActivity = ui.serverActivity[len(ui.serverActivity)-w:]
	}
	ui.serverJobs.SetData(ui.serverActivity)
	ui.serverJobs.SetTitle(fmt.Sprintf("Server's jobs: active: %d, waiting: %d", active, waiting))
	if active > 0 {
		ui.lastTimeServerActive.Store(time.Now().Unix())
	}
}
