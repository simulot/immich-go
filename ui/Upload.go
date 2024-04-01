package ui

import (
	"fmt"
	"slices"
	"strings"
	"sync"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/log"
)

// AddEntryFn
type AddEntryFn func(file string, action UpLdAction, keyval ...string)

// Upload implements the TUI for the Upload command.
type Upload struct {
	messages          []newMessage
	receivedAssets    float64
	receivedAssetsBar progress.Model
	l                 *log.Logger
	p                 *tea.Program
	counters          *counters
}

// Check if Upload implements tea.Model
var _ tea.Model = (*Upload)(nil)

// Create an upload TUI
func NewUpload(log *log.Logger, opts ...tea.ProgramOption) *Upload {
	u := Upload{
		l:                 log,
		counters:          newCounters(),
		receivedAssetsBar: progress.New(progress.WithWidth(40)),
	}
	p := tea.NewProgram(&u, opts...)
	u.p = p
	return &u
}

// Run starts the even loop for the page
func (u *Upload) Run() error {
	_, err := u.p.Run()
	return err
}

// Quit terminates the event loop for the update page
func (u *Upload) Quit() {
	u.p.Send(tea.Quit)
}

// Send the message to the page program
func (u *Upload) Send(msg tea.Msg) {
	u.p.Send(msg)
}

// SetReceived set percentage of received assets
type updReceiveAsset float64

func (u *Upload) ReceiveAsset(percent float64) {
	u.p.Send(updReceiveAsset(percent))
}

// AddEntry add an event and dispatch the update
type updJnlMsg any

func (u *Upload) AddEntry(file string, action UpLdAction, keyval ...string) {
	switch action {
	case ERROR, ServerError:
		u.Error(action, append([]string{"file", file}, keyval...))
	case DiscoveredFile:
		u.Debug(action, append([]string{"file", file}, keyval...))
	case Uploaded:
		u.Print(action, append([]string{"file", file}, keyval...))
	default:
		u.Info(action, append([]string{"file", file}, keyval...))
	}
	u.counters.Add(action)
	if u.p != nil {
		u.p.Send(updJnlMsg(nil))
	}
}

// Init implements the tea.Model
func (u *Upload) Init() tea.Cmd {
	return nil
}

// View implements the tea.Model
func (u *Upload) View() string {
	sb := strings.Builder{}
	sb.WriteRune('\n')
	for _, m := range u.messages {
		sb.WriteString(m.level.String())
		sb.WriteString(": ")
		sb.WriteString(m.message)
		sb.WriteRune('\n')
	}
	if u.receivedAssets >= 0 {
		sb.WriteString("Receiving assets from the Immich server: ")
		sb.WriteString(u.receivedAssetsBar.ViewAs(u.receivedAssets))
		sb.WriteRune('\n')
	}
	sb.WriteString(u.counters.View())
	return sb.String()
}

// Update  implements the tea.Model
func (u *Upload) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case newMessage:
		u.messages = append(u.messages, msg)
		if len(u.messages) > 10 {
			u.messages = slices.Delete(u.messages, 0, 1)
		}
	case updReceiveAsset:
		u.receivedAssets = float64(msg)
		return u, nil
	case tea.QuitMsg:
		return u, tea.Sequence(tea.Quit)
	case tea.KeyMsg:
		if msg.String() == "q" {
			return u, tea.Quit
		}
	}
	return u, nil
}

// Implements some Log functions to display errors and log everything
func (u *Upload) Debug(msg interface{}, keyvals ...interface{}) {
	u.l.Debug(msg, keyvals...)
}
func (u *Upload) Debugf(format string, args ...interface{}) {
	u.l.Debug(format, args...)
}

func (u *Upload) Error(msg interface{}, keyvals ...interface{}) {
	u.l.Error(msg, keyvals...)
}
func (u *Upload) Errorf(format string, args ...interface{}) {
	u.l.Error(format, args...)
}

func (u *Upload) Info(msg interface{}, keyvals ...interface{}) {
	u.l.Info(msg, keyvals...)
}
func (u *Upload) Infof(format string, args ...interface{}) {
	u.l.Info(format, args...)
}

func (u *Upload) Print(msg interface{}, keyvals ...interface{}) {
	u.l.Print(msg, keyvals...)
}
func (u *Upload) Printf(format string, args ...interface{}) {
	u.l.Print(format, args...)
}

