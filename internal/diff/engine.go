package diff

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode/utf8"

	"github.com/pmezard/go-difflib/difflib"
)

// DiffLine represents a single line in the diff
type DiffLine struct {
	Type       LineType
	Content    string
	LineNo1    int // Line number in file 1 (0 if not applicable)
	LineNo2    int // Line number in file 2 (0 if not applicable)
	Highlights []Highlight
}

// Highlight marks a token range that changed within a line.
type Highlight struct {
	Start int // rune offset (inclusive)
	End   int // rune offset (exclusive)
}

// LineType defines the type of diff line
type LineType int

const (
	Equal LineType = iota
	Added
	Removed
)

// DiffResult contains the results of a diff operation
type DiffResult struct {
	Lines      []DiffLine
	File1Name  string
	File2Name  string
	File1Lines []string
	File2Lines []string
}

// Engine handles diff operations
type Engine struct {
	options          EngineOptions
	tokenizers       map[string]Tokenizer
	defaultTokenizer Tokenizer
	ignorePatterns   []*regexp.Regexp
}

// EngineOptions controls diff behavior.
type EngineOptions struct {
	Language         string
	IgnoreWhitespace bool
	IgnorePatterns   []string
	TokenPatterns    map[string]string
}

// Token represents a tokenized fragment of a line.
type Token struct {
	Value string
	Start int
	End   int
}

// Tokenizer splits a line into tokens.
type Tokenizer interface {
	Tokenize(line string) []Token
}

// RegexTokenizer tokenizes text using a regular expression.
type RegexTokenizer struct {
	pattern *regexp.Regexp
}

const defaultTokenPattern = `\s+|[A-Za-z_][A-Za-z0-9_]*|0x[0-9A-Fa-f]+|\d+|==|!=|<=|>=|:=|&&|\|\||[{}()[\].,;:+\-*/&|<>!=]`

// NewRegexTokenizer compiles a regex tokenizer.
func NewRegexTokenizer(pattern string) *RegexTokenizer {
	compiled := regexp.MustCompile(pattern)
	return &RegexTokenizer{pattern: compiled}
}

// Tokenize splits the line using the configured regex and records rune offsets.
func (r *RegexTokenizer) Tokenize(line string) []Token {
	matches := r.pattern.FindAllStringIndex(line, -1)
	if len(matches) == 0 {
		return []Token{{Value: line, Start: 0, End: utf8.RuneCountInString(line)}}
	}

	tokens := make([]Token, 0, len(matches))
	for _, match := range matches {
		startByte, endByte := match[0], match[1]
		start := utf8.RuneCountInString(line[:startByte])
		end := start + utf8.RuneCountInString(line[startByte:endByte])
		tokens = append(tokens, Token{Value: line[startByte:endByte], Start: start, End: end})
	}
	return tokens
}

// NewEngine creates a new diff engine
func NewEngine(options EngineOptions) *Engine {
	engine := &Engine{options: options}
	engine.defaultTokenizer = NewRegexTokenizer(defaultTokenPattern)
	engine.tokenizers = engine.buildTokenizers(options.TokenPatterns)
	engine.ignorePatterns = compileIgnorePatterns(options.IgnorePatterns)
	return engine
}

// DiffFiles compares two files and returns the differences
func (e *Engine) DiffFiles(file1, file2 string) (*DiffResult, error) {
	lines1, err := readFileLines(file1)
	if err != nil {
		return nil, err
	}

	lines2, err := readFileLines(file2)
	if err != nil {
		return nil, err
	}

	return e.DiffLines(lines1, lines2, file1, file2), nil
}

