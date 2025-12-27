//go:build ui_terminal

package terminal

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/simulot/immich-go/internal/ui/core/messages"
	"github.com/simulot/immich-go/internal/ui/core/services"
	"github.com/simulot/immich-go/internal/ui/core/state"
)

const (
	defaultTerminalWidth  = 100
	defaultTerminalHeight = 28
	minCardWidth          = 38
	maxRecentEntries      = 7
	maxAlertEntries       = 5
	maxLogEntries         = 1000 // Store many logs, display will be limited by screen height
	sparklinePoints       = 24
	columnGapWidth        = 4
	rowGapLines           = 1
)

type eventMsg struct {
	event messages.Event
}

type streamClosedMsg struct{}

// runUploadDashboard spins the Bubble Tea program that renders the upload dashboard layout.
func runUploadDashboard(ctx context.Context, cfg Config, stream messages.Stream) error {
	model := newUploadDashboardModel(cfg, stream)
	program := tea.NewProgram(model, tea.WithContext(ctx), tea.WithAltScreen())
	_, err := program.Run()
	return err
}

// uploadDashboardModel holds UI state for the upload dashboard.
type uploadDashboardModel struct {
	cfg     Config
	theme   services.Theme
	profile string

	stream messages.Stream

	width  int
	height int

	stats     state.RunStats
	jobs      []state.JobSummary
	inventory state.ServerInventory
	recent    []state.AssetEvent
	alerts    []state.LogEvent
	logs      []state.LogEvent
	spotlight state.AssetEvent

	serverURL string
	userEmail string

	streamClosed bool
}

func newUploadDashboardModel(cfg Config, stream messages.Stream) *uploadDashboardModel {
	theme := cfg.Theme
	if len(theme.Colors) == 0 {
		theme = services.DefaultTheme()
	}
	profile := cfg.ProfileLabel
	if profile == "" {
		profile = "default"
	}
	return &uploadDashboardModel{
		cfg:       cfg,
		theme:     theme,
		profile:   profile,
		stream:    stream,
		serverURL: cfg.ServerURL,
		userEmail: cfg.UserEmail,
		stats: state.RunStats{
			Stage: state.StagePending,
		},
	}
}

func (m *uploadDashboardModel) Init() tea.Cmd {
	return listenToStream(m.stream)
}

func (m *uploadDashboardModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		}
	case eventMsg:
		m.consumeEvent(msg.event)
		return m, listenToStream(m.stream)
	case streamClosedMsg:
		m.streamClosed = true
		m.stream = nil
	}
	return m, nil
}

func (m *uploadDashboardModel) View() string {
	width := m.layoutWidth()
	height := m.layoutHeight()

	header := renderHeader(m.serverURL, m.userEmail, m.theme)
	separator1 := renderSeparator(width, m.theme)
	statsRow := renderStatsRow(m.stats, m.inventory, m.jobs, width, m.theme)
	separator2 := renderSeparator(width, m.theme)
	footer := renderFooter(m.stats, m.streamClosed, width, m.theme)

	// Calculate remaining height for logs
	// header=1, separator1=1, stats=height of statsRow, separator2=1, footer=1
	statsHeight := lipgloss.Height(statsRow)
	logHeight := height - 4 - statsHeight
	if logHeight < 1 {
		logHeight = 1
	}
	logs := renderLogPanel(m.logs, width, logHeight, m.theme)

	sections := []string{
		header,
		separator1,
		statsRow,
		separator2,
		logs,
		footer,
	}
	content := lipgloss.JoinVertical(lipgloss.Left, sections...)
	return lipgloss.Place(width, height, lipgloss.Left, lipgloss.Top, content)
}

func (m *uploadDashboardModel) consumeEvent(evt messages.Event) {
	switch evt.Type {
	case messages.EventStatsUpdated:
		if stats, ok := evt.Payload.(state.RunStats); ok {
			m.stats = stats
		}
	case messages.EventJobsUpdated:
		if jobs, ok := evt.Payload.([]state.JobSummary); ok {
			m.jobs = jobs
		}
	case messages.EventInventoryUpdated:
		if inv, ok := evt.Payload.(state.ServerInventory); ok {
			m.inventory = inv
		}
	case messages.EventLogLine:
		if entry, ok := evt.Payload.(state.LogEvent); ok {
			m.appendLog(entry)
		}
	case messages.EventAssetQueued, messages.EventAssetUploaded, messages.EventAssetFailed:
		if assetEvt, ok := evt.Payload.(state.AssetEvent); ok {
			m.appendActivity(assetEvt)
		}
	}
}

