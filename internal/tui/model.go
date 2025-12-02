package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cj3636/gdiff/internal/config"
	"github.com/cj3636/gdiff/internal/diff"
)

// Model represents the application state
type Model struct {
	diffs            []*diff.DiffResult
	activeDiff       int
	config           *config.Config
	styles           *Styles
	viewport         Viewport
	width            int
	height           int
	showHelp         bool
	showStats        bool
	sideBySideMode   bool
	syntaxHighlight  bool
	quickSwitchOpen  bool
	quickSwitchIndex int
	filePicker       FilePicker
	workingDir       string
	err              error
	helpPanelHeight  int
	statsPanelHeight int
}

// Viewport controls the visible portion of the diff
type Viewport struct {
	offset int // Current scroll position
	height int // Available height for content
}

// Styles holds all the lipgloss styles
type Styles struct {
	added       lipgloss.Style
	removed     lipgloss.Style
	unchanged   lipgloss.Style
	lineNumber  lipgloss.Style
	border      lipgloss.Style
	title       lipgloss.Style
	help        lipgloss.Style
	statusBar   lipgloss.Style
	tabActive   lipgloss.Style
	tabInactive lipgloss.Style
	modal       lipgloss.Style
}

// FilePicker is a lightweight file selection helper for creating new diffs
type FilePicker struct {
	Visible         bool
	Entries         []fileEntry
	Cursor          int
	SelectingSecond bool
	FirstFile       string
	Err             error
}

type fileEntry struct {
	Name  string
	Path  string
	IsDir bool
}

// NewModel creates a new TUI model
func NewModel(diffResult *diff.DiffResult, cfg *config.Config) Model {
	styles := createStyles(cfg.Theme)
	wd, _ := os.Getwd()
	return Model{
		diffs:            []*diff.DiffResult{diffResult},
		config:           cfg,
		styles:           styles,
		viewport:         Viewport{offset: 0, height: 20},
		showHelp:         false,
		showStats:        false,
		sideBySideMode:   false,
		syntaxHighlight:  true, // Default to enabled
		quickSwitchIndex: 0,
		workingDir:       wd,
		helpPanelHeight:  12,
		statsPanelHeight: 17,
	}
}

// createStyles initializes all lipgloss styles based on theme
func createStyles(theme config.Theme) *Styles {
	return &Styles{
		added: lipgloss.NewStyle().
			Foreground(theme.AddedFg).
			Background(theme.AddedBg),
		removed: lipgloss.NewStyle().
			Foreground(theme.RemovedFg).
			Background(theme.RemovedBg),
		unchanged: lipgloss.NewStyle().
			Foreground(theme.UnchangedFg),
		lineNumber: lipgloss.NewStyle().
			Foreground(theme.LineNumberFg).
			Width(6).
			Align(lipgloss.Right),
		border: lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(theme.BorderFg),
		title: lipgloss.NewStyle().
			Foreground(theme.TitleFg).
			Background(theme.TitleBg).
			Bold(true).
			Padding(0, 1),
		help: lipgloss.NewStyle().
			Foreground(theme.HelpFg).
			Italic(true),
		statusBar: lipgloss.NewStyle().
			Foreground(theme.TitleFg).
			Background(theme.TitleBg).
			Padding(0, 1),
		tabActive: lipgloss.NewStyle().
			Foreground(theme.TitleFg).
			Background(theme.TitleBg).
			Padding(0, 1).
			Bold(true),
		tabInactive: lipgloss.NewStyle().
			Foreground(theme.HelpFg).
			Padding(0, 1),
		modal: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(theme.BorderFg).
			Padding(1, 2).
			Background(theme.TitleBg).
			Foreground(theme.TitleFg),
	}
}

// Init initializes the model
func (m Model) Init() tea.Cmd {
	return nil
}

