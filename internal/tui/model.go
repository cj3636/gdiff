package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/cj3636/gdiff/internal/config"
	"github.com/cj3636/gdiff/internal/diff"
	"github.com/cj3636/gdiff/internal/export"
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
	showCommand      bool
	sideBySideMode   bool
	syntaxHighlight  bool
	showBlame        bool
	err              error
	helpPanelHeight  int
	statsPanelHeight int
	commandHeight    int
	activePanel      panelType
	gitCtx           GitContext
	branchIndex      int
	paletteEntries   []paletteEntry
	paletteIndex     int
	goToLineActive   bool
	goToLineValue    string
	goToLineError    string
	wrapLines        bool
	minimapWidth     int
	minimapStartCol  int
	minimapHeight    int
	statusMessage    string
}

type paletteEntry struct {
	section      string
	label        string
	description  string
	action       paletteAction
	offsetTarget int
	format       export.Format
}

type paletteAction int

const (
	paletteActionNone paletteAction = iota
	paletteActionToggleHelp
	paletteActionToggleStats
	paletteActionToggleSideBySide
	paletteActionToggleSyntax
	paletteActionToggleBlame
	paletteActionToggleWrap
	paletteActionGoTop
	paletteActionGoBottom
	paletteActionGoToLine
	paletteActionJumpOffset
	paletteActionCopyDiff
	paletteActionSaveDiff
)

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
	selection  lipgloss.Style
	section    lipgloss.Style
	minimapAdd lipgloss.Style
	minimapDel lipgloss.Style
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
		showCommand:      false,
		sideBySideMode:   false,
		syntaxHighlight:  true, // Default to enabled
		showBlame:        gitCtx.ShowBlame,
		helpPanelHeight:  12,
		statsPanelHeight: 17,
		commandHeight:    16,
		gitCtx:           gitCtx,
		wrapLines:        false,
		minimapWidth:     14,
	}

	if gitCtx.Enabled {
		for i, b := range gitCtx.Branches {
			if b == gitCtx.Ref2 {
				model.branchIndex = i
				break
			}
		}
	}

	model.refreshPaletteEntries()
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
		selection: lipgloss.NewStyle().
			Foreground(theme.TitleBg).
			Background(theme.TitleFg).
			Bold(true),
		section: lipgloss.NewStyle().
			Foreground(theme.TitleFg).
			Bold(true),
		minimapAdd: lipgloss.NewStyle().Foreground(theme.AddedFg),
		minimapDel: lipgloss.NewStyle().Foreground(theme.RemovedFg),
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
		if m.goToLineActive {
			m.handleGoToLineInput(msg)
			return m, nil
		}

		if m.showCommand {
			m.handlePaletteInput(msg)
			return m, nil
		}

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
		case "p":
			m.toggleCommandPalette()
		case "v":
			m.sideBySideMode = !m.sideBySideMode
		case "c":
			m.syntaxHighlight = !m.syntaxHighlight
		case "w":
			m.wrapLines = !m.wrapLines
		case "b":
			m.showBlame = !m.showBlame
			if m.showBlame && m.gitCtx.Enabled && m.gitCtx.Blame == nil {
				m.gitCtx.Blame, m.err = m.collectBlame()
			}
		case "y":
			m.copyDiff(export.FormatMarkdown)
		case "o":
			m.saveDiff(export.FormatHTML)
		case "<":
			m.adjustMinimapWidth(-2)
		case ">":
			m.adjustMinimapWidth(2)
		case "n":
			m.jumpToNextChange()
		case "N":
			m.jumpToPreviousChange()
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
		case "L":
			m.openGoToLineDialog()
		case "[":
			m.selectPreviousBranch()
		case "]":
			m.selectNextBranch()
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.updateViewportHeight()
	case tea.MouseMsg:
		m.handleMouse(msg)
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

	if m.showCommand {
		sections = append(sections, m.renderCommandPalette())
	}

	if m.goToLineActive {
		sections = append(sections, m.renderGoToLineDialog())
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

	contentWidth := m.availableContentWidth()
	var lines []string
	if m.sideBySideMode {
		lines = m.renderSideBySideLines(start, end, contentWidth)
	} else {
		lines = m.renderUnifiedLines(start, end, contentWidth)
	}

	lines = m.padLines(lines, m.viewport.height)
	mainView := lipgloss.NewStyle().Width(contentWidth).Render(strings.Join(lines, "\n"))
	minimap := m.renderMinimap()
	return lipgloss.JoinHorizontal(lipgloss.Top, mainView, minimap)
}

