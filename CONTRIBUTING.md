# Contributing to gdiff

Thank you for your interest in contributing to gdiff! This document provides guidelines and instructions for contributing to the project.

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Git
- Basic familiarity with terminal applications

### Setting Up Development Environment

1. Fork the repository on GitHub
2. Clone your fork:
   ```bash
   git clone https://github.com/YOUR_USERNAME/gdiff.git
   cd gdiff
   ```
3. Add upstream remote:
   ```bash
   git remote add upstream https://github.com/cj3636/gdiff.git
   ```
4. Install dependencies:
   ```bash
   go mod download
   ```
5. Build the project:
   ```bash
   go build -o gdiff .
   ```

## Project Structure

```
gdiff/
├── main.go              # Entry point and CLI handling
├── internal/
│   ├── config/          # Configuration and theme system
│   │   └── config.go
│   ├── diff/            # Diff engine and algorithms
│   │   └── engine.go
│   └── tui/             # Terminal UI components
│       └── model.go     # Bubble Tea model
├── go.mod
├── go.sum
├── README.md
├── LICENSE
├── EXAMPLES.md
└── CONTRIBUTING.md
```

## Development Guidelines

### Code Style

- Follow standard Go conventions and idioms
- Use `gofmt` to format your code
- Write clear, descriptive variable and function names
- Add comments for exported functions and complex logic
- Keep functions focused and concise

### Architecture Principles

1. **Separation of Concerns**: Keep diff logic, UI, and configuration separate
2. **Extensibility**: Design for future enhancements
3. **Testability**: Write testable code with minimal dependencies
4. **Performance**: Consider performance for large files
5. **User Experience**: Prioritize intuitive, responsive UI

### Making Changes

1. Create a feature branch:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. Make your changes following the guidelines

3. Build and test:
   ```bash
   go build -o gdiff .
   ./gdiff test1.txt test2.txt
   ```

4. Format your code:
   ```bash
   go fmt ./...
   ```

5. Commit your changes:
   ```bash
   git add .
   git commit -m "Add feature: description"
   ```

6. Push to your fork:
   ```bash
   git push origin feature/your-feature-name
   ```

7. Create a Pull Request on GitHub

## Types of Contributions

### Bug Fixes

- Check existing issues first
- Describe the bug clearly
- Include steps to reproduce
- Provide a fix with explanation

### New Features

- Discuss in an issue first for major features
- Ensure it aligns with project goals
- Update documentation
- Add examples if applicable

### Documentation

- Fix typos and clarify explanations
- Add examples and use cases
- Improve README or other docs
- Update help text if needed

### Performance Improvements

- Profile and measure before and after
- Explain the optimization approach
- Ensure correctness is maintained

## Coding Standards

### Go Best Practices

- Use meaningful package names
- Avoid global variables
- Handle errors explicitly
- Use interfaces for flexibility
- Write table-driven tests when appropriate

### Comments

```go
// Good: Clear, concise explanation
// DiffFiles compares two files and returns the differences
func (e *Engine) DiffFiles(file1, file2 string) (*DiffResult, error) {
    // ...
}

// Bad: Obvious or redundant
// This function diffs files
func (e *Engine) DiffFiles(file1, file2 string) (*DiffResult, error) {
    // ...
}
```

### Error Handling

```go
// Good: Wrap errors with context
if err := doSomething(); err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// Bad: Silent failures
if err := doSomething(); err != nil {
    return nil
}
```

## Testing

### Running Tests

```bash
go test ./...
```

### Writing Tests

- Test exported functions
- Use table-driven tests for multiple scenarios
- Include edge cases
- Test error conditions

Example:
```go
func TestDiffLines(t *testing.T) {
    tests := []struct {
        name     string
        lines1   []string
        lines2   []string
        expected int
    }{
        {
            name:     "identical files",
            lines1:   []string{"line1", "line2"},
            lines2:   []string{"line1", "line2"},
            expected: 0,
        },
        // Add more test cases
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            // Test implementation
        })
    }
}
```

## Pull Request Process

1. **Title**: Use clear, descriptive titles
   - ✅ "Add support for unified diff view"
   - ❌ "Update code"

2. **Description**: Include:
   - What changes were made
   - Why they were made
   - How to test them
   - Any breaking changes

3. **Review**: Be responsive to feedback
   - Address reviewer comments
   - Make requested changes
   - Ask questions if unclear

4. **Merge**: Once approved, maintainers will merge

## Feature Requests

To request a new feature:

1. Check existing issues to avoid duplicates
2. Create a new issue with:
   - Clear description of the feature
   - Use case and motivation
   - Proposed implementation (if any)
   - Examples of similar features elsewhere

## Bug Reports

To report a bug:

1. Check if it's already reported
2. Create a new issue with:
   - Clear, descriptive title
   - Steps to reproduce
   - Expected behavior
   - Actual behavior
   - Environment (OS, Go version, etc.)
   - Sample files if applicable

## Community Guidelines

- Be respectful and inclusive
- Welcome newcomers
- Provide constructive feedback
- Help others when possible
- Follow the [Code of Conduct](https://www.contributor-covenant.org/version/2/0/code_of_conduct/)

## Future Roadmap

Areas we're looking to improve:

- **Syntax-aware diffing**: Parse code structure for smarter diffs
- **Language detection**: Auto-detect file types for highlighting
- **Git integration**: Browse repository diffs
- **Multi-file view**: Compare multiple files at once
- **Theme system**: User-customizable color schemes
- **Plugin architecture**: Extensibility for custom features

## Questions?

If you have questions:
- Open a discussion on GitHub
- Check existing issues and PRs
- Read the documentation

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

## Thank You!

Every contribution, no matter how small, helps make gdiff better. We appreciate your time and effort! 🎉
