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
)

// Model represents the application state
type Model struct {
	diffResult       *diff.DiffResult
	config           *config.Config
	keybindings      config.Keybindings
	overrideKeys     config.Keybindings
	useOverrides     bool
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
	settingsEntries  []settingsEntry
	settingsIndex    int
	showSettings     bool
	goToLineActive   bool
	goToLineValue    string
	goToLineError    string
	wrapLines        bool
	minimapWidth     int
	minimapStartCol  int
	minimapHeight    int
}

type settingsEntry struct {
	section     string
	label       string
	description string
	action      settingsAction
}

type settingsAction int

const (
	settingsActionTheme settingsAction = iota
	settingsActionContrast
	settingsActionLineNumbers
	settingsActionLineNumberWidth
	settingsActionLinePadding
	settingsActionLineSpacing
	settingsActionKeybindings
)

const (
	actionQuit              = "quit"
	actionToggleHelp        = "toggle_help"
	actionToggleStats       = "toggle_stats"
	actionToggleStatus      = "toggle_status"
	actionToggleBranches    = "toggle_branches"
	actionToggleHistory     = "toggle_history"
	actionTogglePalette     = "toggle_palette"
	actionToggleSettings    = "toggle_settings"
	actionToggleSideBySide  = "toggle_side_by_side"
	actionToggleSyntax      = "toggle_syntax"
	actionToggleWrap        = "toggle_wrap"
	actionToggleBlame       = "toggle_blame"
	actionToggleLineNumbers = "toggle_line_numbers"
	actionMinimapNarrow     = "minimap_narrow"
	actionMinimapWiden      = "minimap_widen"
	actionNextChange        = "next_change"
	actionPrevChange        = "prev_change"
	actionScrollDown        = "scroll_down"
	actionScrollUp          = "scroll_up"
	actionPageDown          = "page_down"
	actionPageUp            = "page_up"
	actionGoTop             = "go_top"
	actionGoBottom          = "go_bottom"
	actionGoLine            = "go_line"
	actionPrevBranch        = "prev_branch"
	actionNextBranch        = "next_branch"
)