func (m *Model) availableContentWidth() int {
	width := m.width - m.minimapWidth
	if width < 20 {
		width = 20
	}
	m.minimapStartCol = width + 1
	m.minimapHeight = m.viewport.height
	return width
}

func (m Model) renderUnifiedLines(start, end, contentWidth int) []string {
	var lines []string

	for i := start; i < end; i++ {
		prefix, style, content := m.buildUnifiedLineParts(m.diffResult.Lines[i])
		available := contentWidth - lipgloss.Width(prefix)
		if available < 10 {
			available = 10
		}

		wrapped := []string{content}
		if m.wrapLines {
			wrapped = wrapText(content, available)
		}

		for _, part := range wrapped {
			trimmed := truncateWidth(part, available)
			lines = append(lines, prefix+style.Render(trimmed))
		}
	}

	return lines
}

func (m Model) renderSideBySideLines(start, end, contentWidth int) []string {
	var lines []string

	columnWidth := (contentWidth - 3) / 2
	if columnWidth < 20 {
		columnWidth = 20
	}

	for i := start; i < end; i++ {
		line := m.diffResult.Lines[i]
		leftContent, rightContent := m.renderSideBySideLine(line, columnWidth)
		combinedLine := leftContent + " │ " + rightContent
		if m.showBlame && m.gitCtx.Enabled {
			if blameText, ok := m.gitCtx.Blame[line.LineNo2]; ok && blameText != "" {
				combinedLine += "  " + m.styles.blame.Render(truncate(blameText, 60))
			}
		}

		lines = append(lines, truncateWidth(combinedLine, contentWidth))
	}

	return lines
}

func (m Model) padLines(lines []string, target int) []string {
	for len(lines) < target {
		lines = append(lines, "")
	}
	return lines
}

func (m Model) renderMinimap() string {
	height := m.viewport.height
	if height < 1 {
		height = 1
	}
	m.minimapHeight = height

	width := m.minimapWidth
	if width < 6 {
		width = 6
	}

	total := len(m.diffResult.Lines)
	if total == 0 {
		return ""
	}

	type bucket struct {
		added   int
		removed int
		equal   int
	}

	buckets := make([]bucket, height)
	for idx, line := range m.diffResult.Lines {
		row := int(float64(idx) / float64(max(total, 1)) * float64(height))
		if row >= height {
			row = height - 1
		}
		switch line.Type {
		case diff.Added:
			buckets[row].added++
		case diff.Removed:
			buckets[row].removed++
		default:
			buckets[row].equal++
		}
	}

	viewStart := int(float64(m.viewport.offset) / float64(max(total, 1)) * float64(height))
	viewEnd := int(float64(min(m.viewport.offset+m.viewport.height, total)) / float64(max(total, 1)) * float64(height))
	if viewEnd >= height {
		viewEnd = height - 1
	}

	divider := m.styles.border.Render("│")
	var rows []string
	for i := 0; i < height; i++ {
		bucket := buckets[i]
		indicator := strings.Repeat("▐", width-1)

		style := m.styles.unchanged
		switch {
		case bucket.added >= bucket.removed && bucket.added > bucket.equal:
			style = m.styles.minimapAdd
		case bucket.removed > bucket.added && bucket.removed > bucket.equal:
			style = m.styles.minimapDel
		}

		if i >= viewStart && i <= viewEnd {
			style = m.styles.selection
		}

		rows = append(rows, divider+style.Width(width-1).Render(indicator))
	}

	return strings.Join(rows, "\n")
}

func (m Model) buildUnifiedLineParts(line diff.DiffLine) (string, lipgloss.Style, string) {
	var parts []string

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

	var symbol string
	var style lipgloss.Style

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
	return strings.Join(parts, ""), style, content
}

