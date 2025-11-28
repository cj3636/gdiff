package main

import (
	"fmt"
	"os"

	"github.com/cj3636/gdiff/internal/config"
	"github.com/cj3636/gdiff/internal/diff"
	"github.com/cj3636/gdiff/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	flag "github.com/spf13/pflag"
)

var (
	showVersion      bool
	noLineNumber     bool
	ignoreWhitespace bool
	tabSize          int
	help             bool
)

func init() {
	flag.BoolVarP(&showVersion, "version", "v", false, "Show version information")
	flag.BoolVarP(&noLineNumber, "no-line-numbers", "n", false, "Hide line numbers")
	flag.BoolVarP(&ignoreWhitespace, "ignore-whitespace", "w", false, "Ignore whitespace changes")
	flag.IntVarP(&tabSize, "tab-size", "t", 4, "Set tab size")
	flag.BoolVarP(&help, "help", "h", false, "Show help information")
	flag.Usage = usage
}

func usage() {
	fmt.Println("gdiff - A beautiful terminal diff viewer built with Charm libraries")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  gdiff [options] <file1> <file2>")
	fmt.Println("")
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  gdiff old.txt new.txt")
	fmt.Println("  gdiff -n old.json new.json          # Hide line numbers")
	fmt.Println("  gdiff -t 2 config1.yaml config2.yaml # Use 2-space tabs")
	fmt.Println("")
	fmt.Println("Keyboard shortcuts:")
	fmt.Println("  j/↓    Scroll down")
	fmt.Println("  k/↑    Scroll up")
	fmt.Println("  d      Scroll half page down")
	fmt.Println("  u      Scroll half page up")
	fmt.Println("  g      Go to top")
	fmt.Println("  G      Go to bottom")
	fmt.Println("  ?/h    Toggle help")
	fmt.Println("  q      Quit")
}

func main() {
	flag.Parse()

	if help {
		usage()
		os.Exit(0)
	}

	if showVersion {
		fmt.Println("gdiff version 0.1.0")
		fmt.Println("A beautiful terminal diff viewer built with Charm libraries")
		os.Exit(0)
	}

	args := flag.Args()
	if len(args) < 2 {
		usage()
		os.Exit(1)
	}

	file1 := args[0]
	file2 := args[1]

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
	cfg.ShowLineNo = !noLineNumber
	cfg.TabSize = tabSize
	cfg.IgnoreWhitespace = ignoreWhitespace

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
