# gdiff Examples

This document provides practical examples of using gdiff.

## Basic Usage

### Compare Two Text Files

```bash
gdiff old.txt new.txt
```

This opens an interactive TUI showing the differences between the two files.

### Compare Configuration Files

```bash
gdiff config/production.yaml config/staging.yaml
```

### Compare Code Files

```bash
gdiff src/main.go src/main.go.backup
```

## Command-Line Options

### Hide Line Numbers

Useful for cleaner output or when line numbers aren't important:

```bash
gdiff -n file1.txt file2.txt
gdiff --no-line-numbers old.json new.json
```

### Ignore Whitespace Changes

Helpful when comparing code with different formatting:

```bash
gdiff -w original.py refactored.py
gdiff --ignore-whitespace code1.js code2.js
```

### Custom Tab Size

Set tab size to match your project's style:

```bash
gdiff -t 2 component1.jsx component2.jsx
gdiff --tab-size 8 makefile1 makefile2
```

### Combine Options

```bash
gdiff -n -w -t 2 file1.css file2.css
```

## Interactive Controls

Once gdiff is running, use these keyboard shortcuts:

### Navigation

- `j` or `↓` - Scroll down one line
- `k` or `↑` - Scroll up one line
- `d` - Scroll down half a page
- `u` - Scroll up half a page
- `g` - Go to the top
- `G` - Go to the bottom

### Views

- `s` - Toggle statistics view
- `?` or `h` - Toggle help screen
- `q` or `Ctrl+C` - Quit

### Statistics View

Press `s` to see detailed statistics about the diff:

- Total lines
- Added lines (count and percentage)
- Removed lines (count and percentage)
- Unchanged lines (count and percentage)
- Change ratio

## Real-World Use Cases

### 1. Review Config Changes

```bash
# Compare database configs before deployment
gdiff config/db.production.json config/db.staging.json
```

### 2. Code Review

```bash
# Review changes in a feature branch
gdiff main/api.go feature/api.go
```

### 3. Documentation Updates

```bash
# Check what changed in documentation
gdiff docs/v1/README.md docs/v2/README.md
```

### 4. Compare Logs

```bash
# Compare log files to spot differences
gdiff logs/yesterday.log logs/today.log
```

### 5. Verify Backups

```bash
# Ensure backup matches original
gdiff /path/to/original.conf /path/to/backup.conf
```

## Integration Examples

### With Git

```bash
# Compare current file with HEAD
git show HEAD:path/to/file.go > /tmp/old.go
gdiff /tmp/old.go path/to/file.go

# Compare two commits
git show commit1:file.txt > /tmp/file1.txt
git show commit2:file.txt > /tmp/file2.txt
gdiff /tmp/file1.txt /tmp/file2.txt
```

### With Scripts

```bash
#!/bin/bash
# Compare all config files in two directories

for file in config/prod/*; do
    filename=$(basename "$file")
    if [ -f "config/dev/$filename" ]; then
        echo "Comparing $filename..."
        gdiff "$file" "config/dev/$filename"
    fi
done
```

## Tips and Tricks

### 1. Quick Identical File Check

```bash
gdiff file1.txt file2.txt
# If identical, exits immediately with message:
# "Files are identical - no differences found."
```

### 2. Reading Help

```bash
gdiff --help
# Shows full usage information without needing files
```

### 3. Vim-Style Navigation

Users familiar with vim will feel at home with the navigation keys:
- `j/k` for up/down
- `g/G` for top/bottom
- `d/u` for page down/up

### 4. Color Coding

- **Green background** - Added lines
- **Red background** - Removed lines
- **Gray text** - Unchanged lines

### 5. Quick Exit

Press `q` at any time to exit, or use `Ctrl+C` for immediate termination.

## Common Patterns

### Before/After Comparison

```bash
# Save original
cp myfile.txt myfile.txt.bak

# Make changes to myfile.txt

# Compare
gdiff myfile.txt.bak myfile.txt
```

### Three-Way Merge Review

```bash
# Compare current with base
gdiff base.txt current.txt

# Compare current with incoming
gdiff current.txt incoming.txt

# Compare base with incoming
gdiff base.txt incoming.txt
```

## Output Examples

### Text Files

```
gdiff: old.txt ↔ new.txt
     1     1   Hello, World!
     2       - This is a test file.
           2 + This is a modified test file.
     3     3   It contains several lines.
```

### Configuration Files

```
gdiff: config_old.json ↔ config_new.json
     3       -   "version": "1.0.0",
           3 +   "version": "2.0.0",
     5       -     "host": "localhost",
           5 +     "host": "0.0.0.0",
```

### Statistics View

```
Diff Statistics
═══════════════

File 1: config_old.json
File 2: config_new.json

Total lines:     38
Added lines:     17 (44.7%)
Removed lines:   10 (26.3%)
Unchanged lines: 11 (28.9%)

Changes:         27
Change ratio:    71.1%
```

## Troubleshooting

### File Not Found

```bash
$ gdiff nonexistent.txt file.txt
Error: file 'nonexistent.txt' does not exist
```

### Invalid Options

```bash
$ gdiff
Usage: gdiff [options] <file1> <file2>
```

### Large Files

For very large files, scrolling might be slower. Use page navigation (`d`, `u`, `g`, `G`) for faster movement.

## Future Features

Features planned for future releases:

- Structural/syntax-aware diffing
- Language-specific syntax highlighting
- Git repository integration
- Multiple file comparison
- Custom theme configuration
- Search within diff
- Jump to next/previous change