func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	var lines []string
	var builder strings.Builder
	currentWidth := 0
	for _, r := range text {
		runeWidth := lipgloss.Width(string(r))
		if currentWidth+runeWidth > width {
			lines = append(lines, builder.String())
			builder.Reset()
			currentWidth = 0
		}
		builder.WriteRune(r)
		currentWidth += runeWidth
	}

	if builder.Len() > 0 {
		lines = append(lines, builder.String())
	}

	if len(lines) == 0 {
		return []string{""}
	}

	return lines
}

func truncateWidth(text string, width int) string {
	if width <= 0 {
		return ""
	}

	if lipgloss.Width(text) <= width {
		return text
	}

	var builder strings.Builder
	current := 0
	for _, r := range text {
		runeWidth := lipgloss.Width(string(r))
		if current+runeWidth > width-3 {
			break
		}
		builder.WriteRune(r)
		current += runeWidth
	}

	return builder.String() + "..."
}

func (m *Model) adjustMinimapWidth(delta int) {
	m.minimapWidth += delta
	if m.minimapWidth < 6 {
		m.minimapWidth = 6
	}
	maxWidth := m.width / 3
	if maxWidth < 10 {
		maxWidth = 10
	}
	if m.minimapWidth > maxWidth {
		m.minimapWidth = maxWidth
	}
}

func (m *Model) lineForMinimapRow(row int) int {
	total := len(m.diffResult.Lines)
	if total == 0 {
		return 0
	}

	if row < 0 {
		row = 0
	}
	if row >= m.minimapHeight {
		row = m.minimapHeight - 1
	}

	fraction := float64(row) / float64(max(m.minimapHeight, 1))
	line := int(fraction * float64(total))
	if line >= total {
		line = total - 1
	}
	return line
}

func (m *Model) handleMouse(msg tea.MouseMsg) {
	if msg.Action != tea.MouseActionPress && msg.Action != tea.MouseActionRelease {
		return
	}

	if m.minimapStartCol == 0 || msg.X < m.minimapStartCol {
		return
	}

	mapTop := 2 // Title occupies first line
	if msg.Y < mapTop || msg.Y >= mapTop+m.minimapHeight {
		return
	}

	targetRow := msg.Y - mapTop
	line := m.lineForMinimapRow(targetRow)
	m.jumpToOffset(line)
}

func (m *Model) jumpToNextChange() {
	changes := m.changeOffsets()
	for _, off := range changes {
		if off > m.viewport.offset {
			m.jumpToOffset(off)
			return
		}
	}

	if len(changes) > 0 {
		m.jumpToOffset(changes[0])
	}
}

func (m *Model) jumpToPreviousChange() {
	changes := m.changeOffsets()
	for i := len(changes) - 1; i >= 0; i-- {
		if changes[i] < m.viewport.offset {
			m.jumpToOffset(changes[i])
			return
		}
	}

	if len(changes) > 0 {
		m.jumpToOffset(changes[len(changes)-1])
	}
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

	wrapMode := "off"
	if m.wrapLines {
		wrapMode = "on"
	}

	gitInfo := ""
	if m.gitCtx.Enabled {
		gitInfo = fmt.Sprintf(" | git: %s→%s", m.gitCtx.Ref1, m.gitCtx.Ref2)
	}

	status := fmt.Sprintf(
		"Lines: +%d -%d =%d | Pos: %d/%d | View: %s | Wrap: %s | Color: %s%s | v:view c:color w:wrap <:map- >:map+ n/N:changes s:stats ?:help q:quit",
		added, removed, unchanged,
		m.viewport.offset+1, len(m.diffResult.Lines),
		viewMode, wrapMode, syntaxMode, gitInfo,
	)

	if m.statusMessage != "" {
		status = fmt.Sprintf("%s | %s", status, m.statusMessage)
	}

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
		"  u         Half page up    │  y         Copy diff        │  o    Save diff (HTML)",
		"  p         Command palette │  L         Go to line       │  g↵   Palette go-to-line",
		"  w         Toggle wrapping │  S         Git status       │  B    Branch switcher",
		"  H         Commit history  │  [ / ]     Cycle branches   │  < / > Resize minimap",
		"  n / N     Next/prev change│  Mouse     Jump via minimap │  q    Quit",
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

func (m Model) renderCommandPalette() string {
	if len(m.paletteEntries) == 0 {
		return ""
	}

	currentSection := ""
	var lines []string
	lines = append(lines, " Command Palette")

	for i, entry := range m.paletteEntries {
		if entry.section != currentSection {
			lines = append(lines, "")
			lines = append(lines, m.styles.section.Render(entry.section))
			currentSection = entry.section
		}

		label := fmt.Sprintf("%s  %s", entry.label, entry.description)
		if i == m.paletteIndex {
			label = m.styles.selection.Render("> " + label)
		} else {
			label = "  " + label
		}
		lines = append(lines, label)
	}

	style := m.styles.help.Copy().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.config.Theme.BorderFg).
		Padding(0, 1).
		Width(m.width - 2)

	return style.Render(strings.Join(lines, "\n"))
}

