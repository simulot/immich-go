package upload

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
	"github.com/simulot/immich-go/logger"
)

type (
	msgQuit         struct{ error }
	msgReceiveAsset float64
	UploadModel     struct {
		// sub models
		messages       []logger.MsgLog
		countersMdl    UploadCountersModel
		spinnerReceive spinner.Model
		spinnerBrowser spinner.Model

		//
		counters            *logger.Counters[logger.UpLdAction]
		receivedAssetPct    float64
		spinnerBrowserLabel string
		app                 *UpCmd
		err                 error
		width, height       int
	}
)

var _ tea.Model = (*UploadModel)(nil)

func NewUploadModel(app *UpCmd, c *logger.Counters[logger.UpLdAction]) UploadModel {
	return UploadModel{
		counters:       c,
		countersMdl:    NewUploadCountersModel(c),
		spinnerReceive: spinner.New(spinner.WithSpinner(spinner.Points)),
		spinnerBrowser: spinner.New(spinner.WithSpinner(spinner.Points)),
		app:            app,
	}
}

func (m UploadModel) Init() tea.Cmd {
	return tea.Batch(cmdTick(), m.spinnerBrowser.Tick, m.spinnerReceive.Tick)
}

func (m UploadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.err = errors.New("interrupted by the user")
			return m, tea.Quit
		}
	case msgQuit:
		m.err = msg.error
		return m, tea.Quit
	case logger.MsgLog:
		m.messages = append(m.messages, msg)
		if len(m.messages) > m.height {
			m.messages = slices.Delete(m.messages, 0, 1)
		}
	case msgReceiveAsset:
		m.receivedAssetPct = float64(msg)
		return m, nil
	case msgTick:
		return m, cmdTick()
	case msgReceivingAssetDone:
		m.receivedAssetPct = 2.0
		return m, nil
	case logger.MsgStageSpinner:
		m.spinnerBrowserLabel = msg.Label
		return m, m.spinnerBrowser.Tick
	case spinner.TickMsg:
		var cmds []tea.Cmd
		var cmd tea.Cmd
		m.spinnerBrowser, cmd = m.spinnerBrowser.Update(msg)
		cmds = append(cmds, cmd)
		m.spinnerReceive, cmd = m.spinnerReceive.Update(msg)
		cmds = append(cmds, cmd)
		return m, tea.Batch(cmds...)
	}
	return m, nil
}

func (m UploadModel) View() string {
	b := strings.Builder{}
	row := 0
	if m.height > 26 {
		b.WriteString(m.app.SharedFlags.Banner)
		row += 5
	}
	if m.counters != nil {
		b.WriteString(m.countersMdl.View())
		b.WriteRune('\n')
		row += 19
	}
	if m.receivedAssetPct > 0 && m.receivedAssetPct < 2.0 {
		b.WriteString(m.spinnerReceive.View())
		b.WriteString(fmt.Sprintf(" Server's assets receiving (%d%%)", int(m.receivedAssetPct*100)))
		b.WriteRune('\n')
		row += 1
	}
	if m.spinnerBrowserLabel != "" {
		b.WriteString(m.spinnerBrowser.View())
		b.WriteString(" ")
		b.WriteString(m.spinnerBrowserLabel)
		b.WriteRune('\n')
		row += 1
	}
	if len(m.messages) > 0 {
		remains := m.height - row
		for i := max(len(m.messages)-remains, 0); i < len(m.messages); i++ {
			if m.messages[i].Lvl != log.InfoLevel {
				b.WriteString(m.messages[i].Lvl.String())
				b.WriteRune(' ')
			}
			b.WriteString(m.messages[i].Message)
			b.WriteRune('\n')
			row++
		}
	}
	return b.String()
}

// UploadCountersModel is a tea.Model for upload counters
type UploadCountersModel struct {
	counters *logger.Counters[logger.UpLdAction]
}

var _ tea.Model = (*UploadCountersModel)(nil)

func NewUploadCountersModel(counters *logger.Counters[logger.UpLdAction]) UploadCountersModel {
	return UploadCountersModel{
		counters: counters,
	}
}

func (m UploadCountersModel) View() string {
	c := m.counters.GetCounters()
	if c == nil {
		return ""
	}

	sb := strings.Builder{}
	checkFiles := c[logger.UpldScannedImage] + c[logger.UpldScannedVideo]
	handledFiles := c[logger.UpldLocalDuplicate] + c[logger.UpldServerDuplicate] + c[logger.UpldServerBetter] + c[logger.UpldUploaded] + c[logger.UpldUpgraded] + c[logger.UpldServerError] + c[logger.UpldNotSelected]

	sb.WriteString("-------------------------------------------------------------------\n")
	sb.WriteString(fmt.Sprintf("%6d discovered files in the input\n", c[logger.UpldDiscoveredFile]))
	sb.WriteString(fmt.Sprintf("%6d photos\n", c[logger.UpldScannedImage]))
	sb.WriteString(fmt.Sprintf("%6d videos\n", c[logger.UpldScannedVideo]))
	sb.WriteString(fmt.Sprintf("%6d metadata files\n", c[logger.UpldMetadata]))
	sb.WriteString(fmt.Sprintf("%6d files with metadata\n", c[logger.UpldAssociatedMetadata]))
	sb.WriteString(fmt.Sprintf("%6d discarded files\n", c[logger.UpldDiscarded]))
	sb.WriteString("\n-------------------------------------------------------------------\n")

	sb.WriteString(fmt.Sprintf("%6d asset(s) received from the server\n", c[logger.UpldReceived]))
	sb.WriteString(fmt.Sprintf("%6d not selected\n", c[logger.UpldNotSelected]))
	sb.WriteString(fmt.Sprintf("%6d uploaded files on the server\n", c[logger.UpldUploaded]))
	sb.WriteString(fmt.Sprintf("%6d upgraded files on the server\n", c[logger.UpldUpgraded]))
	sb.WriteString(fmt.Sprintf("%6d files already on the server\n", c[logger.UpldServerDuplicate]))
	sb.WriteString(fmt.Sprintf("%6d discarded files because duplicated in the input\n", c[logger.UpldLocalDuplicate]))
	sb.WriteString(fmt.Sprintf("%6d discarded files because server has a better image\n", c[logger.UpldServerBetter]))
	sb.WriteString(fmt.Sprintf("%6d errors when uploading\n", c[logger.UpldServerError]))

	sb.WriteString(fmt.Sprintf("%6d handled total (difference %d)\n", handledFiles, checkFiles-handledFiles))
	return sb.String()
}

// Init implements the tea.Model
func (m UploadCountersModel) Init() tea.Cmd {
	return nil
}

// Update  implements the tea.Model
func (m *UploadCountersModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	return m, nil
}

type msgTick time.Time

func cmdTick() tea.Cmd {
	return tea.Tick(time.Millisecond*100, func(t time.Time) tea.Msg {
		return msgTick(t)
	})
}

type msgReceivingAssetDone struct{}