type paletteEntry struct {
	section      string
	label        string
	description  string
	action       paletteAction
	offsetTarget int
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
	paletteActionOpenSettings
	paletteActionGoTop
	paletteActionGoBottom
	paletteActionGoToLine
	paletteActionJumpOffset
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
	if cfg.Keybindings == nil {
		cfg.Keybindings = config.Keybindings{}
	}

	cfg.Theme = config.ThemeForPreset(cfg.ThemePreset, cfg.HighContrast)

	styles := createStyles(cfg)
	overrides := copyKeybindings(cfg.Keybindings)
	model := Model{
		diffResult:       diffResult,
		config:           cfg,
		keybindings:      config.MergeKeybindings(overrides),
		overrideKeys:     overrides,
		useOverrides:     len(overrides) > 0,
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
	model.refreshSettingsEntries()
	return model
}

// createStyles initializes all lipgloss styles based on theme
func createStyles(cfg *config.Config) *Styles {
	theme := cfg.Theme
	padding := cfg.Spacing.LinePadding
	gutterWidth := cfg.Spacing.LineNumberWidth
	if gutterWidth < 4 {
		gutterWidth = 4
	}

	return &Styles{
		added: lipgloss.NewStyle().
			Foreground(theme.AddedFg).
			Background(theme.AddedBg).
			Padding(0, padding),
		removed: lipgloss.NewStyle().
			Foreground(theme.RemovedFg).
			Background(theme.RemovedBg).
			Padding(0, padding),
		unchanged: lipgloss.NewStyle().
			Foreground(theme.UnchangedFg).
			Padding(0, padding),
		lineNumber: lipgloss.NewStyle().
			Foreground(theme.LineNumberFg).
			Width(gutterWidth).
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

		if m.showSettings {
			m.handleSettingsInput(msg)
			return m, nil
		}

		switch {
		case m.matchesKey(actionQuit, msg):
			return m, tea.Quit
		case m.matchesKey(actionToggleHelp, msg):
			m.togglePanel(helpPanel)
		case m.matchesKey(actionToggleStats, msg):
			m.togglePanel(statsPanel)
		case m.matchesKey(actionToggleStatus, msg):
			m.togglePanel(statusPanel)
		case m.matchesKey(actionToggleBranches, msg):
			m.togglePanel(branchPanel)
		case m.matchesKey(actionToggleHistory, msg):
			m.togglePanel(historyPanel)
		case m.matchesKey(actionTogglePalette, msg):
			m.toggleCommandPalette()
		case m.matchesKey(actionToggleSettings, msg):
			m.toggleSettings()
		case m.matchesKey(actionToggleSideBySide, msg):
			m.sideBySideMode = !m.sideBySideMode
		case m.matchesKey(actionToggleSyntax, msg):
			m.syntaxHighlight = !m.syntaxHighlight
		case m.matchesKey(actionToggleWrap, msg):
			m.wrapLines = !m.wrapLines
		case m.matchesKey(actionToggleBlame, msg):
			m.showBlame = !m.showBlame
			if m.showBlame && m.gitCtx.Enabled && m.gitCtx.Blame == nil {
				m.gitCtx.Blame, m.err = m.collectBlame()
			}
		case m.matchesKey(actionToggleLineNumbers, msg):
			m.config.ShowLineNo = !m.config.ShowLineNo
		case m.matchesKey(actionMinimapNarrow, msg):
			m.adjustMinimapWidth(-2)
		case m.matchesKey(actionMinimapWiden, msg):
			m.adjustMinimapWidth(2)
		case m.matchesKey(actionNextChange, msg):
			m.jumpToNextChange()
		case m.matchesKey(actionPrevChange, msg):
			m.jumpToPreviousChange()
		case m.matchesKey(actionScrollDown, msg):
			m.scrollDown()
		case m.matchesKey(actionScrollUp, msg):
			m.scrollUp()
		case m.matchesKey(actionPageDown, msg):
			m.scrollPageDown()
		case m.matchesKey(actionPageUp, msg):
			m.scrollPageUp()
		case m.matchesKey(actionGoTop, msg):
			m.scrollToTop()
		case m.matchesKey(actionGoBottom, msg):
			m.scrollToBottom()
		case m.matchesKey(actionGoLine, msg):
			m.openGoToLineDialog()
		case m.matchesKey(actionPrevBranch, msg):
			m.selectPreviousBranch()
		case m.matchesKey(actionNextBranch, msg):
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

	if m.showSettings {
		sections = append(sections, m.renderSettingsModal())
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
			for s := 0; s < m.config.Spacing.LineSpacing; s++ {
				lines = append(lines, "")
			}
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
		for s := 0; s < m.config.Spacing.LineSpacing; s++ {
			lines = append(lines, "")
		}
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
		lineNo1, lineNo2 := m.lineNumberStrings(line)
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
		lineNo1, lineNo2 := m.lineNumberStrings(line)
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
		contentWidth = columnWidth - (m.lineNumberGutterWidth() + 1)
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
		lineNo1, lineNo2 := m.lineNumberStrings(line)
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

	lineNumbers := "off"
	if m.config.ShowLineNo {
		lineNumbers = "on"
	}

	themeLabel := string(m.config.ThemePreset)
	if m.config.HighContrast {
		themeLabel += "+hc"
	}

	gitInfo := ""
	if m.gitCtx.Enabled {
		gitInfo = fmt.Sprintf(" | git: %s→%s", m.gitCtx.Ref1, m.gitCtx.Ref2)
	}

	status := fmt.Sprintf(
		"Lines: +%d -%d =%d | Pos: %d/%d | View: %s | Wrap: %s | Color: %s | Theme: %s | Ln: %s | pad:%d space:%d%s | %s settings",
		added, removed, unchanged,
		m.viewport.offset+1, len(m.diffResult.Lines),
		viewMode, wrapMode, syntaxMode, themeLabel, lineNumbers, m.config.Spacing.LinePadding, m.config.Spacing.LineSpacing, gitInfo, m.keyDisplay(actionToggleSettings),
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
	helps := []string{
		"",
		"Keyboard Shortcuts:",
		fmt.Sprintf("  %-10s Scroll down     │  %-10s Go to top        │  %-6s Toggle side-by-side", m.keyDisplay(actionScrollDown), m.keyDisplay(actionGoTop), m.keyDisplay(actionToggleSideBySide)),
		fmt.Sprintf("  %-10s Scroll up       │  %-10s Go to bottom     │  %-6s Toggle syntax colors", m.keyDisplay(actionScrollUp), m.keyDisplay(actionGoBottom), m.keyDisplay(actionToggleSyntax)),
		fmt.Sprintf("  %-10s Half page down  │  %-10s Toggle stats     │  %-6s Toggle blame", m.keyDisplay(actionPageDown), m.keyDisplay(actionToggleStats), m.keyDisplay(actionToggleBlame)),
		fmt.Sprintf("  %-10s Half page up    │  %-10s Toggle wrapping  │  %-6s Quit", m.keyDisplay(actionPageUp), m.keyDisplay(actionToggleWrap), m.keyDisplay(actionQuit)),
		fmt.Sprintf("  %-10s Command palette │  %-10s Go to line       │  %-6s Settings", m.keyDisplay(actionTogglePalette), m.keyDisplay(actionGoLine), m.keyDisplay(actionToggleSettings)),
		fmt.Sprintf("  %-10s Git status      │  %-10s Branch switcher  │  %-6s Commit history", m.keyDisplay(actionToggleStatus), m.keyDisplay(actionToggleBranches), m.keyDisplay(actionToggleHistory)),
		fmt.Sprintf("  %-10s Cycle branches  │  %-10s Resize minimap", m.keyDisplay(actionPrevBranch)+" / "+m.keyDisplay(actionNextBranch), m.keyDisplay(actionMinimapNarrow)+" / "+m.keyDisplay(actionMinimapWiden)),
		fmt.Sprintf("  %-10s Next/prev change│  Mouse     Jump via minimap", m.keyDisplay(actionNextChange)+" / "+m.keyDisplay(actionPrevChange)),
		"",
	}

	// Create a bordered box for the help panel
	helpStyle := m.styles.help.Copy().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(m.config.Theme.BorderFg).
		Padding(0, 1).
		Width(m.width - 2)

	return helpStyle.Render(strings.Join(helps, "\n"))
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

func (m Model) renderSettingsModal() string {
	if len(m.settingsEntries) == 0 {
		return ""
	}

	currentSection := ""
	var lines []string
	lines = append(lines, " Settings")

	for i, entry := range m.settingsEntries {
		if entry.section != currentSection {
			lines = append(lines, "")
			lines = append(lines, m.styles.section.Render(entry.section))
			currentSection = entry.section
		}

		value := m.settingDescription(entry)
		label := fmt.Sprintf("%s  %s", entry.label, value)
		if i == m.settingsIndex {
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

func (m *Model) toggleSettings() {
	m.showSettings = !m.showSettings
	if m.showSettings {
		m.showCommand = false
		m.goToLineActive = false
		m.refreshSettingsEntries()
	} else {
		m.settingsIndex = 0
	}
	m.updateViewportHeight()
}

func (m *Model) refreshSettingsEntries() {
	m.settingsEntries = []settingsEntry{
		{section: "Theme", label: "Preset", action: settingsActionTheme},
		{section: "Theme", label: "High contrast", action: settingsActionContrast},
		{section: "Layout", label: "Line numbers", action: settingsActionLineNumbers},
		{section: "Layout", label: "Line number width", action: settingsActionLineNumberWidth},
		{section: "Layout", label: "Line padding", action: settingsActionLinePadding},
		{section: "Layout", label: "Line spacing", action: settingsActionLineSpacing},
		{section: "Input", label: "Keybindings", action: settingsActionKeybindings},
	}

	if m.settingsIndex >= len(m.settingsEntries) {
		m.settingsIndex = max(0, len(m.settingsEntries)-1)
	}
}

func (m Model) settingDescription(entry settingsEntry) string {
	switch entry.action {
	case settingsActionTheme:
		return string(m.config.ThemePreset)
	case settingsActionContrast:
		if m.config.HighContrast {
			return "On"
		}
		return "Off"
	case settingsActionLineNumbers:
		if m.config.ShowLineNo {
			return "Shown"
		}
		return "Hidden"
	case settingsActionLineNumberWidth:
		return fmt.Sprintf("%d cols", m.config.Spacing.LineNumberWidth)
	case settingsActionLinePadding:
		return fmt.Sprintf("%d spaces", m.config.Spacing.LinePadding)
	case settingsActionLineSpacing:
		return fmt.Sprintf("%d extra", m.config.Spacing.LineSpacing)
	case settingsActionKeybindings:
		if len(m.overrideKeys) == 0 {
			return "Defaults"
		}
		if m.useOverrides {
			return "Overrides"
		}
		return "Defaults"
	default:
		return ""
	}
}

func (m *Model) handleSettingsInput(msg tea.KeyMsg) {
	switch msg.String() {
	case "esc":
		m.showSettings = false
		m.updateViewportHeight()
		return
	case "up", "k":
		m.settingsIndex--
		if m.settingsIndex < 0 {
			m.settingsIndex = 0
		}
	case "down", "j":
		m.settingsIndex++
		if m.settingsIndex >= len(m.settingsEntries) {
			m.settingsIndex = len(m.settingsEntries) - 1
		}
	case "left", "h":
		m.applySettingsAction(-1)
	case "right", "l", "enter", " ":
		m.applySettingsAction(1)
	}
}

func (m *Model) applySettingsAction(direction int) {
	if len(m.settingsEntries) == 0 {
		return
	}

	entry := m.settingsEntries[m.settingsIndex]
	switch entry.action {
	case settingsActionTheme:
		presets := []config.ThemePreset{config.PresetDefault, config.PresetSolarize, config.PresetDracula}
		idx := 0
		for i, p := range presets {
			if p == m.config.ThemePreset {
				idx = i
				break
			}
		}
		idx = (idx + direction + len(presets)) % len(presets)
		m.config.ThemePreset = presets[idx]
		m.applyTheme()
	case settingsActionContrast:
		m.config.HighContrast = !m.config.HighContrast
		m.applyTheme()
	case settingsActionLineNumbers:
		m.config.ShowLineNo = !m.config.ShowLineNo
	case settingsActionLineNumberWidth:
		options := []int{4, 6, 8}
		m.config.Spacing.LineNumberWidth = cycleInt(options, m.config.Spacing.LineNumberWidth, direction)
		m.refreshStyles()
	case settingsActionLinePadding:
		options := []int{0, 1, 2}
		m.config.Spacing.LinePadding = cycleInt(options, m.config.Spacing.LinePadding, direction)
		m.refreshStyles()
	case settingsActionLineSpacing:
		options := []int{0, 1, 2}
		m.config.Spacing.LineSpacing = cycleInt(options, m.config.Spacing.LineSpacing, direction)
	case settingsActionKeybindings:
		if len(m.overrideKeys) == 0 {
			m.keybindings = config.DefaultKeybindings()
			m.useOverrides = false
			m.config.Keybindings = config.Keybindings{}
			break
		}
		m.useOverrides = !m.useOverrides
		if m.useOverrides {
			m.keybindings = config.MergeKeybindings(m.overrideKeys)
			m.config.Keybindings = copyKeybindings(m.overrideKeys)
		} else {
			m.keybindings = config.DefaultKeybindings()
			m.config.Keybindings = config.Keybindings{}
		}
	}

	m.refreshSettingsEntries()
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
	case paletteActionOpenSettings:
		m.toggleSettings()
	case paletteActionGoTop:
		m.scrollToTop()
	case paletteActionGoBottom:
		m.scrollToBottom()
	case paletteActionGoToLine:
		m.openGoToLineDialog()
	case paletteActionJumpOffset:
		m.jumpToOffset(entry.offsetTarget)
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
		paletteEntry{section: "Commands", label: "Settings", description: ",", action: paletteActionOpenSettings},
		paletteEntry{section: "Commands", label: "Toggle blame", description: "b", action: paletteActionToggleBlame},
		paletteEntry{section: "Commands", label: "Go to top", description: "g", action: paletteActionGoTop},
		paletteEntry{section: "Commands", label: "Go to bottom", description: "G", action: paletteActionGoBottom},
		paletteEntry{section: "Commands", label: "Go to line", description: "L", action: paletteActionGoToLine},
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
	m.showSettings = false
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

	if m.showSettings {
		baseHeight -= min(8, len(m.settingsEntries)+4)
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

func (m Model) matchesKey(action string, msg tea.KeyMsg) bool {
	keys := m.keybindings[action]
	if len(keys) == 0 {
		return false
	}
	for _, key := range keys {
		if msg.String() == key {
			return true
		}
	}
	return false
}

func (m Model) keyDisplay(action string) string {
	keys := m.keybindings[action]
	if len(keys) == 0 {
		return ""
	}
	return strings.Join(keys, "/")
}

func (m Model) lineNumberGutterWidth() int {
	width := m.config.Spacing.LineNumberWidth
	if width < 4 {
		width = 4
	}
	return width
}

func (m Model) lineNumberStrings(line diff.DiffLine) (string, string) {
	width := m.lineNumberGutterWidth() - 1
	if width < 2 {
		width = 2
	}
	blank := strings.Repeat(" ", width)
	format := fmt.Sprintf("%%%dd", width)

	lineNo1 := blank
	lineNo2 := blank
	if line.LineNo1 > 0 {
		lineNo1 = fmt.Sprintf(format, line.LineNo1)
	}
	if line.LineNo2 > 0 {
		lineNo2 = fmt.Sprintf(format, line.LineNo2)
	}
	return lineNo1, lineNo2
}

func (m *Model) applyTheme() {
	m.config.Theme = config.ThemeForPreset(m.config.ThemePreset, m.config.HighContrast)
	m.refreshStyles()
}

func (m *Model) refreshStyles() {
	m.styles = createStyles(m.config)
}

func copyKeybindings(src config.Keybindings) config.Keybindings {
	dup := config.Keybindings{}
	for action, keys := range src {
		copied := make([]string, len(keys))
		copy(copied, keys)
		dup[action] = copied
	}
	return dup
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

func cycleInt(options []int, current int, direction int) int {
	if len(options) == 0 {
		return current
	}
	idx := 0
	for i, v := range options {
		if v == current {
			idx = i
			break
		}
	}
	idx = (idx + direction + len(options)) % len(options)
	return options[idx]
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
