package config

import (
	"github.com/charmbracelet/lipgloss"
)

// Config holds the application configuration
type Config struct {
	Theme      Theme
	DiffMode   DiffMode
	ShowLineNo bool
	TabSize    int
}

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
		Theme:      DefaultTheme(),
		DiffMode:   SideBySide,
		ShowLineNo: true,
		TabSize:    4,
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
