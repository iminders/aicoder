# Contributing to aicoder

Thank you for your interest in contributing to aicoder! This document provides guidelines and instructions for contributing.

## Development Setup

### Prerequisites

- Go 1.19 or higher
- Git
- Make (optional, but recommended)

### Clone and Build

```bash
# Clone the repository
git clone https://github.com/iminders/aicoder.git
cd aicoder

# Install dependencies
go mod download

# Build
make build

# Run tests
make test

# Run linter
make lint
```

### Project Structure

```
aicoder/
├── cmd/                    # CLI entry point
├── internal/               # Internal packages
│   ├── agent/             # Agent loop and execution
│   ├── llm/               # LLM provider implementations
│   ├── tools/             # Built-in tools
│   ├── session/           # Session management
│   ├── config/            # Configuration system
│   ├── context/           # Project context collection
│   ├── slash/             # Slash commands
│   ├── ui/                # Terminal UI
│   └── logger/            # Logging
├── pkg/                   # Public packages
│   ├── diff/              # Diff utilities
│   └── version/           # Version information
├── docs/                  # Documentation
└── scripts/               # Build and release scripts
```

## Development Workflow

### 1. Create a Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/your-bug-fix
```

### 2. Make Changes

- Write clean, idiomatic Go code
- Follow the existing code style
- Add tests for new functionality
- Update documentation as needed

### 3. Test Your Changes

```bash
# Run all tests
make test

# Run tests with coverage
go test -cover ./...

# Run specific package tests
go test ./internal/agent/

# Run linter
make lint
```

### 4. Commit Your Changes

We follow [Conventional Commits](https://www.conventionalcommits.org/):

```bash
# Feature
git commit -m "feat: add new tool for X"

# Bug fix
git commit -m "fix: resolve issue with Y"

# Documentation
git commit -m "docs: update README with Z"

# Refactoring
git commit -m "refactor: improve performance of W"

# Tests
git commit -m "test: add tests for V"

# Chore
git commit -m "chore: update dependencies"
```

### 5. Push and Create Pull Request

```bash
git push origin feature/your-feature-name
```

Then create a Pull Request on GitHub.

## Code Style Guidelines

### Go Code

- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` for formatting
- Use meaningful variable and function names
- Add comments for exported functions and types
- Keep functions small and focused
- Handle errors explicitly

### Example

```go
// Good
func ReadFile(path string) ([]byte, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, fmt.Errorf("failed to read file %s: %w", path, err)
    }
    return data, nil
}

// Bad
func read(p string) []byte {
    d, _ := os.ReadFile(p)
    return d
}
```

### Testing

- Write table-driven tests when possible
- Use descriptive test names
- Test both success and error cases
- Mock external dependencies

```go
func TestReadFile(t *testing.T) {
    tests := []struct {
        name    string
        path    string
        want    []byte
        wantErr bool
    }{
        {
            name:    "existing file",
            path:    "testdata/file.txt",
            want:    []byte("content"),
            wantErr: false,
        },
        {
            name:    "non-existent file",
            path:    "testdata/missing.txt",
            want:    nil,
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ReadFile(tt.path)
            if (err != nil) != tt.wantErr {
                t.Errorf("ReadFile() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !bytes.Equal(got, tt.want) {
                t.Errorf("ReadFile() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

## Adding New Features

### Adding a New Tool

1. Create a new file in `internal/tools/<category>/`
2. Implement the `Tool` interface
3. Register the tool in the init function
4. Add tests
5. Update documentation

Example:

```go
package filesystem

import "github.com/iminders/aicoder/internal/tools"

type MyTool struct{}

func (t *MyTool) Name() string {
    return "my_tool"
}

func (t *MyTool) Description() string {
    return "Does something useful"
}

func (t *MyTool) Parameters() interface{} {
    return struct {
        Path string `json:"path" description:"File path"`
    }{}
}

func (t *MyTool) Execute(params interface{}) (interface{}, error) {
    // Implementation
    return nil, nil
}

func init() {
    tools.Register(&MyTool{})
}
```

### Adding a New Slash Command

1. Add the command handler in `internal/slash/commands.go`
2. Update the switch statement in `Handle()`
3. Add to `AllCommands()` in `completion.go`
4. Add tests
5. Update help text

### Adding a New LLM Provider

1. Create a new package in `internal/llm/<provider>/`
2. Implement the `Provider` interface
3. Add to the factory in `cmd/root.go`
4. Add tests
5. Update documentation

## Pull Request Guidelines

### Before Submitting

- [ ] Tests pass: `make test`
- [ ] Linter passes: `make lint`
- [ ] Code is formatted: `gofmt -s -w .`
- [ ] Documentation is updated
- [ ] CHANGELOG.md is updated (for significant changes)
- [ ] Commit messages follow Conventional Commits

### PR Description

Include:
- What changes were made
- Why the changes were made
- How to test the changes
- Any breaking changes
- Related issues (if any)

### Review Process

1. Automated checks must pass (CI)
2. At least one maintainer approval required
3. All review comments must be addressed
4. Squash and merge when approved

## Release Process

Releases are automated via GitHub Actions and GoReleaser:

1. Update version in relevant files
2. Update CHANGELOG.md
3. Create and push a tag:
   ```bash
   git tag -a v1.0.0 -m "Release v1.0.0"
   git push origin v1.0.0
   ```
4. GitHub Actions will automatically:
   - Run tests
   - Build binaries for all platforms
   - Create GitHub Release
   - Update Homebrew formula
   - Publish to npm

## Getting Help

- Open an issue for bugs or feature requests
- Join discussions in GitHub Discussions
- Check existing documentation in `docs/`

## Code of Conduct

- Be respectful and inclusive
- Welcome newcomers
- Focus on constructive feedback
- Assume good intentions

## License

By contributing to aicoder, you agree that your contributions will be licensed under the MIT License.
