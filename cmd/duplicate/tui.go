package duplicate

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/key"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/ui"
)

type DuplicateModel struct {
	receivedPct int
	receivedDup int
	list        list.Model
	ready       bool

	app *DuplicateCmd

	width, height int
	err           error
}

type (
	msgReceivePct int
	msgDuplicate  int
	msgError      struct {
		Err error
	}
)

const bannerHeight = 6

var _ tea.Model = (*DuplicateModel)(nil)

func NewDuplicateModel(app *DuplicateCmd, keys []duplicateKey) DuplicateModel {
	l := list.New([]list.Item{}, list.NewDefaultDelegate(), 80, 15)
	l.KeyMap.NextPage = key.NewBinding(
		key.WithKeys("l", "pgdown", "f", "d"),
		key.WithHelp("l/pgdn", "next page"),
	)
	l.KeyMap.PrevPage = key.NewBinding(
		key.WithKeys("h", "pgup", "b", "u"),
		key.WithHelp("h/pgup", "prev page"),
	)
	m := DuplicateModel{
		app:  app,
		list: l,
	}
	m.adjustListTitle(false)
	return m
}

func (m DuplicateModel) Init() tea.Cmd {
	m.list.SetSpinner(spinner.Dot)
	return m.list.StartSpinner()
}

func (m DuplicateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// send the event to the table
	m.list, cmd = m.list.Update(msg)
	cmds = append(cmds, cmd)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.list.SetWidth(m.width)
		m.list.SetHeight(m.height - bannerHeight)
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.err = errors.New("interrupted by the user")
			cmds = append(cmds, tea.Quit)
		}

	case msgError:
		m.err = msg.Err
		cmds = append(cmds, tea.Quit)

	case msgReceivePct:
		m.receivedPct = int(msg)
		m.list = m.adjustListTitle(false)
	case msgDuplicate:
		m.receivedDup = int(msg)
		m.list = m.adjustListTitle(false)
	case []duplicateKey:
		// the table is ready
		m.list, cmd = m.loadTable(msg)
		m.ready = true
		cmds = append(cmds, cmd)

		// case duplicateKey:
		// 	// an item is highlighted
		// 	m.sideList, cmd = m.selectKey(msg)
		// 	cmds = append(cmds, cmd)
	}
	return m, tea.Batch(cmds...)
}

func (m DuplicateModel) View() string {
	b := strings.Builder{}
	row := 0
	if m.height > 10 {
		b.WriteString(m.app.SharedFlags.Banner)
		row += 5
	}
	b.WriteString(m.list.View())
	_ = row
	return b.String()
}

// duplicateItem is a list item.
type duplicateItem struct {
	key    duplicateKey    // Duplicate key
	assets []*immich.Asset // List of duplicates
	keep   int
}

func (i duplicateItem) Title() string {
	return fmt.Sprintf("%s %s", i.key.Date.Format("2006.01.02 15:04:05 Z07:00"), i.key.Name)
}

func (i duplicateItem) Description() string {
	b := strings.Builder{}
	for j := range i.assets {
		if j > 0 {
			b.WriteString(", ")
		}
		b.WriteString("Dev:" + i.assets[j].DeviceAssetID + ", File:" + path.Base(i.assets[j].OriginalPath))
		b.WriteString("Size:" + ui.FormatBytes(i.assets[j].ExifInfo.FileSizeInByte))
		if j == i.keep {
			b.WriteString(" ✅")
		}
	}
	return b.String()
}

func (i duplicateItem) FilterValue() string {
	return i.key.Name + i.key.Date.Format("2006.01.02 15:04:05 Z07:00")
}

type itemDelegate struct {
	list.DefaultDelegate
	lastSelected  int
	itemStyle     lipgloss.Style
	selectedStyle lipgloss.Style
}

// type msgSelected struct {
// 	idx  int
// 	when time.Time
// }

