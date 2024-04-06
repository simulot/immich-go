package upload

import (
	"fmt"
	"slices"
	"strings"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/simulot/immich-go/logger"
)

const (
	padding  = 2
	maxWidth = 80
)

type uiStage int

const (
	uiInit uiStage = iota
	uiGetAssets
)

type ChangeStage uiStage
type ReceiveAssetMsg float64
type ErrorAndQuit error

// UploadModel is a tea.Model to follow the Upload task
type UploadModel struct {
	// sub models
	messages         []logger.LogMessage
	countersMdl      UploadCountersModel
	receivedAssetBar progress.Model

	//
	counters         *logger.Counters[logger.UpLdAction]
	receivedAssetPct float64
}

var _ tea.Model = (*UploadModel)(nil)

func NewUploadModel(c *logger.Counters[logger.UpLdAction]) UploadModel {
	return UploadModel{
		counters:    c,
		countersMdl: NewUploadCountersModel(c),
	}
}

func cmdChangeStage(newStage uiStage) func() tea.Msg {
	return func() tea.Msg {
		return newStage
	}
}

func (m UploadModel) Init() tea.Cmd {
	return cmdChangeStage(uiInit)
}

func (m UploadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.receivedAssetBar.Width = msg.Width - padding*2 - 4
		if m.receivedAssetBar.Width > maxWidth {
			m.receivedAssetBar.Width = maxWidth
		}
		return m, nil

	case logger.LogMessage:
		m.messages = append(m.messages, msg)
		if len(m.messages) > 10 {
			m.messages = slices.Delete(m.messages, 0, 1)
		}
	case progress.FrameMsg:
		progressModel, cmd := m.receivedAssetBar.Update(msg)
		m.receivedAssetBar = progressModel.(progress.Model)
		return m, cmd
	case ReceiveAssetMsg:
		m.receivedAssetPct = float64(msg)
		return m, m.receivedAssetBar.SetPercent(float64(msg))
	case logger.RefreshCounters:
		return m.countersMdl.Update(msg)
	}
	return m, nil
}

func (m UploadModel) View() string {
	b := strings.Builder{}
	for i := range m.messages {
		b.WriteString(m.messages[i].Lvl.String())
		b.WriteRune(' ')
		b.WriteString(m.messages[i].Message)
		b.WriteRune('\n')
	}
	b.WriteString(m.receivedAssetBar.View())
	b.WriteRune('\n')
	if m.counters != nil {
		b.WriteString(m.countersMdl.View())
		b.WriteRune('\n')
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
	checkFiles := c[logger.UpldScannedImage] + c[logger.UpldScannedVideo] + c[logger.UpldMetadata] + c[logger.UpldUnsupported] + c[logger.UpldFailedVideo] + c[logger.UpldDiscarded]
	handledFiles := c[logger.UpldNotSelected] + c[logger.UpldLocalDuplicate] + c[logger.UpldServerDuplicate] + c[logger.UpldServerBetter] + c[logger.UpldUploaded] + c[logger.UpldUpgraded] + c[logger.UpldServerError]

	sb.WriteString("Scan of the sources:\n")
	sb.WriteString(fmt.Sprintf("%6d files in the input\n", c[logger.UpldDiscoveredFile]))
	sb.WriteString("--------------------------------------------------------\n")
	sb.WriteString(fmt.Sprintf("%6d photos\n", c[logger.UpldScannedImage]))
	sb.WriteString(fmt.Sprintf("%6d videos\n", c[logger.UpldScannedVideo]))
	sb.WriteString(fmt.Sprintf("%6d metadata files\n", c[logger.UpldMetadata]))
	sb.WriteString(fmt.Sprintf("%6d files with metadata\n", c[logger.UpldAssociatedMetadata]))
	sb.WriteString(fmt.Sprintf("%6d discarded files\n", c[logger.UpldDiscarded]))
	sb.WriteString(fmt.Sprintf("%6d files having a type not supported\n", c[logger.UpldUnsupported]))
	sb.WriteString(fmt.Sprintf("%6d discarded files because in folder failed videos\n", c[logger.UpldFailedVideo]))

	sb.WriteString(fmt.Sprintf("%6d input total (difference %d)\n", checkFiles, c[logger.UpldDiscoveredFile]-checkFiles))
	sb.WriteString("--------------------------------------------------------\n")

	sb.WriteString(fmt.Sprintf("%6d uploaded files on the server\n", c[logger.UpldUploaded]))
	sb.WriteString(fmt.Sprintf("%6d upgraded files on the server\n", c[logger.UpldUpgraded]))
	sb.WriteString(fmt.Sprintf("%6d files already on the server\n", c[logger.UpldServerDuplicate]))
	sb.WriteString(fmt.Sprintf("%6d discarded files because of options\n", c[logger.UpldNotSelected]))
	sb.WriteString(fmt.Sprintf("%6d discarded files because duplicated in the input\n", c[logger.UpldLocalDuplicate]))
	sb.WriteString(fmt.Sprintf("%6d discarded files because server has a better image\n", c[logger.UpldServerBetter]))
	sb.WriteString(fmt.Sprintf("%6d errors when uploading\n", c[logger.UpldServerError]))

	sb.WriteString(fmt.Sprintf("%6d handled total (difference %d)\n", handledFiles, c[logger.UpldScannedImage]+c[logger.UpldScannedVideo]-handledFiles))
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
