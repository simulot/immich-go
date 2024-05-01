package duplicatelist

import (
	"fmt"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/simulot/immich-go/ui/duplicatepage/duplicateitem"
)

type Model struct {
	list list.Model
}

func NewListModel(items []list.Item, width, height int) Model {
	m := Model{}
	m.list = list.New(items, list.NewDefaultDelegate(), width, height)

	return m
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)
	m.list, cmd = m.list.Update(msg)
	if cmd != nil {
		cmds = append(cmds, cmd)
	}
	switch msg := msg.(type) {
	case []list.Item:
		m.list = list.New(msg, list.NewDefaultDelegate(), m.list.Width(), m.list.Height())
		m.list.Title = "Duplicates handling"
	case DuplicateLoadingMsg:
		m.list.Title = fmt.Sprintf("Loading assets %d(%d%%), %d duplicates detected.", msg.Checked, 100*msg.Checked/msg.Total, msg.Duplicated)

	case tea.KeyMsg:
		switch msg.String() { //nolint:gocritic
		case "enter":
			cmds = append(cmds, sendMsg(duplicateitem.EditGroup{
				Index: m.list.Index(),
				Group: m.list.Items()[m.list.Index()].(duplicateitem.Group),
			}))
		}
	}

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	return m.list.View()
}

func (m *Model) Resize(width, height int) {
	m.list.SetWidth(width)
	m.list.SetHeight(height)
}

func sendMsg[T any](m T) tea.Cmd {
	return func() tea.Msg {
		return m
	}
}