// Update handles messages and updates the model
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.filePicker.Visible {
			return m.handleFilePickerKey(msg)
		}

		if m.quickSwitchOpen {
			return m.handleQuickSwitchKey(msg)
		}

		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "?", "h":
			m.showHelp = !m.showHelp
			// Close stats if opening help
			if m.showHelp {
				m.showStats = false
			}
			m.updateViewportHeight()
		case "s":
			m.showStats = !m.showStats
			// Close help if opening stats
			if m.showStats {
				m.showHelp = false
			}
			m.updateViewportHeight()
		case "v":
			m.sideBySideMode = !m.sideBySideMode
		case "c":
			m.syntaxHighlight = !m.syntaxHighlight
		case "j", "down":
			m.scrollDown()
		case "k", "up":
			m.scrollUp()
		case "d":
			m.scrollPageDown()
		case "u":
			m.scrollPageUp()
		case "g":
			m.scrollToTop()
		case "G":
			m.scrollToBottom()
		case "tab":
			m.nextTab()
		case "shift+tab":
			m.prevTab()
		case "t":
			m.toggleQuickSwitch()
		case "o":
			m.openFilePicker()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateViewportHeight()
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	if m.activeDiffResult() == nil {
		return "No diff to display\n"
	}

	var sections []string

	// Tabs and title
	sections = append(sections, m.renderTabs())
	sections = append(sections, m.renderTitle())

	// Main diff content (always shown)
	sections = append(sections, m.renderDiff())

	// Bottom panel (help or stats) - shown below main view if toggled
	if m.showHelp {
		sections = append(sections, m.renderHelpPanel())
	} else if m.showStats {
		sections = append(sections, m.renderStatsPanel())
	}

	// Status bar
	sections = append(sections, m.renderStatusBar())

	content := lipgloss.JoinVertical(lipgloss.Left, sections...)

	if m.quickSwitchOpen {
		content = lipgloss.JoinVertical(lipgloss.Left, content, m.renderQuickSwitch())
	}

	if m.filePicker.Visible {
		content = lipgloss.JoinVertical(lipgloss.Left, content, m.renderFilePicker())
	}

	return content
}

// renderTitle renders the title bar
func (m Model) renderTitle() string {
	title := fmt.Sprintf("gdiff: %s ↔ %s",
		truncate(m.activeDiffResult().File1Name, 40),
		truncate(m.activeDiffResult().File2Name, 40))
	return m.styles.title.Render(title)
}

// renderTabs renders the list of open diffs
func (m Model) renderTabs() string {
	var tabs []string
	for i, d := range m.diffs {
		label := fmt.Sprintf("%d:%s", i+1, truncate(filepath.Base(d.File1Name)+"↔"+filepath.Base(d.File2Name), 22))
		if i == m.activeDiff {
			tabs = append(tabs, m.styles.tabActive.Render(label))
		} else {
			tabs = append(tabs, m.styles.tabInactive.Render(label))
		}
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, tabs...)
}

// renderDiff renders the diff content
func (m Model) renderDiff() string {
	// Calculate visible range
	start := m.viewport.offset
	end := min(start+m.viewport.height, len(m.activeDiffResult().Lines))

	if start >= len(m.activeDiffResult().Lines) {
		start = max(0, len(m.activeDiffResult().Lines)-m.viewport.height)
		m.viewport.offset = start
		end = len(m.activeDiffResult().Lines)
	}

	if start >= end || len(m.activeDiffResult().Lines) == 0 {
		return m.styles.unchanged.Render("No differences found.")
	}

	// Render based on mode
	if m.sideBySideMode {
		return m.renderSideBySide(start, end)
	}
	return m.renderUnified(start, end)
}

// renderUnified renders the diff in unified mode (traditional view)
func (m Model) renderUnified(start, end int) string {
	var lines []string

	for i := start; i < end; i++ {
		line := m.activeDiffResult().Lines[i]
		lines = append(lines, m.renderLine(line))
	}

	return strings.Join(lines, "\n")
}

