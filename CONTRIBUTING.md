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

## Architecture Rules

- Keep runtime dependencies zero unless absolutely necessary.
- Respect layer boundaries:
  - `cmd`: flag parsing + presentation only
  - `internal/app`: use-case orchestration
  - `internal/domain`: pure scheduling logic
  - `internal/ci`: providers/cache/infrastructure

## Documentation Rules

- Update `docs/commands.md` when adding/changing flags.
- Update `docs/action.md` when changing action inputs/outputs.
- Update `README.md` for user-visible behavior changes.
- Add release notes in `CHANGELOG.md`.

## Pull Request Checklist

- [ ] Tests added/updated where applicable
- [ ] `go test ./...` passes
- [ ] `go vet ./...` passes
- [ ] `gofmt` applied
- [ ] Docs updated
- [ ] Changelog updated

## Governance Notes

Branch/review policy is documented in [`docs/governance.md`](docs/governance.md).
