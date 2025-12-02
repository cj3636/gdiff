package config

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Config holds the application configuration
type Config struct {
	Theme            Theme
	ThemePreset      ThemePreset
	HighContrast     bool
	DiffMode         DiffMode
	ShowLineNo       bool
	TabSize          int
	IgnoreWhitespace bool
	Spacing          SpacingOptions
	Keybindings      Keybindings
}

// ThemePreset describes a named theme configuration.
type ThemePreset string

const (
	PresetDefault  ThemePreset = "default"
	PresetSolarize ThemePreset = "solarized"
	PresetDracula  ThemePreset = "dracula"
)

// SpacingOptions controls layout spacing and line number formatting.
type SpacingOptions struct {
	LinePadding     int
	LineSpacing     int
	LineNumberWidth int
}

// Keybindings maps semantic actions to one or more key sequences.
type Keybindings map[string][]string

// Theme defines the color scheme for the application
type Theme struct {
	AddedBg      lipgloss.Color
	AddedFg      lipgloss.Color
	RemovedBg    lipgloss.Color
	RemovedFg    lipgloss.Color
	UnchangedFg  lipgloss.Color
	LineNumberFg lipgloss.Color
	BorderFg     lipgloss.Color
	TitleFg      lipgloss.Color
	TitleBg      lipgloss.Color
	HelpFg       lipgloss.Color
}

// DiffMode specifies how differences should be displayed
type DiffMode int

const (
	SideBySide DiffMode = iota
	Unified
	Split
)

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		ThemePreset:      PresetDefault,
		Theme:            ThemeForPreset(PresetDefault, false),
		HighContrast:     false,
		DiffMode:         SideBySide,
		ShowLineNo:       true,
		TabSize:          4,
		IgnoreWhitespace: false,
		Spacing:          DefaultSpacing(),
		Keybindings:      DefaultKeybindings(),
	}
}

// DefaultTheme returns the default color theme
func DefaultTheme() Theme {
	return Theme{
		AddedBg:      lipgloss.Color("#2D4A2B"),
		AddedFg:      lipgloss.Color("#A8E6A3"),
		RemovedBg:    lipgloss.Color("#4A2D2D"),
		RemovedFg:    lipgloss.Color("#E6A3A3"),
		UnchangedFg:  lipgloss.Color("#B0B0B0"),
		LineNumberFg: lipgloss.Color("#666666"),
		BorderFg:     lipgloss.Color("#3A3A3A"),
		TitleFg:      lipgloss.Color("#FFFFFF"),
		TitleBg:      lipgloss.Color("#5F5FAF"),
		HelpFg:       lipgloss.Color("#888888"),
	}
}

// ThemeForPreset resolves a preset name to a concrete Theme, optionally
// applying a high-contrast variation.
func ThemeForPreset(preset ThemePreset, highContrast bool) Theme {
	switch preset {
	case PresetSolarize:
		return applyContrast(Theme{
			AddedBg:      lipgloss.Color("#073642"),
			AddedFg:      lipgloss.Color("#859900"),
			RemovedBg:    lipgloss.Color("#3C1F1E"),
			RemovedFg:    lipgloss.Color("#DC322F"),
			UnchangedFg:  lipgloss.Color("#93A1A1"),
			LineNumberFg: lipgloss.Color("#586E75"),
			BorderFg:     lipgloss.Color("#657B83"),
			TitleFg:      lipgloss.Color("#EEE8D5"),
			TitleBg:      lipgloss.Color("#586E75"),
			HelpFg:       lipgloss.Color("#93A1A1"),
		}, highContrast)
	case PresetDracula:
		return applyContrast(Theme{
			AddedBg:      lipgloss.Color("#244443"),
			AddedFg:      lipgloss.Color("#50FA7B"),
			RemovedBg:    lipgloss.Color("#402036"),
			RemovedFg:    lipgloss.Color("#FF79C6"),
			UnchangedFg:  lipgloss.Color("#F8F8F2"),
			LineNumberFg: lipgloss.Color("#6272A4"),
			BorderFg:     lipgloss.Color("#44475A"),
			TitleFg:      lipgloss.Color("#F8F8F2"),
			TitleBg:      lipgloss.Color("#6272A4"),
			HelpFg:       lipgloss.Color("#BD93F9"),
		}, highContrast)
	default:
		return applyContrast(DefaultTheme(), highContrast)
	}
}