// renderSideBySide renders the diff in side-by-side mode
func (m Model) renderSideBySide(start, end int) string {
	var lines []string

	// Calculate column width (split screen in half, minus borders)
	columnWidth := (m.width - 4) / 2
	if columnWidth < 20 {
		columnWidth = 20
	}

	// Render each line with left (file1) and right (file2) columns
	for i := start; i < end; i++ {
		line := m.activeDiffResult().Lines[i]
		leftContent, rightContent := m.renderSideBySideLine(line, columnWidth)

		// Join left and right with separator
		combinedLine := leftContent + " │ " + rightContent
		lines = append(lines, combinedLine)
	}

	return strings.Join(lines, "\n")
}

// renderSideBySideLine renders a single line in side-by-side mode
func (m Model) renderSideBySideLine(line diff.DiffLine, columnWidth int) (string, string) {
	var leftParts, rightParts []string
	var leftStyle, rightStyle lipgloss.Style

	// Default styles
	leftStyle = m.styles.unchanged
	rightStyle = m.styles.unchanged

	// Apply styles based on line type and syntax highlighting setting
	if m.syntaxHighlight {
		switch line.Type {
		case diff.Removed:
			leftStyle = m.styles.removed
			rightStyle = m.styles.unchanged.Faint(true)
		case diff.Added:
			leftStyle = m.styles.unchanged.Faint(true)
			rightStyle = m.styles.added
		case diff.Equal:
			leftStyle = m.styles.unchanged
			rightStyle = m.styles.unchanged
		}
	} else {
		// No syntax highlighting - use plain style
		leftStyle = m.styles.unchanged
		rightStyle = m.styles.unchanged
	}

	// Line numbers
	if m.config.ShowLineNo {
		lineNo1 := "     "
		lineNo2 := "     "
		if line.LineNo1 > 0 {
			lineNo1 = fmt.Sprintf("%5d", line.LineNo1)
		}
		if line.LineNo2 > 0 {
			lineNo2 = fmt.Sprintf("%5d", line.LineNo2)
		}
		leftParts = append(leftParts, m.styles.lineNumber.Render(lineNo1)+" ")
		rightParts = append(rightParts, m.styles.lineNumber.Render(lineNo2)+" ")
	}

	// Content
	leftContent := ""
	rightContent := ""

	switch line.Type {
	case diff.Removed:
		leftContent = "- " + line.Content
		rightContent = ""
	case diff.Added:
		leftContent = ""
		rightContent = "+ " + line.Content
	case diff.Equal:
		leftContent = "  " + line.Content
		rightContent = "  " + line.Content
	}

	// Calculate content width based on whether line numbers are shown
	contentWidth := columnWidth
	if m.config.ShowLineNo {
		contentWidth = columnWidth - 8 // Account for line numbers (5 digits + 1 space + padding)
	}

	// Ensure minimum width
	if contentWidth < 10 {
		contentWidth = 10
	}

	// Safely truncate content to fit column width
	if len(leftContent) > contentWidth && contentWidth > 3 {
		leftContent = leftContent[:contentWidth-3] + "..."
	} else if len(leftContent) > contentWidth {
		if contentWidth > 0 {
			leftContent = leftContent[:contentWidth]
		} else {
			leftContent = ""
		}
	}

	if len(rightContent) > contentWidth && contentWidth > 3 {
		rightContent = rightContent[:contentWidth-3] + "..."
	} else if len(rightContent) > contentWidth {
		if contentWidth > 0 {
			rightContent = rightContent[:contentWidth]
		} else {
			rightContent = ""
		}
	}

	// Pad to column width
	leftContent = fmt.Sprintf("%-*s", contentWidth, leftContent)
	rightContent = fmt.Sprintf("%-*s", contentWidth, rightContent)

	leftParts = append(leftParts, leftStyle.Render(leftContent))
	rightParts = append(rightParts, rightStyle.Render(rightContent))

	return strings.Join(leftParts, ""), strings.Join(rightParts, "")
}