func (m Model) renderGoToLineDialog() string {
	content := fmt.Sprintf("Go to line: %s", m.goToLineValue)
	if m.goToLineError != "" {
		content += "  " + m.styles.removed.Render(m.goToLineError)
	}
	style := m.styles.help.Copy().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.config.Theme.BorderFg).
		Padding(0, 1).
		Width(m.width - 2)

	return style.Render(content)
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

func (m *Model) toggleCommandPalette() {
	m.showCommand = !m.showCommand
	m.activePanel = noPanel
	m.goToLineActive = false
	m.refreshPaletteEntries()
	m.updateViewportHeight()
}

func (m *Model) handlePaletteInput(msg tea.KeyMsg) {
	switch msg.String() {
	case "esc", "q":
		m.showCommand = false
		m.updateViewportHeight()
	case "up", "k":
		m.movePaletteSelection(-1)
	case "down", "j":
		m.movePaletteSelection(1)
	case "enter", " ":
		m.executePaletteSelection()
	}
}

func (m *Model) movePaletteSelection(delta int) {
	if len(m.paletteEntries) == 0 {
		return
	}
	m.paletteIndex += delta
	if m.paletteIndex < 0 {
		m.paletteIndex = 0
	}
	if m.paletteIndex >= len(m.paletteEntries) {
		m.paletteIndex = len(m.paletteEntries) - 1
	}
}

func (m *Model) executePaletteSelection() {
	if len(m.paletteEntries) == 0 {
		return
	}

	entry := m.paletteEntries[m.paletteIndex]
	switch entry.action {
	case paletteActionToggleHelp:
		m.togglePanel(helpPanel)
	case paletteActionToggleStats:
		m.togglePanel(statsPanel)
	case paletteActionToggleSideBySide:
		m.sideBySideMode = !m.sideBySideMode
	case paletteActionToggleSyntax:
		m.syntaxHighlight = !m.syntaxHighlight
	case paletteActionToggleBlame:
		m.showBlame = !m.showBlame
		if m.showBlame && m.gitCtx.Enabled && m.gitCtx.Blame == nil {
			m.gitCtx.Blame, m.err = m.collectBlame()
		}
	case paletteActionToggleWrap:
		m.wrapLines = !m.wrapLines
	case paletteActionGoTop:
		m.scrollToTop()
	case paletteActionGoBottom:
		m.scrollToBottom()
	case paletteActionGoToLine:
		m.openGoToLineDialog()
	case paletteActionJumpOffset:
		m.jumpToOffset(entry.offsetTarget)
	case paletteActionCopyDiff:
		m.copyDiff(entry.format)
	case paletteActionSaveDiff:
		m.saveDiff(entry.format)
	}

	if entry.action != paletteActionGoToLine {
		m.showCommand = false
	}
	m.refreshPaletteEntries()
	m.updateViewportHeight()
}