// func (d itemDelegate) Height() int  { return d.DefaultDelegate.Height() }
// func (d itemDelegate) Spacing() int { return d.DefaultDelegate.Spacing() }
func (d itemDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	ix := m.Index()

	switch msg := msg.(type) { //nolint:gocritic
	// case msgSelected:
	// 	if ix == msg.idx && time.Since(msg.when) > 500*time.Millisecond {
	// 	}

	case tea.KeyMsg:
		switch msg.String() {
		case "left":
			item, ok := m.SelectedItem().(duplicateItem)
			if ok {
				if item.keep > 0 {
					item.keep--
				} else {
					item.keep = len(item.assets) - 1
				}
				m.SetItem(ix, item)
			}
		case "right":
			ix := m.Index()
			item, ok := m.SelectedItem().(duplicateItem)
			if ok {
				if item.keep < len(item.assets)-1 {
					item.keep++
				} else {
					item.keep = 0
				}
				m.SetItem(ix, item)
			}
		}
	}
	cmds := []tea.Cmd{} // cmds := []tea.Cmd{d.detectSelectionChange(ix)}
	if d.DefaultDelegate.UpdateFunc != nil {
		cmds = append(cmds, d.DefaultDelegate.UpdateFunc(msg, m))
	}
	return tea.Batch(cmds...)
}

// func (d itemDelegate) detectSelectionChange(newIdx int) tea.Cmd {
// 	if d.lastSelected != newIdx {
// 		d.lastSelected = newIdx
// 		return sendMsg(msgSelected{
// 			idx:  newIdx,
// 			when: time.Now(),
// 		})
// 	}
// 	return nil
// }

// func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
// 	i, ok := listItem.(duplicateItem)
// 	if !ok {
// 		return
// 	}

// 	str := fmt.Sprintf("%d. %s", index+1, i)

// 	fn := d.itemStyle.Render
// 	if index == m.Index() {
// 		fn = func(s ...string) string {
// 			return d.selectedStyle.Render(s)
// 		}
// 	}

// 	fmt.Fprintf(w, fn(str))
// }

// loadTable with the result of the scan
func (m DuplicateModel) loadTable(keys []duplicateKey) (list.Model, tea.Cmd) { //nolint:unparam
	items := make([]list.Item, len(keys))
	for i, k := range keys {
		l := m.app.assetsByBaseAndDate[k]
		sort.Slice(l, func(i, j int) bool { return l[i].ExifInfo.FileSizeInByte < l[j].ExifInfo.FileSizeInByte })
		items[i] = duplicateItem{
			key:    k,
			assets: l,
			keep:   len(l) - 1,
		}
	}
	m.list.SetItems(items)
	m.list.SetDelegate(itemDelegate{
		DefaultDelegate: list.NewDefaultDelegate(),
		lastSelected:    -1,
	})
	return m.list, nil
}

// func (m DuplicateModel) selectKey(k duplicateKey) (list.Model, tea.Cmd) { //nolint:unparam
// 	m.selectedKey = k
// 	l := m.app.assetsByBaseAndDate[k]
// 	sort.Slice(l, func(i, j int) bool { return l[i].ExifInfo.FileSizeInByte < l[j].ExifInfo.FileSizeInByte })
// 	items := make([]list.Item, len(l))
// 	for i := range l {
// 		item := duplicateItem{
// 			a:      l[i],
// 			keepMe: i == len(l)-1,
// 		}
// 		items[i] = item
// 	}

// 	m.sideList = list.New(items, list.NewDefaultDelegate(), 50, m.height-6)
// 	return m.sideList, nil
// }

func (m DuplicateModel) adjustListTitle(done bool) list.Model {
	if !done {
		m.list.Title = fmt.Sprintf("Receiving asstets (%d%%), %d duplicates", m.receivedPct, m.receivedDup)
	} else {
		m.list.StopSpinner()
		m.list.Title = "List of duplicates"
	}
	return m.list
}

func sendMsg[T any](m T) tea.Cmd {
	return func() tea.Msg {
		return m
	}
}
