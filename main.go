package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/cj3636/gdiff/internal/config"
	"github.com/cj3636/gdiff/internal/diff"
	"github.com/cj3636/gdiff/internal/export"
	"github.com/cj3636/gdiff/internal/tui"
	flag "github.com/spf13/pflag"
)

var (
	showVersion      bool
	noLineNumber     bool
	ignoreWhitespace bool
	tabSize          int
	help             bool
	ref1             string
	ref2             string
	showBlame        bool
	exportFormat     string
	exportFile       string
	exportCopy       bool
)

func init() {
	flag.BoolVarP(&showVersion, "version", "v", false, "Show version information")
	flag.BoolVarP(&noLineNumber, "no-line-numbers", "n", false, "Hide line numbers")
	flag.BoolVarP(&ignoreWhitespace, "ignore-whitespace", "w", false, "Ignore whitespace changes")
	flag.IntVarP(&tabSize, "tab-size", "t", 4, "Set tab size")
	flag.StringVar(&ref1, "ref1", "", "Git reference for the left side (defaults to HEAD if ref2 is set)")
	flag.StringVar(&ref2, "ref2", "", "Git reference for the right side (defaults to working tree)")
	flag.BoolVar(&showBlame, "blame", false, "Show git blame information when available")
	flag.StringVar(&exportFormat, "export-format", "", "Export diff as html, markdown, or ansi without launching the TUI")
	flag.StringVar(&exportFile, "export-file", "", "Write exported diff to the provided file path")
	flag.BoolVar(&exportCopy, "export-copy", false, "Copy the exported diff to your clipboard")
	flag.BoolVarP(&help, "help", "h", false, "Show help information")
	flag.Usage = usage
}

func usage() {
	fmt.Println("gdiff - A beautiful terminal diff viewer built with Charm libraries")
	fmt.Println("")
	fmt.Println("Usage:")
	fmt.Println("  gdiff [options] <file1> <file2>")
	fmt.Println("  gdiff --ref1 <refA> --ref2 <refB> <tracked file>")
	fmt.Println("")
	fmt.Println("Options:")
	flag.PrintDefaults()
	fmt.Println("")
	fmt.Println("Examples:")
	fmt.Println("  gdiff old.txt new.txt")
	fmt.Println("  gdiff -n old.json new.json          # Hide line numbers")
	fmt.Println("  gdiff -t 2 config1.yaml config2.yaml # Use 2-space tabs")
	fmt.Println("  gdiff --export-format html --export-file diff.html fileA fileB # Export without TUI")
	fmt.Println("")
	fmt.Println("Keyboard shortcuts:")
	fmt.Println("  j/↓    Scroll down")
	fmt.Println("  k/↑    Scroll up")
	fmt.Println("  d      Scroll half page down")
	fmt.Println("  u      Scroll half page up")
	fmt.Println("  g      Go to top")
	fmt.Println("  G      Go to bottom")
	fmt.Println("  v      Toggle side-by-side view")
	fmt.Println("  c      Toggle syntax highlighting")
	fmt.Println("  s      Toggle statistics panel")
	fmt.Println("  b      Toggle blame overlay")
	fmt.Println("  S      Show git status")
	fmt.Println("  B      Open branch switcher (cycle with [ and ])")
	fmt.Println("  H      View recent commit history")
	fmt.Println("  ?/h    Toggle help panel")
	fmt.Println("  q      Quit")
}

func parseExportFormat(raw string) (export.Format, error) {
	switch strings.ToLower(raw) {
	case "", string(export.FormatMarkdown), "md":
		return export.FormatMarkdown, nil
	case string(export.FormatHTML), "htm":
		return export.FormatHTML, nil
	case string(export.FormatANSI), "text", "ansi":
		return export.FormatANSI, nil
	default:
		return "", fmt.Errorf("unsupported export format: %s", raw)
	}
}

func buildExportTitle(result *diff.DiffResult) string {
	if result == nil {
		return ""
	}
	return fmt.Sprintf("%s ↔ %s", filepath.Base(result.File1Name), filepath.Base(result.File2Name))
}

