# Contributing to PlexLinker

Thank you for your interest in contributing to PlexLinker! This document provides guidelines and instructions for contributing.

## 📋 Table of Contents
- [Code of Conduct](#code-of-conduct)
- [Getting Started](#getting-started)
- [Development Workflow](#development-workflow)
- [Coding Standards](#coding-standards)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)

## 🤝 Code of Conduct

- Be respectful and inclusive
- Welcome newcomers and help them get started
- Focus on constructive feedback
- Keep discussions on-topic

## 🚀 Getting Started

### Prerequisites
- Go 1.21 or higher
- Git
- aria2c (for testing download scripts)

### Setup Development Environment

```bash
# Clone the repository
git clone https://github.com/yourusername/plexlinker.git
cd plexlinker

# Install dependencies
go mod download

# Build the project
go build -o plexlinker ./cmd/plexlinker

# Run tests
go test ./...
```

## 🔄 Development Workflow

### 1. Create a Branch
```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/your-bug-fix
```

### 2. Make Changes
- Write clear, well-documented code
- Follow existing code style
- Add tests for new functionality
- Update documentation as needed

### 3. Test Your Changes
```bash
# Run all tests
go test ./...

# Run specific package tests
go test ./internal/downloader

# Run with coverage
go test -cover ./...

# Run linters
go vet ./...
go fmt ./...
```

### 4. Commit Your Changes
```bash
git add .
git commit -m "feat: add new feature description"
# or
git commit -m "fix: fix bug description"
```

**Commit Message Format:**
- `feat:` New feature
- `fix:` Bug fix
- `docs:` Documentation changes
- `test:` Adding or updating tests
- `refactor:` Code refactoring
- `style:` Code style changes (formatting)
- `chore:` Maintenance tasks

### 5. Push and Create Pull Request
```bash
git push origin feature/your-feature-name
```

Then open a Pull Request on GitHub.

## 📝 Coding Standards

### Go Style Guide
- Follow standard Go conventions
- Use `gofmt` for formatting
- Use `go vet` for static analysis
- Keep functions focused and small
- Write clear variable and function names

### Project-Specific Guidelines

#### File Organization
```
internal/
├── package/
│   ├── file.go           # Main implementation
│   ├── file_windows.go   # Windows-specific code (//go:build windows)
│   ├── file_default.go   # Unix-specific code (//go:build !windows)
│   └── file_test.go      # Tests
```

#### Build Tags
Use the new build tag format:
```go
//go:build windows
// NOT: // +build windows
```

#### Error Handling
```go
// Good
if err != nil {
    return fmt.Errorf("failed to do something: %w", err)
}

// Bad
if err != nil {
    return err  // Lost context
}
```

#### Logging
Use the centralized logger:
```go
import "plexlinker/internal/logger"

logger.Get().Main().Printf("Message: %v", value)
```

#### UI Components
- Keep UI logic in `internal/ui/`
- Use tcell for terminal rendering
- Implement keyboard navigation consistently
- Handle window resize events

## 🧪 Testing

### Writing Tests
```go
func TestFeature(t *testing.T) {
    // Arrange
    input := "test"
    expected := "result"

    // Act
    result := DoSomething(input)

    // Assert
    if result != expected {
        t.Errorf("Expected %s, got %s", expected, result)
    }
}
```

### Test Coverage
- Aim for >70% coverage on new code
- Focus on critical paths first
- Test edge cases and error conditions

### Platform-Specific Testing
```go
//go:build windows

func TestWindowsFeature(t *testing.T) {
    // Windows-specific test
}
```

## 📤 Submitting Changes

### Pull Request Checklist
- [ ] Code builds successfully
- [ ] All tests pass
- [ ] New tests added for new features
- [ ] Documentation updated (README, CLAUDE.md)
- [ ] Code formatted with `gofmt`
- [ ] No `go vet` warnings
- [ ] Commit messages follow convention
- [ ] PR description explains changes clearly

### Pull Request Template
```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
How has this been tested?

## Checklist
- [ ] Code follows project style
- [ ] Tests pass
- [ ] Documentation updated
```

### Code Review Process
1. Maintainer reviews PR
2. Feedback provided (if needed)
3. Changes requested (if needed)
4. Approval and merge

## 🐛 Reporting Bugs

### Bug Report Template
```markdown
**Describe the bug**
A clear description of the bug

**To Reproduce**
Steps to reproduce the behavior:
1. Go to '...'
2. Press '....'
3. See error

**Expected behavior**
What you expected to happen

**Screenshots**
If applicable, add screenshots

**Environment:**
- OS: [e.g., Windows 11]
- PlexLinker Version: [e.g., 1.0.0]
- Go Version: [e.g., 1.21.0]

**Additional context**
Any other context about the problem
```

## 💡 Feature Requests

We welcome feature requests! Please:
1. Check existing issues first
2. Describe the use case clearly
3. Explain why it would be useful
4. Consider implementation complexity

## 📚 Additional Resources

- [Go Documentation](https://go.dev/doc/)
- [tcell Documentation](https://github.com/gdamore/tcell)
- [Project Wiki](https://github.com/yourusername/plexlinker/wiki)

## 🙏 Thank You!

Your contributions make PlexLinker better for everyone. We appreciate your time and effort!

---

Questions? Open an issue or start a discussion on GitHub.