// renderLine renders a single diff line in unified mode
func (m Model) renderLine(line diff.DiffLine) string {
	var parts []string

	// Line numbers
	if m.config.ShowLineNo {
		lineNo1 := " "
		lineNo2 := " "
		if line.LineNo1 > 0 {
			lineNo1 = fmt.Sprintf("%5d", line.LineNo1)
		} else {
			lineNo1 = "     "
		}
		if line.LineNo2 > 0 {
			lineNo2 = fmt.Sprintf("%5d", line.LineNo2)
		} else {
			lineNo2 = "     "
		}
		parts = append(parts, m.styles.lineNumber.Render(lineNo1))
		parts = append(parts, m.styles.lineNumber.Render(lineNo2))
		parts = append(parts, " ")
	}

	// Content with appropriate styling
	var symbol string
	var style lipgloss.Style

	// Apply syntax highlighting if enabled
	if m.syntaxHighlight {
		switch line.Type {
		case diff.Added:
			symbol = "+"
			style = m.styles.added
		case diff.Removed:
			symbol = "-"
			style = m.styles.removed
		case diff.Equal:
			symbol = " "
			style = m.styles.unchanged
		default:
			symbol = " "
			style = m.styles.unchanged
		}
	} else {
		// No syntax highlighting - just show symbols
		switch line.Type {
		case diff.Added:
			symbol = "+"
		case diff.Removed:
			symbol = "-"
		case diff.Equal:
			symbol = " "
		default:
			symbol = " "
		}
		style = m.styles.unchanged
	}

	content := symbol + " " + line.Content
	parts = append(parts, style.Render(content))

	return strings.Join(parts, "")
}

// renderStatusBar renders the status bar
func (m Model) renderStatusBar() string {
	added, removed, unchanged := m.activeDiffResult().GetStats()

	// View mode indicator
	viewMode := "unified"
	if m.sideBySideMode {
		viewMode = "side-by-side"
	}

	// Syntax highlighting indicator
	syntaxMode := "on"
	if !m.syntaxHighlight {
		syntaxMode = "off"
	}

	status := fmt.Sprintf(
		"Tab %d/%d | Lines: +%d -%d =%d | Pos: %d/%d | View: %s | Color: %s | tab/shift+tab switch t quick switch o open | v:view c:color s:stats ?:help q:quit",
		m.activeDiff+1, len(m.diffs),
		added, removed, unchanged,
		m.viewport.offset+1, len(m.activeDiffResult().Lines),
		viewMode, syntaxMode,
	)

	return m.styles.statusBar.Width(m.width).Render(status)
}

func (m Model) renderQuickSwitch() string {
	var items []string
	for i, d := range m.diffs {
		label := fmt.Sprintf("%d: %s ↔ %s", i+1, filepath.Base(d.File1Name), filepath.Base(d.File2Name))
		if i == m.quickSwitchIndex {
			label = m.styles.tabActive.Render(label)
		} else {
			label = m.styles.tabInactive.Render(label)
		}
		items = append(items, label)
	}

	content := strings.Join(items, "\n")
	header := m.styles.title.Render("Quick Switch (enter to open, esc to close)")
	return m.styles.modal.Width(m.width - 4).Render(header + "\n" + content)
}

func (m Model) renderFilePicker() string {
	var lines []string
	lines = append(lines, m.styles.title.Render("Open new diff from "+m.workingDir))

	if m.filePicker.Err != nil {
		lines = append(lines, m.styles.removed.Render("Error: "+m.filePicker.Err.Error()))
	}

	hint := "Select first file"
	if m.filePicker.SelectingSecond {
		hint = fmt.Sprintf("First: %s | Select second file", filepath.Base(m.filePicker.FirstFile))
	}
	lines = append(lines, m.styles.help.Render(hint))

	for i, entry := range m.filePicker.Entries {
		prefix := "  "
		if i == m.filePicker.Cursor {
			prefix = "➜ "
		}

		name := entry.Name
		if entry.IsDir {
			name += "/"
		}

		rendered := m.styles.unchanged.Render(name)
		if entry.IsDir {
			rendered = m.styles.help.Render(name)
		}

		lines = append(lines, prefix+rendered)
	}

	footer := m.styles.help.Render("enter: select • esc: close • directories open navigation")
	lines = append(lines, footer)

	return m.styles.modal.Width(m.width - 4).Render(strings.Join(lines, "\n"))
}