func (m *uploadDashboardModel) appendActivity(evt state.AssetEvent) {
	m.recent = prependAssetEvent(m.recent, evt, maxRecentEntries)
	if evt.Asset.ID != "" || evt.Asset.Path != "" {
		m.spotlight = evt
	}
}

func (m *uploadDashboardModel) appendLog(entry state.LogEvent) {
	m.logs = prependLogEvent(m.logs, entry, maxLogEntries)
	if isAlertLevel(entry.Level) {
		m.alerts = prependLogEvent(m.alerts, entry, maxAlertEntries)
	}
}

func (m *uploadDashboardModel) layoutWidth() int {
	if m.width <= 0 {
		return defaultTerminalWidth
	}
	if m.width < minCardWidth*2 {
		return minCardWidth * 2
	}
	return m.width
}

func (m *uploadDashboardModel) layoutHeight() int {
	if m.height <= 0 {
		return defaultTerminalHeight
	}
	return m.height
}

func listenToStream(stream messages.Stream) tea.Cmd {
	if stream == nil {
		return nil
	}
	return func() tea.Msg {
		evt, ok := <-stream
		if !ok {
			return streamClosedMsg{}
		}
		return eventMsg{event: evt}
	}
}

func renderHeader(serverURL, userEmail string, theme services.Theme) string {
	var parts []string
	parts = append(parts, "immich-go v dev")
	parts = append(parts, "Upload")
	if serverURL != "" {
		parts = append(parts, fmt.Sprintf("Server %s", serverURL))
	}
	if userEmail != "" {
		parts = append(parts, fmt.Sprintf("User: %s", userEmail))
	}
	header := strings.Join(parts, " • ")
	return header
}

func renderSeparator(width int, theme services.Theme) string {
	line := strings.Repeat("─", width)
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#4C566A")).Render(line)
}

func renderStatsRow(stats state.RunStats, inv state.ServerInventory, jobs []state.JobSummary, width int, theme services.Theme) string {
	// Determine layout based on width
	// Breakpoints: narrow (1 per row), medium (2x2 grid), wide (4 in a row)
	const minTileWidth = 25
	const vertSepWidth = 3 // " │ "

	var tileWidth int
	var layout int // 1=vertical, 2=2x2, 3=4-in-row

	if width < minTileWidth*2+vertSepWidth {
		// Layout 1: Vertical stack (1 tile per row)
		layout = 1
		tileWidth = width
	} else if width < minTileWidth*4+vertSepWidth*3 {
		// Layout 2: 2x2 grid (2 tiles per row)
		layout = 2
		tileWidth = (width - vertSepWidth) / 2
	} else {
		// Layout 3: 4 tiles in a row
		layout = 3
		tileWidth = (width - vertSepWidth*3) / 4
	}

	// Render all cards with equal width
	discovery := renderDiscoveryCard(stats, tileWidth, theme)
	processing := renderProcessingCard(stats, tileWidth, theme)
	progress := renderProgressCard(stats, tileWidth, theme)
	serverJobs := renderServerJobsCard(jobs, tileWidth, theme)

	// Get max height for vertical separator
	maxHeight := maxInt(lipgloss.Height(discovery), lipgloss.Height(processing))
	maxHeight = maxInt(maxHeight, lipgloss.Height(progress))
	maxHeight = maxInt(maxHeight, lipgloss.Height(serverJobs))
	vertSep := renderVerticalSeparator(maxHeight, theme)

	switch layout {
	case 1:
		// Layout 1: Vertical stack (1 tile per row)
		return lipgloss.JoinVertical(lipgloss.Left,
			discovery,
			"",
			processing,
			"",
			progress,
			"",
			serverJobs,
		)
	case 2:
		// Layout 2: 2x2 grid
		row1 := lipgloss.JoinHorizontal(lipgloss.Top, discovery, vertSep, processing)
		row2 := lipgloss.JoinHorizontal(lipgloss.Top, progress, vertSep, serverJobs)
		return lipgloss.JoinVertical(lipgloss.Left, row1, "", row2)
	default:
		// Layout 3: 4 tiles in a row
		return lipgloss.JoinHorizontal(lipgloss.Top,
			discovery, vertSep,
			processing, vertSep,
			progress, vertSep,
			serverJobs,
		)
	}
}

