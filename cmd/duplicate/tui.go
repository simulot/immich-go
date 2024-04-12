package duplicate

import (
	"errors"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type DuplicateModel struct {
	spinnerReceive        spinner.Model
	spinnerReceiveLabel   string
	spinnerDuplicate      spinner.Model
	spinnerDuplicateLabel string
	ready                 bool
	keyIndex              int
	unselectedStyle       lipgloss.Style
	selectedStyle         lipgloss.Style

	app *DuplicateCmd

	width, height int
	err           error
}

type (
	msgSpinnerReceive   string
	msgSpinnerDuplicate string
	msgError            struct {
		Err error
	}
	msgReady any
)

var _ tea.Model = (*DuplicateModel)(nil)

func NewDuplicateModel(app *DuplicateCmd) DuplicateModel {
	return DuplicateModel{
		app:              app,
		spinnerReceive:   spinner.New(spinner.WithSpinner(spinner.Points)),
		spinnerDuplicate: spinner.New(spinner.WithSpinner(spinner.Points)),
		unselectedStyle:  lipgloss.NewStyle().MarginRight(1).MarginLeft(1),
		selectedStyle:    lipgloss.NewStyle().MarginRight(1).MarginLeft(1).Foreground(lipgloss.Color("10")).Background(lipgloss.Color("7")),
	}
}

func (m DuplicateModel) Init() tea.Cmd {
	return m.spinnerReceive.Tick
}

func (m DuplicateModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.err = errors.New("interrupted by the user")
			return m, tea.Quit
		case "up":
			if m.ready && m.keyIndex > 0 {
				m.keyIndex--
			}
		case "pgup":
			if m.ready && m.keyIndex > 0 {
				m.keyIndex = max(m.keyIndex-m.app.rows, 0)
			}
		case "down":
			if m.ready && m.keyIndex < len(m.app.keys)-1 {
				m.keyIndex++
			}
		case "pgdown":
			if m.ready && m.keyIndex < len(m.app.keys)-1 {
				m.keyIndex = min(m.keyIndex+m.app.rows, len(m.app.keys)-1)
			}

		}
	case spinner.TickMsg:
		if !m.ready {
			var cmds []tea.Cmd
			var cmd tea.Cmd

			m.spinnerDuplicate, cmd = m.spinnerDuplicate.Update(msg)
			cmds = append(cmds, cmd)
			m.spinnerReceive, cmd = m.spinnerReceive.Update(msg)
			cmds = append(cmds, cmd)
			return m, tea.Batch(cmds...)
		}

	case msgError:
		m.err = msg.Err
		return m, tea.Quit

	case msgSpinnerReceive:
		m.spinnerReceiveLabel = string(msg)
	case msgSpinnerDuplicate:
		m.spinnerDuplicateLabel = string(msg)
	case msgReady:
		m.ready = true
	}
	return m, nil
}

func (m DuplicateModel) View() string {
	b := strings.Builder{}
	row := 0
	if m.height > 10 {
		b.WriteString(m.app.SharedFlags.Banner)
		row += 5
	}
	if m.spinnerReceiveLabel != "" {
		if !m.ready {
			b.WriteString(m.spinnerReceive.View())
			b.WriteString(" ")
		}
		b.WriteString(m.spinnerReceiveLabel)
		b.WriteRune('\n')
		row++
	}

	if m.spinnerDuplicateLabel != "" {
		if !m.ready {
			b.WriteString(m.spinnerDuplicate.View())
			b.WriteString(" ")
		}
		b.WriteString(m.spinnerDuplicateLabel)
		b.WriteRune('\n')
		row++
	}

	m.app.rows = min(m.height-row, len(m.app.keys))
	start := max(0, m.keyIndex-m.app.rows/2)
	end := min(len(m.app.keys), start+m.app.rows)
	for i := start; i < end; i++ {
		k := m.app.keys[i]
		if i == m.keyIndex {
			b.WriteString(m.selectedStyle.Render(k.Date.Format(time.RFC3339)))
			b.WriteString(m.selectedStyle.Copy().Width(40).Render(k.Name))
		} else {
			b.WriteString(m.unselectedStyle.Render(k.Date.Format(time.RFC3339)))
			b.WriteString(m.unselectedStyle.Copy().Width(40).Render(k.Name))
		}
		b.WriteRune('\n')
		row++
	}
	_ = row
	return b.String()
}
