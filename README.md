# gdiff

**Glam Diff** - A beautiful terminal diff viewer built with Charm libraries.

![Go Version](https://img.shields.io/badge/Go-%3E%3D%201.21-blue)
![License](https://img.shields.io/badge/License-MIT-green)

## Features

- 🎨 **Beautiful TUI** - Built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and [Lip Gloss](https://github.com/charmbracelet/lipgloss)
- 🔍 **Smart Diff** - Structural diffing with intelligent alignment
- 🎯 **Side-by-Side View** - Clear visual representation of changes
- ⌨️ **Vim-style Navigation** - Intuitive keyboard shortcuts
- 🎨 **Syntax Highlighting** - Color-coded additions and deletions
- ⚙️ **Configurable** - Extensible architecture for customization
- 📊 **Statistics** - Track additions, deletions, and unchanged lines

## Installation

### From Source

```bash
git clone https://github.com/cj3636/gdiff.git
cd gdiff
make build
```

Or manually:

```bash
go build -o gdiff .
```

### Using Go Install

```bash
go install github.com/cj3636/gdiff@latest
```

### Using Make Targets

```bash
make help     # Show all available commands
make build    # Build the application
make install  # Install to $GOPATH/bin
make clean    # Remove build artifacts
make fmt      # Format code
make test     # Run tests
```

## Usage

### Basic Usage

```bash
gdiff file1.txt file2.txt
```

### Command-Line Options

```bash
gdiff [options] <file1> <file2>

Options:
  -h, --help              Show help information
  -v, --version           Show version information
  -n, --no-line-numbers   Hide line numbers
  -t, --tab-size int      Set tab size (default 4)
```

### Examples

```bash
# Compare two text files
gdiff old.txt new.txt

# Compare without line numbers
gdiff -n config1.json config2.json

# Compare with custom tab size
gdiff -t 2 code1.go code2.go
```

## Keyboard Shortcuts

| Key        | Action                  |
|------------|------------------------|
| `j` / `↓`  | Scroll down one line   |
| `k` / `↑`  | Scroll up one line     |
| `d`        | Scroll half page down  |
| `u`        | Scroll half page up    |
| `g`        | Go to top              |
| `G`        | Go to bottom           |
| `?` / `h`  | Toggle help            |
| `q` / `^C` | Quit                   |

## Architecture

The project is organized for extensibility and maintainability:

```
gdiff/
├── main.go              # Entry point with CLI handling
├── internal/
│   ├── config/          # Configuration system
│   ├── diff/            # Diff engine with multiple algorithms
│   └── tui/             # Terminal UI components
├── go.mod
└── README.md
```

### Key Components

- **Diff Engine** (`internal/diff`): Handles file comparison using intelligent algorithms
- **TUI Layer** (`internal/tui`): Bubble Tea model for interactive display
- **Config System** (`internal/config`): Flexible configuration with themes

## Comparison with Difftastic

`gdiff` is inspired by [difftastic](https://difftastic.wilfred.me.uk/) and aims to provide:

- ✅ Beautiful terminal UI with Charm libraries
- ✅ Easy-to-use interface with intuitive navigation
- ✅ Extensible architecture for future enhancements
- ✅ Side-by-side diff view
- ✅ Color-coded changes
- 🚧 Structural/syntax-aware diffing (planned)
- 🚧 Language detection (planned)
- 🚧 Git integration (planned)

## Future Enhancements

This is the foundation for a full TUI for managing and viewing local git repositories. Planned features include:

- [ ] Syntax-aware structural diffing
- [ ] Language detection and specific highlighting
- [ ] Git repository integration
- [ ] Interactive staging/unstaging
- [ ] Multiple file comparison
- [ ] Custom theme configuration
- [ ] Plugin system for extensibility

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

## License

MIT License - See LICENSE file for details.

## Acknowledgments

- Built with [Charm](https://charm.sh/) libraries
- Inspired by [difftastic](https://difftastic.wilfred.me.uk/)
- Uses [go-difflib](https://github.com/pmezard/go-difflib) for diff algorithms 
