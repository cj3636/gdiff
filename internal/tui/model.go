package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cj3636/gdiff/internal/config"
	"github.com/cj3636/gdiff/internal/diff"
)

// Model represents the application state
type Model struct {
	diffResult       *diff.DiffResult
	config           *config.Config
	diffEngine       *diff.Engine
	styles           *Styles
	viewport         Viewport
	width            int
	height           int
	showHelp         bool
	showStats        bool
	sideBySideMode   bool
	syntaxHighlight  bool
	showBlame        bool
	err              error
	helpPanelHeight  int
	statsPanelHeight int
	activePanel      panelType
	gitCtx           GitContext
	branchIndex      int
}

type panelType int

const (
	noPanel panelType = iota
	helpPanel
	statsPanel
	statusPanel
	branchPanel
	historyPanel
)

// Viewport controls the visible portion of the diff
type Viewport struct {
	offset int // Current scroll position
	height int // Available height for content
}

// Styles holds all the lipgloss styles
type Styles struct {
	added      lipgloss.Style
	removed    lipgloss.Style
	unchanged  lipgloss.Style
	lineNumber lipgloss.Style
	border     lipgloss.Style
	title      lipgloss.Style
	help       lipgloss.Style
	statusBar  lipgloss.Style
	blame      lipgloss.Style
}

