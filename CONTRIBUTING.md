# Contributing to cpm

Thank you for your interest in contributing to cpm (Claude Plugin Manager)! We welcome contributions of all kinds, from bug reports and feature requests to code improvements and documentation.

## Getting Started

1. Fork the repository
2. Clone your fork: `git clone https://github.com/your-username/cpm.git`
3. Create a feature branch: `git checkout -b feature/your-feature-name`

## Development Setup

```bash
# Install tools (requires mise)
mise install

# Set up git hooks
lefthook install

# Build the application
mise run build

# Run tests
mise run test
```

## Making Changes

1. Follow the [Conventional Commits](https://www.conventionalcommits.org/) format for commit messages
2. Run `mise run fmt` to format your code
3. Run `mise run lint` to check for linting issues
4. Run `mise run test` to verify tests pass
5. Run `mise run ci` to run all checks locally before pushing

## Testing

Before submitting a pull request, please:

1. Run all tests: `mise run test`
2. Run linting: `mise run lint`
3. Build the application: `mise run build`
4. Manually test the changes in the TUI

## Pull Request Process

1. Ensure your branch is up to date with `main`
2. Create a pull request with a clear description of your changes
3. Link any related issues in the PR description
4. Ensure all CI checks pass
5. Request review from maintainers

## Questions?

Feel free to open an issue or discussion if you have questions about contributing or the development process.