func (m Model) activeDiffResult() *diff.DiffResult {
	if m.activeDiff < 0 || m.activeDiff >= len(m.diffs) {
		return nil
	}
	return m.diffs[m.activeDiff]
}

func (m *Model) resetViewport() {
	m.viewport.offset = 0
}

func (m *Model) nextTab() {
	if len(m.diffs) == 0 {
		return
	}
	m.activeDiff = (m.activeDiff + 1) % len(m.diffs)
	m.quickSwitchIndex = m.activeDiff
	m.resetViewport()
}

func (m *Model) prevTab() {
	if len(m.diffs) == 0 {
		return
	}
	m.activeDiff = (m.activeDiff - 1 + len(m.diffs)) % len(m.diffs)
	m.quickSwitchIndex = m.activeDiff
	m.resetViewport()
}

func (m *Model) toggleQuickSwitch() {
	m.quickSwitchOpen = !m.quickSwitchOpen
	m.quickSwitchIndex = m.activeDiff
}

func (m *Model) handleQuickSwitchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.quickSwitchOpen = false
	case "enter":
		m.activeDiff = m.quickSwitchIndex
		m.quickSwitchOpen = false
		m.resetViewport()
	case "up", "k":
		if m.quickSwitchIndex > 0 {
			m.quickSwitchIndex--
		}
	case "down", "j":
		if m.quickSwitchIndex < len(m.diffs)-1 {
			m.quickSwitchIndex++
		}
	}

	return m, nil
}

func (m *Model) openFilePicker() {
	m.filePicker.Visible = true
	m.filePicker.Cursor = 0
	m.filePicker.SelectingSecond = false
	m.filePicker.FirstFile = ""
	m.filePicker.Err = nil
	m.refreshFilePickerEntries()
}

func (m *Model) refreshFilePickerEntries() {
	entries, err := os.ReadDir(m.workingDir)
	if err != nil {
		m.filePicker.Err = err
		return
	}

	var items []fileEntry
	if parent := filepath.Dir(m.workingDir); parent != m.workingDir {
		items = append(items, fileEntry{Name: "..", Path: parent, IsDir: true})
	}
	for _, entry := range entries {
		info := fileEntry{
			Name:  entry.Name(),
			Path:  filepath.Join(m.workingDir, entry.Name()),
			IsDir: entry.IsDir(),
		}
		items = append(items, info)
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDir == items[j].IsDir {
			return items[i].Name < items[j].Name
		}
		return items[i].IsDir && !items[j].IsDir
	})

	m.filePicker.Entries = items
	if m.filePicker.Cursor >= len(items) {
		m.filePicker.Cursor = max(0, len(items)-1)
	}
}

func (m *Model) handleFilePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.filePicker.Visible = false
		return m, nil
	case "up", "k":
		if m.filePicker.Cursor > 0 {
			m.filePicker.Cursor--
		}
	case "down", "j":
		if m.filePicker.Cursor < len(m.filePicker.Entries)-1 {
			m.filePicker.Cursor++
		}
	case "enter":
		if len(m.filePicker.Entries) == 0 {
			return m, nil
		}

		entry := m.filePicker.Entries[m.filePicker.Cursor]
		if entry.IsDir {
			m.workingDir = entry.Path
			m.refreshFilePickerEntries()
			return m, nil
		}

		if !m.filePicker.SelectingSecond {
			m.filePicker.FirstFile = entry.Path
			m.filePicker.SelectingSecond = true
			return m, nil
		}

		engine := diff.NewEngine()
		result, err := engine.DiffFiles(m.filePicker.FirstFile, entry.Path)
		if err != nil {
			m.filePicker.Err = err
			return m, nil
		}

		m.diffs = append(m.diffs, result)
		m.activeDiff = len(m.diffs) - 1
		m.quickSwitchIndex = m.activeDiff
		m.resetViewport()
		m.filePicker.Visible = false
		m.filePicker.SelectingSecond = false
		m.filePicker.FirstFile = ""
	}

	return m, nil
}

