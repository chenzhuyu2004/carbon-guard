# Contributing to Carbon Guard

Thanks for contributing.

## Development Setup

1. Install Go 1.22+.
2. Clone the repository.
3. Run:

```bash
make test
make lint
make build
```

## Coding Rules

- Keep zero third-party runtime dependencies unless absolutely necessary.
- Follow the current architecture boundaries:
  - `cmd` for CLI parsing and presentation only
  - `internal/app` for use-case orchestration
  - `internal/domain` for pure scheduling logic
  - `internal/ci` for provider adapters and cache implementation
- Keep behavior changes covered by tests.

## Pull Request Checklist

- [ ] Tests added/updated
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes
- [ ] `gofmt` applied
- [ ] README/docs updated for user-visible changes
- [ ] Changelog updated (`CHANGELOG.md`)

## Commit Guidance

Prefer small, themed commits:

1. Feature or refactor
2. Tests
3. Docs
