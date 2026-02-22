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

### Changed

- Action runtime input contract now prefers explicit `duration` or `start_time`.
- `.gitignore` expanded for build, coverage, logs, and editor artifacts.
- Text report emission display now auto-scales across common units for readability while preserving `kgCO2` reference values.

### Fixed

- Dedicated exit code for carbon budget exceedance.
