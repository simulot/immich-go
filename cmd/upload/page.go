package upload

import (
	"fmt"
	"io"
	"log/slog"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/navidys/tvxwidgets"
	"github.com/rivo/tview"
	"github.com/simulot/immich-go/helpers/fileevent"
)

type page struct {
	app           *UpCmd
	screen        *tview.Grid
	footer        *tview.Grid
	prepareCounts *tview.Grid
	uploadCounts  *tview.Grid
	logView       *tview.TextView
	counts        map[fileevent.Code]*tview.TextView
	prevSlog      *slog.Logger
	// prevLogFile   io.WriteCloser

	immichReading *tvxwidgets.PercentageModeGauge
	immichPrepare *tvxwidgets.PercentageModeGauge
	immichUpload  *tvxwidgets.PercentageModeGauge

	page     *tview.Application
	quitting chan any
}

func (app *UpCmd) newPage() *page {
	p := &page{
		app:      app,
		quitting: make(chan any),
		counts:   map[fileevent.Code]*tview.TextView{},
	}
	return p
}

func (p *page) Page() *tview.Application {
	app := tview.NewApplication()

	p.screen = tview.NewGrid()

	p.screen.AddItem(tview.NewTextView().SetText(p.app.Banner.String()), 0, 0, 1, 1, 0, 0, false)

	p.prepareCounts = tview.NewGrid()
	p.prepareCounts.SetBorder(true).SetTitle("Input analysis")

	p.addCounter(p.prepareCounts, 0, "Images", fileevent.DiscoveredImage)
	p.addCounter(p.prepareCounts, 1, "Videos", fileevent.DiscoveredVideo)
	p.addCounter(p.prepareCounts, 2, "Metadata files", fileevent.DiscoveredSidecar)
	p.addCounter(p.prepareCounts, 3, "Discarded files", fileevent.DiscoveredDiscarded)
	p.addCounter(p.prepareCounts, 4, "Unsupported files", fileevent.DiscoveredUnsupported)
	p.addCounter(p.prepareCounts, 5, "Duplicates in the input", fileevent.AnalysisLocalDuplicate)
	p.addCounter(p.prepareCounts, 6, "Files with a sidecar", fileevent.AnalysisAssociatedMetadata)
	p.addCounter(p.prepareCounts, 7, "Files without sidecar", fileevent.AnalysisMissingAssociatedMetadata)

	p.prepareCounts.SetSize(8, 2, 1, 1).SetColumns(30, 10)

	p.uploadCounts = tview.NewGrid()
	p.uploadCounts.SetBorder(true).SetTitle("Uploading")

	p.addCounter(p.uploadCounts, 0, "Files uploaded", fileevent.Uploaded)
	p.addCounter(p.uploadCounts, 1, "Errors during upload", fileevent.UploadServerError)
	p.addCounter(p.uploadCounts, 2, "Files not selected", fileevent.UploadNotSelected)
	p.addCounter(p.uploadCounts, 3, "Server's asset upgraded", fileevent.UploadUpgraded)
	p.addCounter(p.uploadCounts, 4, "Server has same quality", fileevent.UploadServerDuplicate)
	p.addCounter(p.uploadCounts, 5, "Server has better quality", fileevent.UploadServerBetter)
	p.uploadCounts.SetSize(6, 2, 1, 1).SetColumns(30, 10)

	counts := tview.NewGrid()
	counts.Box = tview.NewBox()

	counts.AddItem(p.prepareCounts, 0, 0, 1, 1, 0, 0, false)
	counts.AddItem(p.uploadCounts, 0, 1, 1, 1, 0, 0, false)
	counts.SetSize(1, 2, 15, 40)

	p.screen.AddItem(counts, 1, 0, 1, 1, 0, 0, false)

	// Hijack the log
	p.logView = tview.NewTextView().SetMaxLines(5).ScrollToEnd()
	p.prevSlog = p.app.SharedFlags.Log

	if p.app.SharedFlags.LogWriterCloser != nil {
		w := io.MultiWriter(p.app.SharedFlags.LogWriterCloser, p.logView)
		p.app.SetLogWriter(w)
	} else {
		p.app.SetLogWriter(p.logView)
	}
	p.app.SharedFlags.Jnl.SetLogger(p.app.SharedFlags.Log)
	p.logView.SetBorder(true).SetTitle("Log")
	p.screen.AddItem(p.logView, 2, 0, 1, 1, 0, 0, false)

	p.immichReading = tvxwidgets.NewPercentageModeGauge()
	p.immichReading.SetRect(0, 0, 50, 1)
	p.immichReading.SetMaxValue(0)
	p.immichReading.SetValue(0)

	p.immichPrepare = tvxwidgets.NewPercentageModeGauge()
	p.immichPrepare.SetRect(0, 0, 50, 1)
	p.immichPrepare.SetMaxValue(0)
	p.immichPrepare.SetValue(0)

	p.immichUpload = tvxwidgets.NewPercentageModeGauge()
	p.immichUpload.SetRect(0, 0, 50, 1)
	p.immichUpload.SetMaxValue(0)
	p.immichUpload.SetValue(0)

	p.footer = tview.NewGrid()
	p.footer.AddItem(tview.NewTextView().SetText("Immich content:").SetTextAlign(tview.AlignCenter), 0, 0, 1, 1, 0, 0, false).AddItem(p.immichReading, 0, 1, 1, 1, 0, 0, false)
	if p.app.GooglePhotos {
		p.footer.AddItem(tview.NewTextView().SetText("Google Photo puzzle:").SetTextAlign(tview.AlignCenter), 0, 2, 1, 1, 0, 0, false).AddItem(p.immichPrepare, 0, 3, 1, 1, 0, 0, false)
		p.footer.AddItem(tview.NewTextView().SetText("Uploading:").SetTextAlign(tview.AlignCenter), 0, 4, 1, 1, 0, 0, false).AddItem(p.immichUpload, 0, 5, 1, 1, 0, 0, false)
		p.footer.SetColumns(25, 0, 25, 0, 25, 0)
	} else {
		p.footer.SetColumns(25, 0)
	}
	// p.footer.SetColumns()
	p.screen.AddItem(p.footer, 3, 0, 1, 1, 0, 0, false)

	// Adjust section's height
	p.screen.SetRows(4, 10, 0, 1)

	app.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlQ, tcell.KeyCtrlC:
			close(p.quitting)
			app.Stop()
			p.app.Log = p.prevSlog
		}
		return event
	})
	go func() {
		t := time.NewTicker(100 * time.Millisecond)
		for {
			select {
			case <-p.quitting:
				t.Stop()
				return
			case <-t.C:
				p.page.QueueUpdateDraw(p.draw)
			}
		}
	}()

	p.page = app
	return app.SetRoot(p.screen, true)
}

