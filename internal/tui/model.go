package tui

import (
	"fmt"
	"strings"

	"github.com/cj3636/gdiff/internal/config"
	"github.com/cj3636/gdiff/internal/diff"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model represents the application state
type Model struct {
	diffResult *diff.DiffResult
	config     *config.Config
	styles     *Styles
	viewport   Viewport
	width      int
	height     int
	showHelp   bool
	showStats  bool
	err        error
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
}

// NewModel creates a new TUI model
func NewModel(diffResult *diff.DiffResult, cfg *config.Config) Model {
	styles := createStyles(cfg.Theme)
	return Model{
		diffResult: diffResult,
		config:     cfg,
		styles:     styles,
		viewport:   Viewport{offset: 0, height: 20},
		showHelp:   false,
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
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "?", "h":
			m.showHelp = !m.showHelp
		case "s":
			m.showStats = !m.showStats
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
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.height = msg.Height - 4 // Reserve space for header and footer
	}

	return m, nil
}

// View renders the UI
func (m Model) View() string {
	if m.err != nil {
		return fmt.Sprintf("Error: %v\n", m.err)
	}

	if m.diffResult == nil {
		return "No diff to display\n"
	}

	var sections []string

	// Title
	sections = append(sections, m.renderTitle())

	// Main content
	if m.showHelp {
		sections = append(sections, m.renderHelp())
	} else if m.showStats {
		sections = append(sections, m.renderStats())
	} else {
		sections = append(sections, m.renderDiff())
	}

	// Status bar
	sections = append(sections, m.renderStatusBar())

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderTitle renders the title bar
func (m Model) renderTitle() string {
	title := fmt.Sprintf("gdiff: %s ↔ %s", 
		truncate(m.diffResult.File1Name, 40),
		truncate(m.diffResult.File2Name, 40))
	return m.styles.title.Render(title)
}

// renderDiff renders the diff content
func (m Model) renderDiff() string {
	var lines []string
	
	// Calculate visible range
	start := m.viewport.offset
	end := min(start+m.viewport.height, len(m.diffResult.Lines))
	
	if start >= len(m.diffResult.Lines) {
		start = max(0, len(m.diffResult.Lines)-m.viewport.height)
		m.viewport.offset = start
		end = len(m.diffResult.Lines)
	}

	// Render visible lines
	for i := start; i < end; i++ {
		line := m.diffResult.Lines[i]
		lines = append(lines, m.renderLine(line))
	}

	if len(lines) == 0 {
		return m.styles.unchanged.Render("No differences found.")
	}

	return strings.Join(lines, "\n")
}

// renderLine renders a single diff line
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

	content := symbol + " " + line.Content
	parts = append(parts, style.Render(content))

	return strings.Join(parts, "")
}

// renderStatusBar renders the status bar
func (m Model) renderStatusBar() string {
	added, removed, unchanged := m.diffResult.GetStats()
	
	status := fmt.Sprintf(
		"Lines: +%d -%d =%d | Scroll: %d/%d | s:stats ?:help q:quit",
		added, removed, unchanged,
		m.viewport.offset+1, len(m.diffResult.Lines),
	)
	
	return m.styles.statusBar.Width(m.width).Render(status)
}

// renderHelp renders the help screen
func (m Model) renderHelp() string {
	helpText := []string{
		"",
		"Keyboard Shortcuts:",
		"",
		"  j, ↓      Scroll down one line",
		"  k, ↑      Scroll up one line",
		"  d         Scroll down half page",
		"  u         Scroll up half page",
		"  g         Go to top",
		"  G         Go to bottom",
		"  s         Toggle statistics",
		"  h, ?      Toggle help",
		"  q, Ctrl+C Quit",
		"",
	}
	
	return m.styles.help.Render(strings.Join(helpText, "\n"))
}

// renderStats renders the statistics screen
func (m Model) renderStats() string {
	added, removed, unchanged := m.diffResult.GetStats()
	total := added + removed + unchanged
	
	addedPercent := 0.0
	removedPercent := 0.0
	unchangedPercent := 0.0
	
	if total > 0 {
		addedPercent = float64(added) * 100.0 / float64(total)
		removedPercent = float64(removed) * 100.0 / float64(total)
		unchangedPercent = float64(unchanged) * 100.0 / float64(total)
	}
	
	statsText := []string{
		"",
		"Diff Statistics",
		"═══════════════",
		"",
		fmt.Sprintf("File 1: %s", m.diffResult.File1Name),
		fmt.Sprintf("File 2: %s", m.diffResult.File2Name),
		"",
		fmt.Sprintf("Total lines:     %d", total),
		fmt.Sprintf("Added lines:     %d (%.1f%%)", added, addedPercent),
		fmt.Sprintf("Removed lines:   %d (%.1f%%)", removed, removedPercent),
		fmt.Sprintf("Unchanged lines: %d (%.1f%%)", unchanged, unchangedPercent),
		"",
		fmt.Sprintf("Changes:         %d", added+removed),
		fmt.Sprintf("Change ratio:    %.1f%%", (float64(added+removed)*100.0)/float64(total)),
		"",
		"Press 's' to return to diff view",
		"",
	}
	
	return m.styles.help.Render(strings.Join(statsText, "\n"))
}

// Scroll functions
func (m *Model) scrollDown() {
	maxOffset := max(0, len(m.diffResult.Lines)-m.viewport.height)
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
	maxOffset := max(0, len(m.diffResult.Lines)-m.viewport.height)
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
	m.viewport.offset = max(0, len(m.diffResult.Lines)-m.viewport.height)
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