// DefaultSpacing returns the default layout spacing configuration.
func DefaultSpacing() SpacingOptions {
	return SpacingOptions{LinePadding: 0, LineSpacing: 0, LineNumberWidth: 6}
}

// DefaultKeybindings returns the built-in keybinding map.
func DefaultKeybindings() Keybindings {
	return Keybindings{
		"quit":                {"ctrl+c", "q"},
		"toggle_help":         {"?", "h"},
		"toggle_stats":        {"s"},
		"toggle_status":       {"S"},
		"toggle_branches":     {"B"},
		"toggle_history":      {"H"},
		"toggle_palette":      {"p"},
		"toggle_settings":     {","},
		"toggle_side_by_side": {"v"},
		"toggle_syntax":       {"c"},
		"toggle_wrap":         {"w"},
		"toggle_blame":        {"b"},
		"toggle_line_numbers": {"ctrl+n"},
		"minimap_narrow":      {"<"},
		"minimap_widen":       {">"},
		"next_change":         {"n"},
		"prev_change":         {"N"},
		"scroll_down":         {"j", "down"},
		"scroll_up":           {"k", "up"},
		"page_down":           {"d"},
		"page_up":             {"u"},
		"go_top":              {"g"},
		"go_bottom":           {"G"},
		"go_line":             {"L"},
		"prev_branch":         {"["},
		"next_branch":         {"]"},
	}
}

// MergeKeybindings overlays user overrides onto defaults.
func MergeKeybindings(overrides Keybindings) Keybindings {
	defaults := DefaultKeybindings()
	for action, keys := range overrides {
		if len(keys) == 0 {
			continue
		}
		defaults[action] = keys
	}
	return defaults
}

func applyContrast(theme Theme, highContrast bool) Theme {
	if !highContrast {
		return theme
	}

	return Theme{
		AddedBg:      lipgloss.Color(adjustBrightness(string(theme.AddedBg), 0.15)),
		AddedFg:      lipgloss.Color(adjustBrightness(string(theme.AddedFg), 0.25)),
		RemovedBg:    lipgloss.Color(adjustBrightness(string(theme.RemovedBg), 0.15)),
		RemovedFg:    lipgloss.Color(adjustBrightness(string(theme.RemovedFg), 0.25)),
		UnchangedFg:  lipgloss.Color(adjustBrightness(string(theme.UnchangedFg), 0.2)),
		LineNumberFg: lipgloss.Color(adjustBrightness(string(theme.LineNumberFg), 0.2)),
		BorderFg:     lipgloss.Color(adjustBrightness(string(theme.BorderFg), 0.2)),
		TitleFg:      lipgloss.Color(adjustBrightness(string(theme.TitleFg), 0.2)),
		TitleBg:      lipgloss.Color(adjustBrightness(string(theme.TitleBg), 0.2)),
		HelpFg:       lipgloss.Color(adjustBrightness(string(theme.HelpFg), 0.2)),
	}
}

func adjustBrightness(hex string, factor float64) string {
	if len(hex) != 7 || hex[0] != '#' {
		return hex
	}

	var r, g, b int
	_, err := fmt.Sscanf(hex, "#%02x%02x%02x", &r, &g, &b)
	if err != nil {
		return hex
	}

	boost := func(value int) int {
		adjusted := float64(value) * (1 + factor)
		if adjusted > 255 {
			adjusted = 255
		}
		return int(adjusted)
	}

	return fmt.Sprintf("#%02x%02x%02x", boost(r), boost(g), boost(b))
}
