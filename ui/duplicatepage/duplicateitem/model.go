package duplicateitem

import (
	"context"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/simulot/immich-go/helpers/asciimage"
	"github.com/simulot/immich-go/immich"
)

type currentFocus int

const (
	focusOnLeft currentFocus = iota
	focusOnRight
)

type Model struct {
	left          list.Model
	right         *huh.Form
	Index         int // Group index in the main list
	Group             // Group being edited
	selected      int // Selected asset in the group
	focus         currentFocus
	height, width int
	fields        *fields
	immich        immich.ImmichInterface
	image         string
}

type fields struct {
	name string
	date string
}

func NewGroupModel(immich immich.ImmichInterface, index int, g Group, width, height int) Model {
	m := Model{
		Index:    index,
		Group:    g,
		selected: -1,
		height:   30,
		width:    80,
		immich:   immich,
	}

	l := []list.Item{}
	for _, a := range g.Assets {
		l = append(l, Item{asset: a})
	}
	m.left = list.New(l, list.NewDefaultDelegate(), width, height)
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
	m.left, cmd = m.left.Update(msg)
	cmds = append(cmds, cmd)

	if newSelected := m.left.Index(); newSelected != m.selected {
		cmd = m.selectItem(newSelected)
		cmds = append(cmds, cmd)

	}

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "shift+tab":
			if m.focus == focusOnLeft {
				cmd = sendMsg(BackFromGroup{})
				cmds = append(cmds, cmd)
			} else {
				m.focus = focusOnLeft
			}
		case "tab":
			if m.focus == focusOnLeft {
				m.focus = focusOnRight
			}
		}
	case DisplayThumbnail:
		m.image = string(msg)

	}
	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	l := m.left.View()
	r := ""
	if m.right != nil {
		r = m.right.View() + "\n"
	}
	r += m.image

	return lipgloss.JoinHorizontal(lipgloss.Top, l, r)
}

func (m *Model) selectItem(selected int) tea.Cmd {
	m.selected = selected
	m.image = ""
	a := m.Group.Assets[m.selected]
	m.fields = &fields{
		name: a.OriginalFileName,
		date: a.ExifInfo.DateTimeOriginal.Format(TimeFormat),
	}
	m.right = huh.NewForm(huh.NewGroup(
		huh.NewInput().Key("name").Title("File name").Placeholder("change the name").Value(&m.fields.name),
		huh.NewInput().Key("date").Title("Date of capture").Placeholder("YYYY/MM/DD HH:MM:SS +00:00").Value(&m.fields.date),
		huh.NewConfirm().Key("done").Title("Save").Affirmative("save").Negative("cancel"),
	))
	cmds := []tea.Cmd{m.right.Init(), m.getThumbnail(selected)}
	return tea.Batch(cmds...)
}

func (m Model) getThumbnail(selected int) tea.Cmd {
	img, err := m.immich.GetAssetThumbnail(context.Background(), m.Group.Assets[selected].ID)
	if err == nil {
		s, _ := asciimage.Utf8Renderer(img, 80, 50)
		return sendMsg(DisplayThumbnail(s))
	}
	return nil
}

// func (m Model) getThumbnail()
func sendMsg[T any](m T) tea.Cmd {
	return func() tea.Msg {
		return m
	}
}
