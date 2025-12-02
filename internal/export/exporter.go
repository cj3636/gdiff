package export

import (
	"errors"
	"fmt"
	"html"
	"path/filepath"
	"strings"

	"github.com/cj3636/gdiff/internal/diff"
)

// Format represents the desired export format.
type Format string

const (
	// FormatHTML emits an HTML document for the diff.
	FormatHTML Format = "html"
	// FormatMarkdown emits a Markdown diff code block.
	FormatMarkdown Format = "markdown"
	// FormatANSI emits an ANSI-colored string.
	FormatANSI Format = "ansi"
)

// Options control how a diff is exported.
type Options struct {
	// Title will be shown in HTML/Markdown outputs when provided.
	Title string
	// ShowLineNumbers determines whether line numbers are included.
	ShowLineNumbers bool
}

// Render returns the diff in the requested format.
func Render(result *diff.DiffResult, format Format, opts Options) (string, error) {
	if result == nil {
		return "", errors.New("diff result is nil")
	}

	switch strings.ToLower(string(format)) {
	case string(FormatHTML):
		return renderHTML(result, opts), nil
	case string(FormatMarkdown), "md":
		return renderMarkdown(result, opts), nil
	case string(FormatANSI), "text":
		return renderANSI(result, opts), nil
	default:
		return "", fmt.Errorf("unsupported export format: %s", format)
	}
}

func renderHTML(result *diff.DiffResult, opts Options) string {
	var b strings.Builder

	b.WriteString("<!DOCTYPE html>\n<html><head><meta charset=\"utf-8\">")
	b.WriteString("<style>body{background:#0f111a;color:#e5e7eb;font-family:Menlo,Consolas,monospace;}" +
		"pre{white-space:pre-wrap;word-wrap:break-word;}" +
		".added{background:#12281a;color:#8dd39e;}" +
		".removed{background:#2b1313;color:#f19999;}" +
		".unchanged{color:#cbd5e1;}" +
		".lineno{color:#9ca3af;margin-right:12px;}" +
		"h1{font-size:18px;margin-bottom:12px;}" +
		"</style></head><body>")

	title := opts.Title
	if title == "" {
		base1 := filepath.Base(result.File1Name)
		base2 := filepath.Base(result.File2Name)
		title = fmt.Sprintf("Diff: %s ↔ %s", base1, base2)
	}
	b.WriteString(fmt.Sprintf("<h1>%s</h1>\n<pre>", html.EscapeString(title)))

	for _, line := range result.Lines {
		class, symbol := classifyLine(line)
		content := html.EscapeString(line.Content)
		prefix := symbol
		if opts.ShowLineNumbers {
			prefix = fmt.Sprintf("%s %s %s", renderLineNoHTML(line.LineNo1), renderLineNoHTML(line.LineNo2), symbol)
		}
		fmt.Fprintf(&b, "<div class=\"%s\">%s%s</div>\n", class, prefix, content)
	}

	b.WriteString("</pre></body></html>")
	return b.String()
}

func renderLineNoHTML(no int) string {
	if no <= 0 {
		return "<span class=\"lineno\">&nbsp;&nbsp;&nbsp;&nbsp;&nbsp;</span>"
	}
	return fmt.Sprintf("<span class=\"lineno\">%5d</span>", no)
}

func renderMarkdown(result *diff.DiffResult, opts Options) string {
	var b strings.Builder

	if opts.Title != "" {
		b.WriteString("# ")
		b.WriteString(opts.Title)
		b.WriteString("\n\n")
	}

	b.WriteString("```diff\n")
	for _, line := range result.Lines {
		symbol := lineSymbol(line.Type)
		if opts.ShowLineNumbers {
			fmt.Fprintf(&b, "%s %5s %5s %s\n", symbol, renderLineNo(line.LineNo1), renderLineNo(line.LineNo2), line.Content)
		} else {
			fmt.Fprintf(&b, "%s %s\n", symbol, line.Content)
		}
	}
	b.WriteString("```\n")
	return b.String()
}

func renderANSI(result *diff.DiffResult, opts Options) string {
	var b strings.Builder
	title := opts.Title
	if title != "" {
		fmt.Fprintf(&b, "%s\n\n", title)
	}

	for _, line := range result.Lines {
		symbol := lineSymbol(line.Type)
		color := ansiColor(line.Type)
		reset := "\u001b[0m"
		if opts.ShowLineNumbers {
			prefix := fmt.Sprintf("%s %s %s", renderLineNoColored(line.LineNo1), renderLineNoColored(line.LineNo2), color+symbol+reset)
			fmt.Fprintf(&b, "%s %s%s%s\n", prefix, color, line.Content, reset)
		} else {
			fmt.Fprintf(&b, "%s%s %s%s\n", color, symbol, line.Content, reset)
		}
	}
	return b.String()
}

func classifyLine(line diff.DiffLine) (class, symbol string) {
	switch line.Type {
	case diff.Added:
		return "added", "+"
	case diff.Removed:
		return "removed", "-"
	default:
		return "unchanged", " "
	}
}

func lineSymbol(t diff.LineType) string {
	switch t {
	case diff.Added:
		return "+"
	case diff.Removed:
		return "-"
	default:
		return " "
	}
}

func renderLineNo(no int) string {
	if no <= 0 {
		return ""
	}
	return fmt.Sprintf("%d", no)
}

func renderLineNoColored(no int) string {
	if no <= 0 {
		return "     "
	}
	return fmt.Sprintf("\u001b[90m%5d\u001b[0m", no)
}

func ansiColor(t diff.LineType) string {
	switch t {
	case diff.Added:
		return "\u001b[32m"
	case diff.Removed:
		return "\u001b[31m"
	default:
		return "\u001b[37m"
	}
}