// DiffLines compares two slices of lines
func (e *Engine) DiffLines(lines1, lines2 []string, file1Name, file2Name string) *DiffResult {
	result := &DiffResult{
		File1Name:  file1Name,
		File2Name:  file2Name,
		File1Lines: lines1,
		File2Lines: lines2,
	}

	normalized1 := e.normalizeLines(lines1)
	normalized2 := e.normalizeLines(lines2)

	// Get the opcodes for a more structured diff
	matcher := difflib.NewMatcher(normalized1, normalized2)
	opcodes := matcher.GetOpCodes()

	var diffLines []DiffLine
	lineNo1, lineNo2 := 1, 1

	tokenizer := e.selectTokenizer(file1Name, file2Name)

	for _, opcode := range opcodes {
		tag := opcode.Tag
		i1, i2, j1, j2 := opcode.I1, opcode.I2, opcode.J1, opcode.J2

		switch tag {
		case 'e': // equal
			for i := i1; i < i2; i++ {
				diffLines = append(diffLines, DiffLine{
					Type:    Equal,
					Content: lines1[i],
					LineNo1: lineNo1,
					LineNo2: lineNo2,
				})
				lineNo1++
				lineNo2++
			}
		case 'd': // delete
			for i := i1; i < i2; i++ {
				diffLines = append(diffLines, DiffLine{
					Type:       Removed,
					Content:    lines1[i],
					LineNo1:    lineNo1,
					LineNo2:    0,
					Highlights: []Highlight{{Start: 0, End: utf8.RuneCountInString(lines1[i])}},
				})
				lineNo1++
			}
		case 'i': // insert
			for j := j1; j < j2; j++ {
				diffLines = append(diffLines, DiffLine{
					Type:       Added,
					Content:    lines2[j],
					LineNo1:    0,
					LineNo2:    lineNo2,
					Highlights: []Highlight{{Start: 0, End: utf8.RuneCountInString(lines2[j])}},
				})
				lineNo2++
			}
		case 'r': // replace
			// Mark as changed - show both removed and added
			maxLen := max(i2-i1, j2-j1)
			for k := 0; k < maxLen; k++ {
				var leftHighlights, rightHighlights []Highlight
				if k < i2-i1 && k < j2-j1 {
					leftHighlights, rightHighlights = e.tokenHighlights(lines1[i1+k], lines2[j1+k], tokenizer)
				} else if k < i2-i1 {
					leftHighlights = []Highlight{{Start: 0, End: utf8.RuneCountInString(lines1[i1+k])}}
				} else if k < j2-j1 {
					rightHighlights = []Highlight{{Start: 0, End: utf8.RuneCountInString(lines2[j1+k])}}
				}

				if k < i2-i1 {
					diffLines = append(diffLines, DiffLine{
						Type:       Removed,
						Content:    lines1[i1+k],
						LineNo1:    lineNo1,
						LineNo2:    0,
						Highlights: leftHighlights,
					})
					lineNo1++
				}
				if k < j2-j1 {
					diffLines = append(diffLines, DiffLine{
						Type:       Added,
						Content:    lines2[j1+k],
						LineNo1:    0,
						LineNo2:    lineNo2,
						Highlights: rightHighlights,
					})
					lineNo2++
				}
			}
		}
	}

	result.Lines = diffLines
	return result
}

func (e *Engine) tokenHighlights(left, right string, tokenizer Tokenizer) ([]Highlight, []Highlight) {
	leftTokens := tokenizer.Tokenize(left)
	rightTokens := tokenizer.Tokenize(right)

	leftValues := make([]string, len(leftTokens))
	for i, t := range leftTokens {
		leftValues[i] = t.Value
	}

	rightValues := make([]string, len(rightTokens))
	for i, t := range rightTokens {
		rightValues[i] = t.Value
	}

	matcher := difflib.NewMatcher(leftValues, rightValues)
	opcodes := matcher.GetOpCodes()

	var leftHighlights []Highlight
	var rightHighlights []Highlight

	for _, opcode := range opcodes {
		switch opcode.Tag {
		case 'r', 'd':
			leftHighlights = append(leftHighlights, tokenRangeToHighlight(leftTokens, opcode.I1, opcode.I2))
		}
		switch opcode.Tag {
		case 'r', 'i':
			rightHighlights = append(rightHighlights, tokenRangeToHighlight(rightTokens, opcode.J1, opcode.J2))
		}
	}

	return mergeHighlights(leftHighlights), mergeHighlights(rightHighlights)
}