func renderVerticalSeparator(height int, theme services.Theme) string {
	var lines []string
	for i := 0; i < height; i++ {
		lines = append(lines, " │ ")
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#4C566A")).Render(strings.Join(lines, "\n"))
}

func renderDiscoveryCard(stats state.RunStats, width int, theme services.Theme) string {
	imagesCount := 0
	imagesSize := int64(0)
	videosCount := 0
	videosSize := int64(0)

	// These would need to come from more detailed stats - for now use totals
	imagesCount = stats.TotalDiscovered
	imagesSize = stats.TotalDiscoveredBytes

	title := lipgloss.NewStyle().Bold(true).Underline(true).Render("Source discovery:")
	lines := []string{
		title,
		fmt.Sprintf("Images             %4d  %s", imagesCount, formatBytes(imagesSize)),
		fmt.Sprintf("Videos             %4d  %s", videosCount, formatBytes(videosSize)),
		"",
		fmt.Sprintf("Same in source     %4d  %s", 0, "0 B"),
		fmt.Sprintf("Same on the server %4d  %s", 0, "0 B"),
		fmt.Sprintf("Filtered           %4d  %s", stats.Discarded, formatBytes(stats.DiscardedBytes)),
		fmt.Sprintf("Banned             %4d  %s", 0, "0 B"),
	}
	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().Width(width).Render(content)
}

func renderProcessingCard(stats state.RunStats, width int, theme services.Theme) string {
	// These counters would need to be tracked separately - using placeholders
	title := lipgloss.NewStyle().Bold(true).Underline(true).Render("Source processing:")
	lines := []string{
		title,
		fmt.Sprintf("Sidecars       %4d", 0),
		fmt.Sprintf("Albums         %4d", 0),
		fmt.Sprintf("Stacked        %4d", 0),
		fmt.Sprintf("Tagged         %4d", 0),
		fmt.Sprintf("Metadata       %4d", 0),
	}
	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().Width(width).Render(content)
}

func renderProgressCard(stats state.RunStats, width int, theme services.Theme) string {
	title := lipgloss.NewStyle().Bold(true).Underline(true).Render("Upload progress:")
	lines := []string{
		title,
		fmt.Sprintf("Pending    %4d  %s", stats.Pending, formatBytes(stats.PendingBytes)),
		fmt.Sprintf("Processed  %4d  %s", stats.Processed, formatBytes(stats.ProcessedBytes)),
		fmt.Sprintf("Discarded  %4d  %s", stats.Discarded, formatBytes(stats.DiscardedBytes)),
		fmt.Sprintf("Errors     %4d  %s", stats.ErrorCount, formatBytes(stats.ErrorBytes)),
		"",
	}
	// Make total bold
	totalLine := fmt.Sprintf("Total      %4d  %s", stats.TotalDiscovered, formatBytes(stats.TotalDiscoveredBytes))
	totalStyled := lipgloss.NewStyle().Bold(true).Render(totalLine)
	lines = append(lines, totalStyled)
	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().Width(width).Render(content)
}

func renderServerJobsCard(jobs []state.JobSummary, width int, theme services.Theme) string {
	activeJobs := 0
	waitingJobs := 0
	for _, job := range jobs {
		activeJobs += job.Active
		waitingJobs += job.Waiting
	}

	title := lipgloss.NewStyle().Bold(true).Underline(true).Render("Server operations:")
	lines := []string{
		title,
		fmt.Sprintf("Active jobs    %4d", activeJobs),
		fmt.Sprintf("Waiting jobs   %4d", waitingJobs),
	}
	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().Width(width).Render(content)
}

func renderQueueCard(stats state.RunStats, width int, theme services.Theme) string {
	throughput := formatThroughput(latestThroughput(stats))
	lines := []string{
		fmt.Sprintf("Discovered: %s", formatCountBytes(stats.TotalDiscovered, stats.TotalDiscoveredBytes)),
		fmt.Sprintf("Pending:    %s", formatCountBytes(stats.Pending, stats.PendingBytes)),
		fmt.Sprintf("Uploaded:   %s   ↑ %s", formatCountBytes(stats.Uploaded, stats.BytesSent), throughput),
		fmt.Sprintf("Failed:     %d", stats.Failed),
		fmt.Sprintf("Retries:    %d", stats.Retries),
		fmt.Sprintf("Workers:    %d", stats.Workers),
		fmt.Sprintf("Sparkline:  %s", renderSparkline(stats.ThroughputSamples)),
	}
	return cardStyle(width).BorderForeground(themeColor(theme, services.ColorHighlight, "#88C0D0")).Render(strings.Join(lines, "\n"))
}

func renderServerCounters(stats state.RunStats, inv state.ServerInventory, jobs []state.JobSummary, width int, theme services.Theme) string {
	lines := []string{
		"Server Counters",
		fmt.Sprintf("Photos:    %s", formatNumber(int64(inv.Photos))),
		fmt.Sprintf("Videos:    %s", formatNumber(int64(inv.Videos))),
		fmt.Sprintf("Total:     %s", formatNumber(int64(inv.Total))),
		fmt.Sprintf("Jobs:      %d (%s)", len(jobs), jobStatusSummary(jobs)),
		fmt.Sprintf("Latency:   %s", formatLatency(inv.Latency)),
		fmt.Sprintf("Updated:   %s", formatTimestamp(inv.UpdatedAt)),
		fmt.Sprintf("Errors:    %d", stats.ErrorCount),
	}
	return cardStyle(width).BorderForeground(themeColor(theme, services.ColorMuted, "#4C566A")).Render(strings.Join(lines, "\n"))
}

func renderRecentActivity(events []state.AssetEvent, width int, theme services.Theme) string {
	if len(events) == 0 {
		return cardStyle(width).Render("Recent Activity\n(no events yet)")
	}
	lines := make([]string, 0, len(events)+1)
	lines = append(lines, "Recent Activity")
	maxLine := maxInt(1, width-12)
	for _, evt := range events {
		summary := summarizeAsset(evt)
		if len(summary) > maxLine {
			summary = truncate(summary, maxLine-3) + "..."
		}
		lines = append(lines, fmt.Sprintf("[%s] %s", stageLabel(evt.Stage), summary))
	}
	return cardStyle(width).BorderForeground(themeColor(theme, services.ColorPrimary, "#5E81AC")).Render(strings.Join(lines, "\n"))
}

func renderAlerts(alerts []state.LogEvent, width int, theme services.Theme) string {
	if len(alerts) == 0 {
		return cardStyle(width).Render("Alerts\n(no alerts)")
	}
	lines := make([]string, 0, len(alerts)+1)
	lines = append(lines, "Alerts")
	for _, alert := range alerts {
		lines = append(lines, fmt.Sprintf("%s %s", severityIcon(alert.Level), truncate(alert.Message, width-6)))
	}
	return cardStyle(width).BorderForeground(themeColor(theme, services.ColorWarning, "#EBCB8B")).Render(strings.Join(lines, "\n"))
}

func renderSpotlight(evt state.AssetEvent, width int, theme services.Theme) string {
	lines := []string{"Asset Spotlight"}
	if evt.Asset.ID == "" && evt.Asset.Path == "" {
		lines = append(lines, "No asset selected yet")
	} else {
		lines = append(lines,
			fmt.Sprintf("Path: %s", truncate(assetDisplayName(evt.Asset), width-8)),
			fmt.Sprintf("Stage: %s", stageLabel(evt.Stage)),
			fmt.Sprintf("Code: %s", evt.CodeLabel),
			fmt.Sprintf("Bytes: %s", formatBytes(evt.Bytes)),
		)
		if evt.Reason != "" {
			lines = append(lines, fmt.Sprintf("Reason: %s", evt.Reason))
		}
	}
	return cardStyle(width).BorderForeground(themeColor(theme, services.ColorHighlight, "#88C0D0")).Render(strings.Join(lines, "\n"))
}

func renderLogPanel(entries []state.LogEvent, width, availableHeight int, theme services.Theme) string {
	if len(entries) == 0 {
		return "(no logs yet)"
	}

	// Build log lines with color coding
	var lines []string
	// entries[0] is newest (prepended), but we want to display oldest to newest
	// (like traditional terminal logs scrolling up)
	// Show the most recent N entries, in chronological order (oldest first, newest last)
	maxEntries := availableHeight
	if len(entries) < maxEntries {
		maxEntries = len(entries)
	}

	// Iterate backwards through the selected entries to show oldest first
	for i := maxEntries - 1; i >= 0; i-- {
		entry := entries[i]
		timestamp := entry.Timestamp.Format("15:04:05")
		level := strings.ToUpper(entry.Level)

		// Color code the level
		var levelStyle lipgloss.Style
		switch strings.ToLower(entry.Level) {
		case "inf", "info":
			levelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#00FFFF")) // cyan
			level = "INF"
		case "wrn", "warn", "warning":
			levelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FFFF00")) // yellow
			level = "WRN"
		case "err", "error":
			levelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000")) // red
			level = "ERR"
		default:
			levelStyle = lipgloss.NewStyle()
		}

		// Format the message with highlighted keywords
		message := highlightKeywords(entry.Message, width-25) // Reserve space for timestamp and level

		line := fmt.Sprintf("%s %s %s", timestamp, levelStyle.Render(level), message)
		lines = append(lines, line)
	}

	return strings.Join(lines, "\n")
}

func highlightKeywords(message string, maxWidth int) string {
	// Truncate if needed
	if len(message) > maxWidth {
		message = message[:maxWidth-3] + "..."
	}

	// Color code keywords like file=, json=, matcher=, album=, tag=
	keywordStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#A3BE8C")) // green
	valueStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#EBCB8B"))   // yellow/gold

	// Simple keyword highlighting
	for _, keyword := range []string{"file=", "json=", "matcher=", "album=", "tag="} {
		if idx := strings.Index(message, keyword); idx >= 0 {
			// Find the end of the value (space or end of string)
			valueStart := idx + len(keyword)
			valueEnd := strings.IndexAny(message[valueStart:], " \t\n")
			if valueEnd == -1 {
				valueEnd = len(message) - valueStart
			}
			valueEnd += valueStart

			// Reconstruct with colored keyword and value
			before := message[:idx]
			keywordPart := keywordStyle.Render(keyword)
			valuePart := valueStyle.Render(message[valueStart:valueEnd])
			after := message[valueEnd:]
			message = before + keywordPart + valuePart + after
			break // Only highlight first occurrence to avoid complexity
		}
	}

	return message
}

func renderFooter(stats state.RunStats, streamClosed bool, width int, theme services.Theme) string {
	status := statusFromStats(stats)
	if streamClosed {
		status += " • stream closed"
	}
	footer := fmt.Sprintf("Q quit  •  Status: %s", status)
	return lipgloss.NewStyle().Foreground(themeColor(theme, services.ColorMuted, "#4C566A")).Width(width).Render(footer)
}

func prependAssetEvent(list []state.AssetEvent, evt state.AssetEvent, limit int) []state.AssetEvent {
	list = append([]state.AssetEvent{evt}, list...)
	if len(list) > limit {
		return list[:limit]
	}
	return list
}

func prependLogEvent(list []state.LogEvent, entry state.LogEvent, limit int) []state.LogEvent {
	list = append([]state.LogEvent{entry}, list...)
	if len(list) > limit {
		return list[:limit]
	}
	return list
}

func isAlertLevel(level string) bool {
	switch strings.ToLower(level) {
	case "warn", "warning", "error", "fatal":
		return true
	default:
		return false
	}
}

func statusFromStats(stats state.RunStats) string {
	switch {
	case stats.Stage == state.StageCompleted:
		return "completed"
	case stats.UploadPaused, stats.Stage == state.StagePaused:
		return "paused"
	case stats.Stage == state.StageFailed:
		return "failed"
	case stats.Stage == state.StagePending:
		return "starting"
	default:
		return "running"
	}
}

func latestThroughput(stats state.RunStats) float64 {
	if n := len(stats.ThroughputSamples); n > 0 {
		return stats.ThroughputSamples[n-1].BytesPerSecond
	}
	return 0
}

func renderSparkline(samples []state.ThroughputSample) string {
	if len(samples) == 0 {
		return "(no data)"
	}
	if len(samples) > sparklinePoints {
		samples = samples[len(samples)-sparklinePoints:]
	}
	maxVal := 0.0
	for _, sample := range samples {
		if sample.BytesPerSecond > maxVal {
			maxVal = sample.BytesPerSecond
		}
	}
	if maxVal == 0 {
		return strings.Repeat("▁", len(samples))
	}
	blocks := []rune{'▁', '▂', '▃', '▄', '▅', '▆', '▇', '█'}
	var b strings.Builder
	for _, sample := range samples {
		ratio := sample.BytesPerSecond / maxVal
		idx := int(math.Round(ratio * float64(len(blocks)-1)))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(blocks) {
			idx = len(blocks) - 1
		}
		b.WriteRune(blocks[idx])
	}
	return b.String()
}

func summarizeAsset(evt state.AssetEvent) string {
	name := assetDisplayName(evt.Asset)
	if name == "" {
		name = evt.Asset.ID
	}
	extra := evt.CodeLabel
	if extra == "" {
		extra = string(evt.Stage)
	}
	return fmt.Sprintf("%s (%s)", name, extra)
}

func assetDisplayName(ref state.AssetRef) string {
	if ref.Path != "" {
		return ref.Path
	}
	return ref.ID
}

func stageLabel(stage state.AssetStage) string {
	if stage == "" {
		return "unknown"
	}
	return string(stage)
}

func severityIcon(level string) string {
	switch strings.ToLower(level) {
	case "error", "fatal":
		return "[x]"
	case "warn", "warning":
		return "[!]"
	default:
		return "[i]"
	}
}

func formatCountBytes(count int, bytes int64) string {
	return fmt.Sprintf("%s (%s)", formatNumber(int64(count)), formatBytes(bytes))
}

func formatBytes(value int64) string {
	if value <= 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	v := float64(value)
	idx := 0
	for v >= 1024 && idx < len(units)-1 {
		v /= 1024
		idx++
	}
	if idx == 0 {
		return fmt.Sprintf("%d %s", value, units[idx])
	}
	return fmt.Sprintf("%.1f %s", v, units[idx])
}

func formatNumber(value int64) string {
	negative := value < 0
	if negative {
		value = -value
	}
	s := fmt.Sprintf("%d", value)
	if len(s) <= 3 {
		if negative {
			return "-" + s
		}
		return s
	}
	var b strings.Builder
	if negative {
		b.WriteByte('-')
	}
	rem := len(s) % 3
	if rem == 0 {
		rem = 3
	}
	b.WriteString(s[:rem])
	for i := rem; i < len(s); i += 3 {
		b.WriteByte(',')
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func formatThroughput(bps float64) string {
	if bps <= 0 {
		return "0 B/s"
	}
	units := []string{"B/s", "KB/s", "MB/s", "GB/s"}
	idx := 0
	for bps >= 1024 && idx < len(units)-1 {
		bps /= 1024
		idx++
	}
	if idx == 0 {
		return fmt.Sprintf("%.0f %s", bps, units[idx])
	}
	return fmt.Sprintf("%.1f %s", bps, units[idx])
}

func formatETA(d time.Duration) string {
	if d <= 0 {
		return "--"
	}
	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60
	return fmt.Sprintf("%02dh%02dm", hours, minutes)
}

func formatLatency(d time.Duration) string {
	if d <= 0 {
		return "--"
	}
	if d < time.Second {
		return fmt.Sprintf("%d ms", d.Milliseconds())
	}
	return d.Truncate(10 * time.Millisecond).String()
}

func formatTimestamp(ts time.Time) string {
	if ts.IsZero() {
		return "--"
	}
	return ts.Format("15:04:05")
}

func jobStatusSummary(jobs []state.JobSummary) string {
	status := "idle"
	for _, job := range jobs {
		if job.Active > 0 {
			return "running"
		}
		if job.Waiting > 0 {
			status = "waiting"
		}
	}
	return status
}

func truncate(s string, width int) string {
	if width <= 0 || len(s) <= width {
		return s
	}
	if width <= 3 {
		return s[:width]
	}
	return s[:width-3] + "..."
}

func cardStyle(width int) lipgloss.Style {
	frame := 4 // 2 for borders, 2 for padding
	inner := width - frame
	if inner < 0 {
		inner = 0
	}
	return lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1).Width(inner).MaxWidth(inner)
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func themeColor(theme services.Theme, token services.ColorToken, fallback string) lipgloss.Color {
	if theme.Colors != nil {
		if hex, ok := theme.Colors[token]; ok && hex != "" {
			return lipgloss.Color(hex)
		}
	}
	return lipgloss.Color(fallback)
}
