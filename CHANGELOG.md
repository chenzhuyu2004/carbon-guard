# Changelog

All notable changes to this project are documented in this file.

## [Unreleased]

### Added

- Carbon budget gating flags for `run`:
  - `--budget-kg`
  - `--baseline-kg`
  - `--fail-on-budget`
- GitHub Action inputs/outputs for budget and baseline signals.
- CI step summary and PR comment reporting workflow.
- Governance documents: CONTRIBUTING, CODE_OF_CONDUCT, SECURITY.
- Full docs suite:
  - Action guide
  - Architecture notes
  - Troubleshooting
  - FAQ
  - Release process
  - Workflow examples
- Shared CLI config package (`internal/config`) with JSON config + env fallback.
- New command flag `--config` on `run-aware`, `optimize`, and `optimize-global`.
- Example config file: `docs/examples/carbon-guard.json`.
- `run-aware` no-regret guard flags:
  - `--max-delay-for-gain`
  - `--min-reduction-for-wait`
- Provider error taxonomy in `internal/ci`:
  - `auth`
  - `rate_limit`
  - `network`
  - `upstream`
  - `invalid_data`

### Changed

- Action runtime input contract now prefers explicit `duration` or `start_time`.
- `.gitignore` expanded for build, coverage, logs, and editor artifacts.
- Text report emission display now auto-scales across common units for readability while preserving `kgCO2` reference values.
- Shared option precedence is now consistently `Flags > Env > Config File > Defaults`.
- CI build job now includes `go vet ./...`.
- Action smoke workflow uses explicit `duration` input for deterministic validation.
- Machine-readable JSON outputs now include `schema_version`:
  - `run --json`
  - `optimize --output json`
  - `optimize-global --output json`
  - JSON error contract

### Fixed

- Dedicated exit code for carbon budget exceedance.