func tokenRangeToHighlight(tokens []Token, start, end int) Highlight {
	if len(tokens) == 0 || start >= len(tokens) || start == end {
		return Highlight{Start: 0, End: 0}
	}
	if end > len(tokens) {
		end = len(tokens)
	}
	return Highlight{Start: tokens[start].Start, End: tokens[end-1].End}
}

func mergeHighlights(highlights []Highlight) []Highlight {
	if len(highlights) == 0 {
		return highlights
	}

	// Filter zero-length entries
	filtered := highlights[:0]
	for _, h := range highlights {
		if h.End > h.Start {
			filtered = append(filtered, h)
		}
	}

	if len(filtered) == 0 {
		return filtered
	}

	// Sort by start
	for i := 0; i < len(filtered)-1; i++ {
		for j := i + 1; j < len(filtered); j++ {
			if filtered[j].Start < filtered[i].Start {
				filtered[i], filtered[j] = filtered[j], filtered[i]
			}
		}
	}

	merged := []Highlight{filtered[0]}
	for _, h := range filtered[1:] {
		last := &merged[len(merged)-1]
		if h.Start <= last.End {
			if h.End > last.End {
				last.End = h.End
			}
			continue
		}
		merged = append(merged, h)
	}

	return merged
}

func (e *Engine) normalizeLines(lines []string) []string {
	normalized := make([]string, len(lines))
	for i, line := range lines {
		normalized[i] = e.normalizeLine(line)
	}
	return normalized
}

func (e *Engine) normalizeLine(line string) string {
	normalized := line
	for _, re := range e.ignorePatterns {
		normalized = re.ReplaceAllString(normalized, "")
	}
	if e.options.IgnoreWhitespace {
		normalized = strings.Join(strings.Fields(normalized), " ")
	}
	return normalized
}

func (e *Engine) buildTokenizers(patterns map[string]string) map[string]Tokenizer {
	result := map[string]Tokenizer{}

	defaultMap := map[string]string{
		".go":   defaultTokenPattern,
		".js":   defaultTokenPattern,
		".ts":   defaultTokenPattern,
		".jsx":  defaultTokenPattern,
		".tsx":  defaultTokenPattern,
		".py":   defaultTokenPattern,
		".rb":   defaultTokenPattern,
		".rs":   defaultTokenPattern,
		".java": defaultTokenPattern,
		".c":    defaultTokenPattern,
		".h":    defaultTokenPattern,
		".cpp":  defaultTokenPattern,
		".css":  defaultTokenPattern,
		".html": defaultTokenPattern,
		".md":   defaultTokenPattern,
	}

	for ext, pattern := range defaultMap {
		result[ext] = NewRegexTokenizer(pattern)
	}

	for ext, pattern := range patterns {
		result[ext] = NewRegexTokenizer(pattern)
	}

	return result
}

func (e *Engine) selectTokenizer(file1Name, file2Name string) Tokenizer {
	if e.options.Language != "" {
		if t, ok := e.tokenizers[e.options.Language]; ok {
			return t
		}
	}

	for _, name := range []string{file1Name, file2Name} {
		ext := filepath.Ext(name)
		if tokenizer, ok := e.tokenizers[ext]; ok {
			return tokenizer
		}
	}

	return e.defaultTokenizer
}

func compileIgnorePatterns(patterns []string) []*regexp.Regexp {
	var compiled []*regexp.Regexp
	for _, pattern := range patterns {
		re, err := regexp.Compile(pattern)
		if err == nil {
			compiled = append(compiled, re)
		}
	}
	return compiled
}

// readFileLines reads a file and returns its lines
func readFileLines(filename string) ([]string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return lines, nil
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// GetStats returns statistics about the diff
func (r *DiffResult) GetStats() (added, removed, unchanged int) {
	for _, line := range r.Lines {
		switch line.Type {
		case Added:
			added++
		case Removed:
			removed++
		case Equal:
			unchanged++
		}
	}
	return
}

// HasChanges returns true if there are any differences
func (r *DiffResult) HasChanges() bool {
	for _, line := range r.Lines {
		if line.Type != Equal {
			return true
		}
	}
	return false
}
