package upload

import "github.com/rivo/tview"

type page struct {
	app *UpCmd
}

func (app *UpCmd) newPage() *page {
	return &page{app: app}
}

func (p *page) view() tview.Primitive {
	return tview.NewGrid().SetRows(3,0,1).
		AddItem(tview.NewTextView(p.app.SharedFlags.B)

}
 