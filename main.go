package main

import (
	"fmt"
	"os"

	"github.com/cj3636/gdiff/internal/config"
	"github.com/cj3636/gdiff/internal/diff"
	"github.com/cj3636/gdiff/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: gdiff <file1> <file2>")
		fmt.Println("")
		fmt.Println("gdiff - A beautiful terminal diff viewer built with Charm libraries")
		fmt.Println("")
		fmt.Println("Examples:")
		fmt.Println("  gdiff old.txt new.txt")
		fmt.Println("  gdiff config/old.json config/new.json")
		os.Exit(1)
	}

	file1 := os.Args[1]
	file2 := os.Args[2]

	// Check if files exist
	if _, err := os.Stat(file1); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: file '%s' does not exist\n", file1)
		os.Exit(1)
	}
	if _, err := os.Stat(file2); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Error: file '%s' does not exist\n", file2)
		os.Exit(1)
	}

	// Initialize configuration
	cfg := config.DefaultConfig()

	// Create diff engine and compute diff
	engine := diff.NewEngine()
	diffResult, err := engine.DiffFiles(file1, file2)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error computing diff: %v\n", err)
		os.Exit(1)
	}

	// If no changes, just report and exit
	if !diffResult.HasChanges() {
		fmt.Println("Files are identical - no differences found.")
		os.Exit(0)
	}

	// Create and run the TUI
	model := tui.NewModel(diffResult, cfg)
	p := tea.NewProgram(model, tea.WithAltScreen())
	
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
