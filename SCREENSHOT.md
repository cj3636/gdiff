# gdiff Screenshots and Output Examples

This document shows examples of what gdiff looks like when running.

## Basic Diff View

When you run `gdiff old.txt new.txt`, you'll see:

```
┌─────────────────────────────────────────────────────────────────┐
│ gdiff: old.txt ↔ new.txt                                        │
├─────────────────────────────────────────────────────────────────┤
│      1     1   Hello, World!                                    │
│      2       - This is a test file.                             │
│            2 + This is a modified test file.                    │
│      3     3   It contains several lines.                       │
│      4       - Some of these lines will be changed.             │
│            4 + Some of these lines have been changed.           │
│      5     5   This line stays the same.                        │
│            6 + A new line was added here.                       │
│      6     7   Another line here.                               │
│      7     8   And one more line.                               │
│            9 + Plus an extra line at the end.                   │
├─────────────────────────────────────────────────────────────────┤
│ Lines: +4 -2 =6 | Scroll: 1/12 | s:stats ?:help q:quit         │
└─────────────────────────────────────────────────────────────────┘
```

**Color Coding:**
- Lines with `-` prefix: Red background (deletions)
- Lines with `+` prefix: Green background (additions)  
- Lines with ` ` prefix: Gray text (unchanged)
- Line numbers: Gray text

## Configuration File Diff

When comparing JSON files with `gdiff config1.json config2.json`:

```
┌─────────────────────────────────────────────────────────────────┐
│ gdiff: config_old.json ↔ config_new.json                        │
├─────────────────────────────────────────────────────────────────┤
│      1     1   {                                                │
│      2     2     "app_name": "MyApp",                           │
│      3       -   "version": "1.0.0",                            │
│            3 +   "version": "2.0.0",                            │
│      4     4     "server": {                                    │
│      5       -     "host": "localhost",                         │
│            5 +     "host": "0.0.0.0",                           │
│      6       -     "port": 8080,                                │
│            6 +     "port": 9000,                                │
│      7       -     "ssl": false                                 │
│            7 +     "ssl": true,                                 │
│            8 +     "ssl_cert": "/path/to/cert.pem"              │
│      8     9     },                                             │
│     ...                                                         │
├─────────────────────────────────────────────────────────────────┤
│ Lines: +17 -10 =11 | Scroll: 1/38 | s:stats ?:help q:quit      │
└─────────────────────────────────────────────────────────────────┘
```

## Statistics View (Press 's')

```
┌─────────────────────────────────────────────────────────────────┐
│ gdiff: config_old.json ↔ config_new.json                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ Diff Statistics                                                 │
│ ═══════════════                                                 │
│                                                                 │
│ File 1: config_old.json                                         │
│ File 2: config_new.json                                         │
│                                                                 │
│ Total lines:     38                                             │
│ Added lines:     17 (44.7%)                                     │
│ Removed lines:   10 (26.3%)                                     │
│ Unchanged lines: 11 (28.9%)                                     │
│                                                                 │
│ Changes:         27                                             │
│ Change ratio:    71.1%                                          │
│                                                                 │
│ Press 's' to return to diff view                                │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ Lines: +17 -10 =11 | Scroll: 1/38 | s:stats ?:help q:quit      │
└─────────────────────────────────────────────────────────────────┘
```

## Help Screen (Press '?')

```
┌─────────────────────────────────────────────────────────────────┐
│ gdiff: old.txt ↔ new.txt                                        │
├─────────────────────────────────────────────────────────────────┤
│                                                                 │
│ Keyboard Shortcuts:                                             │
│                                                                 │
│   j, ↓      Scroll down one line                                │
│   k, ↑      Scroll up one line                                  │
│   d         Scroll down half page                               │
│   u         Scroll up half page                                 │
│   g         Go to top                                           │
│   G         Go to bottom                                        │
│   s         Toggle statistics                                   │
│   h, ?      Toggle help                                         │
│   q, Ctrl+C Quit                                                │
│                                                                 │
├─────────────────────────────────────────────────────────────────┤
│ Lines: +4 -2 =6 | Scroll: 1/12 | s:stats ?:help q:quit         │
└─────────────────────────────────────────────────────────────────┘
```

## Without Line Numbers (--no-line-numbers flag)

When run with `gdiff -n old.txt new.txt`:

```
┌─────────────────────────────────────────────────────────────────┐
│ gdiff: old.txt ↔ new.txt                                        │
├─────────────────────────────────────────────────────────────────┤
│   Hello, World!                                                 │
│ - This is a test file.                                          │
│ + This is a modified test file.                                 │
│   It contains several lines.                                    │
│ - Some of these lines will be changed.                          │
│ + Some of these lines have been changed.                        │
│   This line stays the same.                                     │
│ + A new line was added here.                                    │
│   Another line here.                                            │
│   And one more line.                                            │
│ + Plus an extra line at the end.                                │
├─────────────────────────────────────────────────────────────────┤
│ Lines: +4 -2 =6 | Scroll: 1/12 | s:stats ?:help q:quit         │
└─────────────────────────────────────────────────────────────────┘
```

## Identical Files

When files are identical:

```bash
$ gdiff file1.txt file1.txt
Files are identical - no differences found.
```

## Key Visual Features

1. **Title Bar**: Shows both filenames being compared
2. **Line Numbers**: Dual columns showing line numbers from both files
3. **Color Coding**: 
   - Green background: Added lines
   - Red background: Removed lines
   - Gray text: Unchanged lines
4. **Status Bar**: Real-time information about:
   - Number of added/removed/unchanged lines
   - Current scroll position
   - Available keyboard shortcuts
5. **Smooth Scrolling**: Navigate through large diffs easily
6. **Multiple Views**: Switch between diff, statistics, and help

## Command Line Examples

```bash
# Basic usage
$ gdiff old.txt new.txt

# Hide line numbers for cleaner output
$ gdiff -n config1.json config2.json

# Ignore whitespace changes
$ gdiff -w formatted.py unformatted.py

# Custom tab size
$ gdiff -t 2 makefile1 makefile2

# Combine options
$ gdiff -n -w -t 4 file1.go file2.go
```

## Future Enhancements

Planned visual improvements:
- Syntax-specific highlighting (different colors for keywords, strings, etc.)
- Inline word-level diffs
- Split-screen mode
- Multiple file tabs
- Custom color themes