// renderHelpPanel renders the help panel below the main view
func (m Model) renderHelpPanel() string {
	helpText := []string{
		"",
		"Keyboard Shortcuts:",
		"  j, ↓      Scroll down     │  g         Go to top        │  v    Toggle side-by-side",
		"  k, ↑      Scroll up       │  G         Go to bottom     │  c    Toggle syntax colors",
		"  d         Half page down  │  s         Toggle stats     │  o    Open new diff",
		"  u         Half page up    │  h, ?      Toggle help      │  t    Quick switch",
		"  tab       Next tab        │  shift+tab Previous tab     │  q    Quit",
		"",
	}

	// Create a bordered box for the help panel
	helpStyle := m.styles.help.Copy().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.config.Theme.BorderFg).
		Padding(0, 1).
		Width(m.width - 2)

	return helpStyle.Render(strings.Join(helpText, "\n"))
}

// renderStatsPanel renders the statistics panel below the main view
func (m Model) renderStatsPanel() string {
	added, removed, unchanged := m.activeDiffResult().GetStats()
	total := added + removed + unchanged

	addedPercent := 0.0
	removedPercent := 0.0
	unchangedPercent := 0.0
	changePercent := 0.0

	if total > 0 {
		addedPercent = float64(added) * 100.0 / float64(total)
		removedPercent = float64(removed) * 100.0 / float64(total)
		unchangedPercent = float64(unchanged) * 100.0 / float64(total)
		changePercent = (float64(added+removed) * 100.0) / float64(total)
	}

	statsText := []string{
		"",
		"Diff Statistics",
		"═══════════════",
		fmt.Sprintf("File 1: %s  │  File 2: %s",
			truncate(m.activeDiffResult().File1Name, 35),
			truncate(m.activeDiffResult().File2Name, 35)),
		"",
		fmt.Sprintf("Total: %d lines  │  Added: %d (%.1f%%)  │  Removed: %d (%.1f%%)  │  Unchanged: %d (%.1f%%)",
			total, added, addedPercent, removed, removedPercent, unchanged, unchangedPercent),
		fmt.Sprintf("Changes: %d (%.1f%% of total)", added+removed, changePercent),
		"",
	}

	// Create a bordered box for the stats panel
	statsStyle := m.styles.help.Copy().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.config.Theme.BorderFg).
		Padding(0, 1).
		Width(m.width - 2)

	return statsStyle.Render(strings.Join(statsText, "\n"))
}

// Scroll functions
func (m *Model) scrollDown() {
	maxOffset := max(0, len(m.activeDiffResult().Lines)-m.viewport.height)
	if m.viewport.offset < maxOffset {
		m.viewport.offset++
	}
}

func (m *Model) scrollUp() {
	if m.viewport.offset > 0 {
		m.viewport.offset--
	}
}

func (m *Model) scrollPageDown() {
	halfPage := m.viewport.height / 2
	if halfPage < 1 {
		halfPage = 1
	}
	m.viewport.offset += halfPage
	maxOffset := max(0, len(m.activeDiffResult().Lines)-m.viewport.height)
	if m.viewport.offset > maxOffset {
		m.viewport.offset = maxOffset
	}
}

func (m *Model) scrollPageUp() {
	halfPage := m.viewport.height / 2
	if halfPage < 1 {
		halfPage = 1
	}
	m.viewport.offset -= halfPage
	if m.viewport.offset < 0 {
		m.viewport.offset = 0
	}
}

func (m *Model) scrollToTop() {
	m.viewport.offset = 0
}

func (m *Model) scrollToBottom() {
	m.viewport.offset = max(0, len(m.activeDiffResult().Lines)-m.viewport.height)
}

// updateViewportHeight calculates and sets the viewport height based on screen size and active panels
func (m *Model) updateViewportHeight() {
	// Base height: total - title bar - status bar
	baseHeight := m.height - 2

	// Subtract panel height if help or stats is shown
	if m.showHelp {
		baseHeight -= m.helpPanelHeight
	} else if m.showStats {
		baseHeight -= m.statsPanelHeight
	}

	// Ensure minimum height
	if baseHeight < 5 {
		baseHeight = 5
	}

	m.viewport.height = baseHeight
}

// Utility functions
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