// NewModel creates a new TUI model
func NewModel(diffResult *diff.DiffResult, cfg *config.Config, engine *diff.Engine, gitCtx GitContext) Model {
	styles := createStyles(cfg.Theme)
	model := Model{
		diffResult:       diffResult,
		config:           cfg,
		diffEngine:       engine,
		styles:           styles,
		viewport:         Viewport{offset: 0, height: 20},
		showHelp:         false,
		showStats:        false,
		sideBySideMode:   false,
		syntaxHighlight:  true, // Default to enabled
		showBlame:        gitCtx.ShowBlame,
		helpPanelHeight:  12,
		statsPanelHeight: 17,
		gitCtx:           gitCtx,
	}

	if gitCtx.Enabled {
		for i, b := range gitCtx.Branches {
			if b == gitCtx.Ref2 {
				model.branchIndex = i
				break
			}
		}
	}
	return model
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
		blame: lipgloss.NewStyle().
			Foreground(theme.HelpFg).
			Faint(true),
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
			m.togglePanel(helpPanel)
		case "s":
			m.togglePanel(statsPanel)
		case "S":
			m.togglePanel(statusPanel)
		case "B":
			m.togglePanel(branchPanel)
		case "H":
			m.togglePanel(historyPanel)
		case "v":
			m.sideBySideMode = !m.sideBySideMode
		case "c":
			m.syntaxHighlight = !m.syntaxHighlight
		case "b":
			m.showBlame = !m.showBlame
			if m.showBlame && m.gitCtx.Enabled && m.gitCtx.Blame == nil {
				m.gitCtx.Blame, m.err = m.collectBlame()
			}
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
		case "[":
			m.selectPreviousBranch()
		case "]":
			m.selectNextBranch()
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

	if m.diffResult == nil {
		return "No diff to display\n"
	}

	var sections []string

	// Title
	sections = append(sections, m.renderTitle())

	// Main diff content (always shown)
	sections = append(sections, m.renderDiff())

	// Bottom panel (help or stats) - shown below main view if toggled
	if m.activePanel != noPanel {
		sections = append(sections, m.renderActivePanel())
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

	if m.gitCtx.Enabled {
		title = fmt.Sprintf("gdiff: %s (%s) ↔ %s (%s)",
			truncate(m.diffResult.File1Name, 25), m.gitCtx.Ref1,
			truncate(m.diffResult.File2Name, 25), m.gitCtx.Ref2)
	}
	return m.styles.title.Render(title)
}

// renderDiff renders the diff content
func (m Model) renderDiff() string {
	// Calculate visible range
	start := m.viewport.offset
	end := min(start+m.viewport.height, len(m.diffResult.Lines))

	if start >= len(m.diffResult.Lines) {
		start = max(0, len(m.diffResult.Lines)-m.viewport.height)
		m.viewport.offset = start
		end = len(m.diffResult.Lines)
	}

	if start >= end || len(m.diffResult.Lines) == 0 {
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
		line := m.diffResult.Lines[i]
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
		line := m.diffResult.Lines[i]
		leftContent, rightContent := m.renderSideBySideLine(line, columnWidth)

		// Join left and right with separator
		combinedLine := leftContent + " │ " + rightContent
		if m.showBlame && m.gitCtx.Enabled {
			if blameText, ok := m.gitCtx.Blame[line.LineNo2]; ok && blameText != "" {
				combinedLine += "  " + m.styles.blame.Render(truncate(blameText, 60))
			}
		}
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

	if m.showBlame && m.gitCtx.Enabled {
		if blameText, ok := m.gitCtx.Blame[line.LineNo2]; ok && blameText != "" {
			parts = append(parts, "  "+m.styles.blame.Render(truncate(blameText, 60)))
		}
	}

	return strings.Join(parts, "")
}

// renderStatusBar renders the status bar
func (m Model) renderStatusBar() string {
	added, removed, unchanged := m.diffResult.GetStats()

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

	gitInfo := ""
	if m.gitCtx.Enabled {
		gitInfo = fmt.Sprintf(" | git: %s→%s", m.gitCtx.Ref1, m.gitCtx.Ref2)
	}

	status := fmt.Sprintf(
		"Lines: +%d -%d =%d | Pos: %d/%d | View: %s | Color: %s%s | v:view c:color s:stats ?:help q:quit",
		added, removed, unchanged,
		m.viewport.offset+1, len(m.diffResult.Lines),
		viewMode, syntaxMode, gitInfo,
	)

	return m.styles.statusBar.Width(m.width).Render(status)
}

func (m Model) renderActivePanel() string {
	switch m.activePanel {
	case helpPanel:
		return m.renderHelpPanel()
	case statsPanel:
		return m.renderStatsPanel()
	case statusPanel:
		return m.renderGitStatusPanel()
	case branchPanel:
		return m.renderBranchesPanel()
	case historyPanel:
		return m.renderHistoryPanel()
	default:
		return ""
	}
}

// renderHelpPanel renders the help panel below the main view
func (m Model) renderHelpPanel() string {
	helpText := []string{
		"",
		"Keyboard Shortcuts:",
		"  j, ↓      Scroll down     │  g         Go to top        │  v    Toggle side-by-side",
		"  k, ↑      Scroll up       │  G         Go to bottom     │  c    Toggle syntax colors",
		"  d         Half page down  │  s         Toggle stats     │  b    Toggle blame",
		"  u         Half page up    │  h, ?      Toggle help      │  q    Quit",
		"  S         Git status      │  B         Branch switcher  │  H    Commit history",
		"  [ / ]     Cycle branches  │",
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
	added, removed, unchanged := m.diffResult.GetStats()
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
			truncate(m.diffResult.File1Name, 35),
			truncate(m.diffResult.File2Name, 35)),
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

func (m Model) renderGitStatusPanel() string {
	if !m.gitCtx.Enabled {
		return m.styles.help.Render("Git repository not detected - status unavailable")
	}

	if len(m.gitCtx.Status) == 0 {
		return m.styles.help.Render("Working tree clean")
	}

	content := append([]string{"Git Status", "─────────"}, m.gitCtx.Status...)
	return m.styles.help.Copy().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.config.Theme.BorderFg).
		Padding(0, 1).
		Width(m.width - 2).
		Render(strings.Join(content, "\n"))
}

func (m Model) renderBranchesPanel() string {
	if !m.gitCtx.Enabled {
		return m.styles.help.Render("Git repository not detected - branches unavailable")
	}

	lines := []string{"Branches", "────────"}

	for i, br := range m.gitCtx.Branches {
		marker := " "
		if br == m.gitCtx.CurrentBranch {
			marker = "*"
		}
		selector := " "
		if i == m.branchIndex {
			selector = ">"
		}
		lines = append(lines, fmt.Sprintf("%s%s %s", selector, marker, br))
	}

	return m.styles.help.Copy().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.config.Theme.BorderFg).
		Padding(0, 1).
		Width(m.width - 2).
		Render(strings.Join(lines, "\n"))
}

func (m Model) renderHistoryPanel() string {
	if !m.gitCtx.Enabled {
		return m.styles.help.Render("Git repository not detected - history unavailable")
	}

	lines := []string{"Recent Commits", "────────────"}
	lines = append(lines, m.gitCtx.CommitHistory...)

	return m.styles.help.Copy().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.config.Theme.BorderFg).
		Padding(0, 1).
		Width(m.width - 2).
		Render(strings.Join(lines, "\n"))
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

func (m *Model) togglePanel(target panelType) {
	if m.activePanel == target {
		m.activePanel = noPanel
	} else {
		m.activePanel = target
	}

	m.showHelp = m.activePanel == helpPanel
	m.showStats = m.activePanel == statsPanel
	m.updateViewportHeight()
}

func (m *Model) selectNextBranch() {
	if !m.gitCtx.Enabled || len(m.gitCtx.Branches) == 0 {
		return
	}
	m.branchIndex = (m.branchIndex + 1) % len(m.gitCtx.Branches)
	m.gitCtx.Ref2 = m.gitCtx.Branches[m.branchIndex]
	m.reloadDiff()
}

func (m *Model) selectPreviousBranch() {
	if !m.gitCtx.Enabled || len(m.gitCtx.Branches) == 0 {
		return
	}
	m.branchIndex--
	if m.branchIndex < 0 {
		m.branchIndex = len(m.gitCtx.Branches) - 1
	}
	m.gitCtx.Ref2 = m.gitCtx.Branches[m.branchIndex]
	m.reloadDiff()
}

func (m *Model) reloadDiff() {
	if m.diffEngine == nil || !m.gitCtx.Enabled {
		return
	}

	lines1, err := m.readLinesForRef(m.gitCtx.Ref1)
	if err != nil {
		m.err = err
		return
	}
	lines2, err := m.readLinesForRef(m.gitCtx.Ref2)
	if err != nil {
		m.err = err
		return
	}

	leftLabel := fmt.Sprintf("%s:%s", m.gitCtx.Ref1, m.gitCtx.FilePath)
	rightLabel := fmt.Sprintf("%s:%s", m.gitCtx.Ref2, m.gitCtx.FilePath)

	m.diffResult = m.diffEngine.DiffLines(lines1, lines2, leftLabel, rightLabel)
	if m.showBlame {
		m.gitCtx.Blame, _ = m.collectBlame()
	}
}

func (m *Model) readLinesForRef(ref string) ([]string, error) {
	if ref == "" || ref == "WORKTREE" {
		data, err := os.ReadFile(filepath.Join(m.gitCtx.RepoRoot, m.gitCtx.FilePath))
		if err != nil {
			return nil, err
		}
		text := strings.TrimSuffix(string(data), "\n")
		if text == "" {
			return []string{}, nil
		}
		return strings.Split(text, "\n"), nil
	}

	cmd := exec.Command("git", "-C", m.gitCtx.RepoRoot, "show", fmt.Sprintf("%s:%s", ref, m.gitCtx.FilePath))
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	text := strings.TrimSuffix(string(out), "\n")
	if text == "" {
		return []string{}, nil
	}

	return strings.Split(text, "\n"), nil
}

func (m *Model) collectBlame() (map[int]string, error) {
	blame := make(map[int]string)
	if !m.gitCtx.Enabled {
		return blame, nil
	}

	target := m.gitCtx.FilePath
	if m.gitCtx.Ref2 != "" && m.gitCtx.Ref2 != "WORKTREE" {
		target = fmt.Sprintf("%s:%s", m.gitCtx.Ref2, m.gitCtx.FilePath)
	}

	cmd := exec.Command("git", "-C", m.gitCtx.RepoRoot, "blame", "-l", target)
	out, err := cmd.Output()
	if err != nil {
		return blame, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for i, line := range lines {
		blame[i+1] = line
	}

	return blame, nil
}

// updateViewportHeight calculates and sets the viewport height based on screen size and active panels
func (m *Model) updateViewportHeight() {
	// Base height: total - title bar - status bar
	baseHeight := m.height - 2

	// Subtract panel height if help or stats is shown
	switch m.activePanel {
	case helpPanel:
		baseHeight -= m.helpPanelHeight
	case statsPanel, statusPanel, branchPanel, historyPanel:
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
