package diff

import (
	"bufio"
	"fmt"
	"os"

	"github.com/pmezard/go-difflib/difflib"
)

// DiffLine represents a single line in the diff
type DiffLine struct {
	Type    LineType
	Content string
	LineNo1 int // Line number in file 1 (0 if not applicable)
	LineNo2 int // Line number in file 2 (0 if not applicable)
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
type Engine struct{}

// NewEngine creates a new diff engine
func NewEngine() *Engine {
	return &Engine{}
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

	opcodes, err := generateOpCodes(lines1, lines2)
	if err != nil {
		// Fall back to a simpler diff strategy when the advanced matcher fails
		result.Lines = simpleDiff(lines1, lines2)
		return result
	}

	var diffLines []DiffLine
	lineNo1, lineNo2 := 1, 1

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
					Type:    Removed,
					Content: lines1[i],
					LineNo1: lineNo1,
					LineNo2: 0,
				})
				lineNo1++
			}
		case 'i': // insert
			for j := j1; j < j2; j++ {
				diffLines = append(diffLines, DiffLine{
					Type:    Added,
					Content: lines2[j],
					LineNo1: 0,
					LineNo2: lineNo2,
				})
				lineNo2++
			}
		case 'r': // replace
			// Mark as changed - show both removed and added
			maxLen := max(i2-i1, j2-j1)
			for k := 0; k < maxLen; k++ {
				if k < i2-i1 {
					diffLines = append(diffLines, DiffLine{
						Type:    Removed,
						Content: lines1[i1+k],
						LineNo1: lineNo1,
						LineNo2: 0,
					})
					lineNo1++
				}
				if k < j2-j1 {
					diffLines = append(diffLines, DiffLine{
						Type:    Added,
						Content: lines2[j1+k],
						LineNo1: 0,
						LineNo2: lineNo2,
					})
					lineNo2++
				}
			}
		}
	}

	result.Lines = diffLines
	return result
}

func generateOpCodes(lines1, lines2 []string) (opcodes []difflib.OpCode, err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("advanced diff failed: %v", r)
		}
	}()

	matcher := difflib.NewMatcher(lines1, lines2)
	return matcher.GetOpCodes(), nil
}

func simpleDiff(lines1, lines2 []string) []DiffLine {
	var diffLines []DiffLine
	lineNo1, lineNo2 := 1, 1
	maxLen := max(len(lines1), len(lines2))

	for i := 0; i < maxLen; i++ {
		has1 := i < len(lines1)
		has2 := i < len(lines2)

		switch {
		case has1 && has2 && lines1[i] == lines2[i]:
			diffLines = append(diffLines, DiffLine{
				Type:    Equal,
				Content: lines1[i],
				LineNo1: lineNo1,
				LineNo2: lineNo2,
			})
			lineNo1++
			lineNo2++
		case has1 && has2:
			diffLines = append(diffLines, DiffLine{
				Type:    Removed,
				Content: lines1[i],
				LineNo1: lineNo1,
			})
			diffLines = append(diffLines, DiffLine{
				Type:    Added,
				Content: lines2[i],
				LineNo2: lineNo2,
			})
			lineNo1++
			lineNo2++
		case has1:
			diffLines = append(diffLines, DiffLine{
				Type:    Removed,
				Content: lines1[i],
				LineNo1: lineNo1,
			})
			lineNo1++
		case has2:
			diffLines = append(diffLines, DiffLine{
				Type:    Added,
				Content: lines2[i],
				LineNo2: lineNo2,
			})
			lineNo2++
		}
	}

	return diffLines
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