type progressUpdate func(value, max int)

func (p *page) updateImmichReading(value, total int) {
	p.immichReading.SetMaxValue(total)
	p.immichReading.SetValue(value)
}

func (p *page) getCountView(c fileevent.Code, count int64) *tview.TextView {
	v, ok := p.counts[c]
	if !ok {
		v = tview.NewTextView()
		p.counts[c] = v
	}
	v.SetText(fmt.Sprintf("%6d", count))
	return v
}

func (p *page) draw() {
	counts := p.app.Jnl.GetCounts()
	for c := range p.counts {
		p.getCountView(c, counts[c])
	}
	if p.app.GooglePhotos {
		p.immichPrepare.SetMaxValue(int(counts[fileevent.DiscoveredImage] + counts[fileevent.DiscoveredVideo]))
		p.immichPrepare.SetValue(int(counts[fileevent.AnalysisAssociatedMetadata]))

		p.immichUpload.SetMaxValue(int(counts[fileevent.DiscoveredImage] + counts[fileevent.DiscoveredVideo]))
		p.immichUpload.SetValue(int(counts[fileevent.UploadNotSelected] +
			counts[fileevent.UploadUpgraded] +
			counts[fileevent.UploadServerDuplicate] +
			counts[fileevent.UploadServerBetter] +
			counts[fileevent.Uploaded]))
	}
}

func (p *page) addCounter(g *tview.Grid, row int, label string, counter fileevent.Code) {
	g.AddItem(tview.NewTextView().SetText(label), row, 0, 1, 1, 0, 0, false)
	g.AddItem(p.getCountView(counter, 0), row, 1, 1, 1, 0, 0, false)
}
