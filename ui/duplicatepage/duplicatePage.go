package duplicatepage

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/simulot/immich-go/immich"
	"github.com/simulot/immich-go/ui/duplicatepage/duplicateitem"
	"github.com/simulot/immich-go/ui/duplicatepage/duplicatelist"
)

type currentView int

const (
	groupView currentView = iota
	itemView
)

func NewDuplicatePage(immich immich.ImmichInterface, banner string) Model {
	m := Model{
		immich:      immich,
		banner:      banner,
		currentView: groupView,

		groupList: duplicatelist.NewListModel([]list.Item{}, 50, 50),
	}

	return m
}

/*
Handle the list of duplicated assets found in Immich
based on lipgloss List
*/

type Model struct {
	immich      immich.ImmichInterface // Immich client
	banner      string
	currentView currentView

	completed     bool
	groupList     duplicatelist.Model
	groupModel    duplicateitem.Model
	width, height int

	Err error
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var (
		cmd  tea.Cmd
		cmds []tea.Cmd
	)

	// keep the underlying list updated
	switch m.currentView {
	case groupView:
		m.groupList, cmd = m.groupList.Update(msg)
	case itemView:
		m.groupModel, cmd = m.groupModel.Update(msg)
	}

	if cmd != nil {
		cmds = append(cmds, cmd)
	}

	switch msg := msg.(type) { //nolint:gocritic
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.groupList.Resize(m.width, m.height)
	case []list.Item:
		m.completed = true
	case duplicateitem.EditGroup:
		m.groupModel = duplicateitem.NewGroupModel(m.immich, msg.Index, msg.Group, m.width, m.height)
		m.currentView = itemView
	case duplicateitem.BackFromGroup:
		m.currentView = groupView
	case tea.QuitMsg:

	}
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	switch m.currentView {
	default:
		return m.groupList.View()
	case itemView:
		return m.groupModel.View()
	}
}

// func (m Model) populateList(items duplicateitem.DuplicateListLoaded) Model {
// 	m.items = &items
// 	m.list = duplicatelist.NewListModel(m.items, m.width, m.height)
// 	return m
// }