func (m *Model) refreshPaletteEntries() {
	var entries []paletteEntry

	entries = append(entries,
		paletteEntry{section: "Commands", label: "Toggle help", description: "? / h", action: paletteActionToggleHelp},
		paletteEntry{section: "Commands", label: "Toggle stats", description: "s", action: paletteActionToggleStats},
		paletteEntry{section: "Commands", label: "Toggle side-by-side", description: "v", action: paletteActionToggleSideBySide},
		paletteEntry{section: "Commands", label: "Toggle syntax colors", description: "c", action: paletteActionToggleSyntax},
		paletteEntry{section: "Commands", label: "Toggle wrapping", description: "w", action: paletteActionToggleWrap},
		paletteEntry{section: "Commands", label: "Toggle blame", description: "b", action: paletteActionToggleBlame},
		paletteEntry{section: "Commands", label: "Go to top", description: "g", action: paletteActionGoTop},
		paletteEntry{section: "Commands", label: "Go to bottom", description: "G", action: paletteActionGoBottom},
		paletteEntry{section: "Commands", label: "Go to line", description: "L", action: paletteActionGoToLine},
		paletteEntry{section: "Export", label: "Copy diff (Markdown)", description: "y", action: paletteActionCopyDiff, format: export.FormatMarkdown},
		paletteEntry{section: "Export", label: "Copy diff (ANSI)", description: "command palette", action: paletteActionCopyDiff, format: export.FormatANSI},
		paletteEntry{section: "Export", label: "Save diff (HTML)", description: "o", action: paletteActionSaveDiff, format: export.FormatHTML},
	)

	for _, offset := range m.changeOffsets() {
		if offset < 0 || offset >= len(m.diffResult.Lines) {
			continue
		}
		line := m.diffResult.Lines[offset]
		displayNo := displayLineNumber(line)
		snippet := truncate(strings.TrimSpace(line.Content), 60)
		entries = append(entries, paletteEntry{
			section:      "Changes",
			label:        fmt.Sprintf("Change at line %d", displayNo),
			description:  snippet,
			action:       paletteActionJumpOffset,
			offsetTarget: offset,
		})
	}

	for _, ln := range m.lineAnchors() {
		offset := m.offsetForLine(ln)
		entries = append(entries, paletteEntry{
			section:      "Lines",
			label:        fmt.Sprintf("Line %d", ln),
			description:  "Jump to line",
			action:       paletteActionJumpOffset,
			offsetTarget: offset,
		})
	}

	m.paletteEntries = entries
	if m.paletteIndex >= len(m.paletteEntries) {
		m.paletteIndex = max(0, len(m.paletteEntries)-1)
	}
}

func (m *Model) copyDiff(format export.Format) {
	if m.diffResult == nil {
		return
	}

	content := m.exportDiff(format)
	if content == "" {
		return
	}

	if err := export.CopyToClipboard(content, os.Stdout); err != nil {
		m.err = err
		return
	}

	m.statusMessage = fmt.Sprintf("Copied %s diff to clipboard", formatLabel(format))
}

func (m *Model) saveDiff(format export.Format) {
	if m.diffResult == nil {
		return
	}

	content := m.exportDiff(format)
	if content == "" {
		return
	}

	filename := m.defaultExportFilename(format)
	if err := os.WriteFile(filename, []byte(content), 0o644); err != nil {
		m.err = err
		return
	}

	m.statusMessage = fmt.Sprintf("Saved %s diff to %s", formatLabel(format), filename)
}

func (m *Model) exportDiff(format export.Format) string {
	if format == "" {
		format = export.FormatMarkdown
	}

	content, err := export.Render(m.diffResult, format, export.Options{
		Title:           m.exportTitle(),
		ShowLineNumbers: m.config.ShowLineNo,
	})
	if err != nil {
		m.err = err
		return ""
	}

	return content
}

func (m Model) exportTitle() string {
	if m.diffResult == nil {
		return ""
	}
	return fmt.Sprintf("%s ↔ %s", m.diffResult.File1Name, m.diffResult.File2Name)
}

func (m Model) defaultExportFilename(format export.Format) string {
	ext := "txt"
	switch format {
	case export.FormatHTML:
		ext = "html"
	case export.FormatMarkdown:
		ext = "md"
	case export.FormatANSI:
		ext = "txt"
	}

	left := sanitizeFilename(filepath.Base(m.diffResult.File1Name))
	right := sanitizeFilename(filepath.Base(m.diffResult.File2Name))
	if left == "" {
		left = "file1"
	}
	if right == "" {
		right = "file2"
	}

	return fmt.Sprintf("gdiff_%s_vs_%s.%s", left, right, ext)
}