type newMessage struct {
	level   log.Level
	message string
}

// UpLdAction describe all possible event during the upload command
type UpLdAction string

const (
	DiscoveredFile     UpLdAction = "File"
	ScannedImage       UpLdAction = "Scanned image"
	ScannedVideo       UpLdAction = "Scanned video"
	Discarded          UpLdAction = "Discarded"
	Uploaded           UpLdAction = "Uploaded"
	Upgraded           UpLdAction = "Server's asset upgraded"
	ERROR              UpLdAction = "Error"
	LocalDuplicate     UpLdAction = "Local duplicate"
	ServerDuplicate    UpLdAction = "Server has photo"
	Stacked            UpLdAction = "Stacked"
	ServerBetter       UpLdAction = "Server's asset is better"
	Album              UpLdAction = "Added to an album"
	LivePhoto          UpLdAction = "Live photo"
	FailedVideo        UpLdAction = "Failed video"
	Unsupported        UpLdAction = "File type not supported"
	Metadata           UpLdAction = "Metadata files"
	AssociatedMetadata UpLdAction = "Associated with metadata"
	INFO               UpLdAction = "Info"
	NotSelected        UpLdAction = "Not selected because of options"
	ServerError        UpLdAction = "Server error"
)

// counters counts the events occurred  during the upload command
type counters struct {
	l        sync.RWMutex
	counters map[UpLdAction]int
}

func newCounters() *counters {
	return &counters{
		counters: map[UpLdAction]int{},
	}
}

func (c *counters) Add(action UpLdAction) {
	c.l.Lock()
	c.counters[action] = c.counters[action] + 1
	c.l.Unlock()
}

func (c *counters) View() string {
	c.l.RLock()
	defer c.l.RUnlock()

	sb := strings.Builder{}
	checkFiles := c.counters[ScannedImage] + c.counters[ScannedVideo] + c.counters[Metadata] + c.counters[Unsupported] + c.counters[FailedVideo] + c.counters[Discarded]
	handledFiles := c.counters[NotSelected] + c.counters[LocalDuplicate] + c.counters[ServerDuplicate] + c.counters[ServerBetter] + c.counters[Uploaded] + c.counters[Upgraded] + c.counters[ServerError]

	sb.WriteString("Scan of the sources:")
	sb.WriteString(fmt.Sprintf("%6d files in the input", c.counters[DiscoveredFile]))
	sb.WriteString("--------------------------------------------------------")
	sb.WriteString(fmt.Sprintf("%6d photos", c.counters[ScannedImage]))
	sb.WriteString(fmt.Sprintf("%6d videos", c.counters[ScannedVideo]))
	sb.WriteString(fmt.Sprintf("%6d metadata files", c.counters[Metadata]))
	sb.WriteString(fmt.Sprintf("%6d files with metadata", c.counters[AssociatedMetadata]))
	sb.WriteString(fmt.Sprintf("%6d discarded files", c.counters[Discarded]))
	sb.WriteString(fmt.Sprintf("%6d files having a type not supported", c.counters[Unsupported]))
	sb.WriteString(fmt.Sprintf("%6d discarded files because in folder failed videos", c.counters[FailedVideo]))

	sb.WriteString(fmt.Sprintf("%6d input total (difference %d)", checkFiles, c.counters[DiscoveredFile]-checkFiles))
	sb.WriteString("--------------------------------------------------------")

	sb.WriteString(fmt.Sprintf("%6d uploaded files on the server", c.counters[Uploaded]))
	sb.WriteString(fmt.Sprintf("%6d upgraded files on the server", c.counters[Upgraded]))
	sb.WriteString(fmt.Sprintf("%6d files already on the server", c.counters[ServerDuplicate]))
	sb.WriteString(fmt.Sprintf("%6d discarded files because of options", c.counters[NotSelected]))
	sb.WriteString(fmt.Sprintf("%6d discarded files because duplicated in the input", c.counters[LocalDuplicate]))
	sb.WriteString(fmt.Sprintf("%6d discarded files because server has a better image", c.counters[ServerBetter]))
	sb.WriteString(fmt.Sprintf("%6d errors when uploading", c.counters[ServerError]))

	sb.WriteString(fmt.Sprintf("%6d handled total (difference %d)", handledFiles, c.counters[ScannedImage]+c.counters[ScannedVideo]-handledFiles))
	return sb.String()
}