func loadGitDiff(engine *diff.Engine, target, leftRef, rightRef string, includeBlame bool) (tui.GitContext, *diff.DiffResult, error) {
	repoRoot, err := findRepoRoot(target)
	if err != nil {
		// Degrade gracefully if not a repository
		return tui.GitContext{}, nil, fmt.Errorf("git repository not detected: %w", err)
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return tui.GitContext{}, nil, err
	}

	relPath, err := filepath.Rel(repoRoot, absTarget)
	if err != nil {
		return tui.GitContext{}, nil, err
	}

	if leftRef == "" && rightRef != "" {
		leftRef = "HEAD"
	}
	if rightRef == "" {
		rightRef = "WORKTREE"
	}

	lines1, err := readLinesFromGit(repoRoot, relPath, leftRef)
	if err != nil {
		return tui.GitContext{}, nil, err
	}
	lines2, err := readLinesFromGit(repoRoot, relPath, rightRef)
	if err != nil {
		return tui.GitContext{}, nil, err
	}

	leftLabel := fmt.Sprintf("%s:%s", leftRef, relPath)
	rightLabel := fmt.Sprintf("%s:%s", rightRef, relPath)

	diffResult := engine.DiffLines(lines1, lines2, leftLabel, rightLabel)

	gitCtx := tui.GitContext{
		RepoRoot: repoRoot,
		FilePath: relPath,
		Ref1:     leftRef,
		Ref2:     rightRef,
		Enabled:  true,
	}

	gitCtx.Status, _ = gitCommandLines(repoRoot, "status", "--short")
	gitCtx.Branches, _ = gitCommandLines(repoRoot, "branch", "--format", "%(refname:short)")
	gitCtx.CurrentBranch, _ = gitCurrentBranch(repoRoot)
	gitCtx.CommitHistory, _ = gitCommandLines(repoRoot, "log", "--oneline", "-n", "20")

	if includeBlame {
		gitCtx.Blame, _ = gitBlame(repoRoot, relPath, rightRef)
		gitCtx.ShowBlame = true
	}

	return gitCtx, diffResult, nil
}

func findRepoRoot(path string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	cmd.Dir = filepath.Dir(path)
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func readLinesFromGit(repoRoot, relPath, ref string) ([]string, error) {
	if ref == "" || ref == "WORKTREE" {
		fullPath := filepath.Join(repoRoot, relPath)
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}
		return strings.Split(strings.TrimSuffix(string(data), "\n"), "\n"), nil
	}

	cmd := exec.Command("git", "-C", repoRoot, "show", fmt.Sprintf("%s:%s", ref, relPath))
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

func gitCommandLines(repoRoot string, args ...string) ([]string, error) {
	cmd := exec.Command("git", append([]string{"-C", repoRoot}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	text := strings.TrimSpace(string(out))
	if text == "" {
		return []string{}, nil
	}
	return strings.Split(text, "\n"), nil
}

func gitCurrentBranch(repoRoot string) (string, error) {
	branches, err := gitCommandLines(repoRoot, "branch", "--show-current")
	if err != nil {
		return "", err
	}
	if len(branches) == 0 {
		return "", nil
	}
	return branches[0], nil
}

func gitBlame(repoRoot, relPath, ref string) (map[int]string, error) {
	blame := make(map[int]string)

	target := relPath
	if ref != "" && ref != "WORKTREE" {
		target = fmt.Sprintf("%s:%s", ref, relPath)
	}

	args := []string{"-C", repoRoot, "blame", "-l", target}
	cmd := exec.Command("git", args...)
	out, err := cmd.Output()
	if err != nil {
		return blame, err
	}

	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for idx, line := range lines {
		blame[idx+1] = strings.TrimSpace(line)
	}

	return blame, nil
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
	engine := diff.NewEngine()

	gitDiffMode := ref1 != "" || ref2 != ""

	var (
		diffResult *diff.DiffResult
		err        error
		gitCtx     tui.GitContext
	)

	if gitDiffMode {
		if len(args) < 1 {
			usage()
			os.Exit(1)
		}

		target := args[0]
		gitCtx, diffResult, err = loadGitDiff(engine, target, ref1, ref2, showBlame)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error preparing git diff: %v\n", err)
			os.Exit(1)
		}
	} else {
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

		diffResult, err = engine.DiffFiles(file1, file2)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error computing diff: %v\n", err)
			os.Exit(1)
		}
	}

	// Initialize configuration
	cfg := config.DefaultConfig()
	cfg.ShowLineNo = !noLineNumber
	cfg.TabSize = tabSize
	cfg.IgnoreWhitespace = ignoreWhitespace

	if exportFormat != "" || exportFile != "" || exportCopy {
		format, err := parseExportFormat(exportFormat)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if format == "" {
			format = export.FormatMarkdown
		}

		rendered, err := export.Render(diffResult, format, export.Options{
			Title:           buildExportTitle(diffResult),
			ShowLineNumbers: cfg.ShowLineNo,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting diff: %v\n", err)
			os.Exit(1)
		}

		if exportFile != "" {
			if err := os.WriteFile(exportFile, []byte(rendered), 0o644); err != nil {
				fmt.Fprintf(os.Stderr, "Error writing export: %v\n", err)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stdout, "Diff saved to %s\n", exportFile)
		}

		if exportCopy {
			if err := export.CopyToClipboard(rendered, os.Stdout); err != nil {
				fmt.Fprintf(os.Stderr, "Error copying diff to clipboard: %v\n", err)
				os.Exit(1)
			}
			fmt.Println("Diff copied to clipboard.")
		}

		if exportFile == "" && !exportCopy {
			fmt.Println(rendered)
		}
		os.Exit(0)
	}

	// If no changes, just report and exit
	if !diffResult.HasChanges() {
		fmt.Println("Files are identical - no differences found.")
		os.Exit(0)
	}

	// Create and run the TUI
	model := tui.NewModel(diffResult, cfg, engine, gitCtx)
	p := tea.NewProgram(model, tea.WithAltScreen(), tea.WithMouseCellMotion())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running TUI: %v\n", err)
		os.Exit(1)
	}
}