func sanitizeFilename(name string) string {
	cleaned := strings.Map(func(r rune) rune {
		switch r {
		case '/', '\\', ':', '*', '?', '"', '<', '>', '|':
			return '_'
		default:
			return r
		}
	}, name)

	return strings.Trim(cleaned, " ")
}

func formatLabel(f export.Format) string {
	switch f {
	case export.FormatHTML:
		return "HTML"
	case export.FormatANSI:
		return "ANSI"
	default:
		return "Markdown"
	}
}

func (m *Model) changeOffsets() []int {
	if m.diffResult == nil {
		return nil
	}
	var offsets []int
	prevChange := false

	for idx, line := range m.diffResult.Lines {
		isChange := line.Type != diff.Equal
		if isChange && !prevChange {
			offsets = append(offsets, idx)
		}
		prevChange = isChange
	}
	return offsets
}

func (m *Model) lineAnchors() []int {
	if m.diffResult == nil {
		return nil
	}

	total := len(m.diffResult.File2Lines)
	if total == 0 {
		total = len(m.diffResult.Lines)
	}
	if total == 0 {
		return nil
	}

	step := total / 6
	if step < 10 {
		step = 10
	}

	anchors := []int{1}
	for i := step; i < total; i += step {
		anchors = append(anchors, i)
	}
	anchors = append(anchors, total)

	unique := make(map[int]struct{})
	var result []int
	for _, v := range anchors {
		if _, ok := unique[v]; ok {
			continue
		}
		unique[v] = struct{}{}
		result = append(result, v)
	}

	return result
}

func (m *Model) openGoToLineDialog() {
	m.goToLineActive = true
	m.goToLineValue = ""
	m.goToLineError = ""
	m.showCommand = false
	m.updateViewportHeight()
}

func (m *Model) handleGoToLineInput(msg tea.KeyMsg) {
	switch msg.Type {
	case tea.KeyEsc:
		m.goToLineActive = false
		m.goToLineValue = ""
		m.goToLineError = ""
		m.updateViewportHeight()
	case tea.KeyEnter:
		m.applyGoToLine()
	case tea.KeyBackspace, tea.KeyDelete:
		if len(m.goToLineValue) > 0 {
			m.goToLineValue = m.goToLineValue[:len(m.goToLineValue)-1]
		}
	default:
		if len(msg.Runes) > 0 {
			for _, r := range msg.Runes {
				if r >= '0' && r <= '9' {
					m.goToLineValue += string(r)
				}
			}
		}
	}
}

func (m *Model) applyGoToLine() {
	if m.goToLineValue == "" {
		m.goToLineError = "Enter a line number"
		return
	}

	lineNumber, err := strconv.Atoi(m.goToLineValue)
	if err != nil || lineNumber < 1 {
		m.goToLineError = "Invalid line"
		return
	}

	offset := m.offsetForLine(lineNumber)
	m.jumpToOffset(offset)
	m.goToLineActive = false
	m.goToLineError = ""
	m.updateViewportHeight()
}

func (m *Model) offsetForLine(lineNumber int) int {
	if m.diffResult == nil {
		return 0
	}

	for idx, line := range m.diffResult.Lines {
		displayNo := displayLineNumber(line)
		if displayNo >= lineNumber && displayNo > 0 {
			return idx
		}
	}

	if len(m.diffResult.Lines) == 0 {
		return 0
	}
	return len(m.diffResult.Lines) - 1
}

func (m *Model) jumpToOffset(offset int) {
	if m.diffResult == nil {
		return
	}
	maxOffset := max(0, len(m.diffResult.Lines)-m.viewport.height)
	if offset < 0 {
		offset = 0
	}
	if offset > maxOffset {
		offset = maxOffset
	}
	m.viewport.offset = offset
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
	m.showCommand = false
	m.goToLineActive = false
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

	m.refreshPaletteEntries()
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

	if m.showCommand {
		baseHeight -= min(m.commandHeight, len(m.paletteEntries)+4)
	}

	if m.goToLineActive {
		baseHeight -= 3
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

func displayLineNumber(line diff.DiffLine) int {
	if line.LineNo2 > 0 {
		return line.LineNo2
	}
	if line.LineNo1 > 0 {
		return line.LineNo1
	}
	return 0
}
