# Contributing to Lunge

Thank you for your interest in contributing to Lunge! This document provides guidelines and instructions for contributing to the project.

## Table of Contents

- [Code of Conduct](#code-of-conduct)
- [Development Environment Setup](#development-environment-setup)
- [Code Style Requirements](#code-style-requirements)
- [Testing Requirements](#testing-requirements)
- [Pull Request Process](#pull-request-process)
- [Issue Reporting Guidelines](#issue-reporting-guidelines)

## Code of Conduct

This project adheres to the Contributor Covenant Code of Conduct. By participating, you are expected to uphold this code. Please read [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) before contributing.

## Development Environment Setup

### Prerequisites

- **Go 1.21 or later** - Lunge is written in Go and requires Go 1.21+ to build
- **Git** - For version control and cloning the repository

### Getting Started

1. **Fork the repository** on GitHub

2. **Clone your fork locally:**
   ```bash
   git clone https://github.com/YOUR_USERNAME/lunge.git
   cd lunge
   ```

3. **Add the upstream remote:**
   ```bash
   git remote add upstream https://github.com/wesleyorama2/lunge.git
   ```

4. **Install dependencies:**
   ```bash
   go mod download
   ```

5. **Verify the setup by running tests:**
   ```bash
   go test ./...
   ```

## Code Style Requirements

We maintain consistent code style across the project using Go's standard tooling:

### Formatting

All code must be formatted with `gofmt`:

```bash
gofmt -w .
```

Or use:
```bash
go fmt ./...
```

### Static Analysis

All code must pass `go vet`:

```bash
go vet ./...
```

### Linting

We recommend using `staticcheck` for additional linting:

```bash
# Install staticcheck
go install honnef.co/go/tools/cmd/staticcheck@latest

# Run staticcheck
staticcheck ./...
```

### General Guidelines

- Follow the [Effective Go](https://golang.org/doc/effective_go) guidelines
- Use meaningful variable and function names
- Add comments for exported functions, types, and packages
- Keep functions focused and reasonably sized
- Handle errors explicitly - don't ignore them

## Testing Requirements

### Running Tests

All changes must pass the existing test suite and include tests for new functionality:

```bash
# Run all tests with race detection
go test -race ./...

# Run tests with verbose output
go test -race -v ./...

# Run short tests only (excludes integration tests)
go test -race -short ./...
```

### Test Guidelines

- Write unit tests for new functions and methods
- Include both positive and negative test cases
- Use table-driven tests where appropriate
- Aim for meaningful coverage, not just high percentages
- Integration tests should be skipped with `-short` flag

### Benchmarks

If your changes affect performance-critical code, include benchmarks:

```bash
go test -bench=. -benchmem ./...
```

## Pull Request Process

1. **Create a feature branch** from `main`:
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** following the code style requirements above

3. **Write or update tests** as needed

4. **Ensure all checks pass:**
   ```bash
   go fmt ./...
   go vet ./...
   staticcheck ./...
   go test -race ./...
   ```

5. **Commit your changes** with a clear, descriptive commit message:
   ```bash
   git commit -m "Add feature: brief description of changes"
   ```

6. **Push to your fork:**
   ```bash
   git push origin feature/your-feature-name
   ```

7. **Open a Pull Request** against the `main` branch

### PR Requirements

- Provide a clear description of the changes
- Reference any related issues (e.g., "Fixes #123")
- Ensure all CI checks pass
- Be responsive to review feedback
- Keep PRs focused - one feature or fix per PR

### Review Process

- At least one maintainer review is required
- Address all review comments before merging
- Maintainers may request changes or ask questions
- Once approved, a maintainer will merge the PR

## Issue Reporting Guidelines

### Bug Reports

When reporting a bug, please include:

1. **Description** - Clear description of the bug
2. **Steps to Reproduce** - Minimal steps to reproduce the issue
3. **Expected Behavior** - What you expected to happen
4. **Actual Behavior** - What actually happened
5. **Environment** - Go version, OS, lunge version
6. **Additional Context** - Logs, screenshots, or other relevant information

### Feature Requests

For feature requests, please include:

1. **Problem Statement** - What problem does this solve?
2. **Proposed Solution** - How would you like it to work?
3. **Alternatives Considered** - Other approaches you've thought about
4. **Additional Context** - Any other relevant information

### Security Issues

For security vulnerabilities, please do **NOT** open a public issue. Instead, follow the process outlined in [SECURITY.md](SECURITY.md).

## Questions?

If you have questions about contributing, feel free to open an issue for discussion or reach out to the maintainers.

Thank you for contributing to Lunge! 🚀