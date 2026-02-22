# Governance & Repository Policy

## Branch Protection Model

`main` is protected with:

- Required status checks: `build`, `action-smoke`, `Analyze (go)`, `docs-links`
- Required conversation resolution: enabled
- Force push/deletion: disabled
- Admin enforcement: enabled

## Review Policy

This repository currently uses **solo-maintainer mode**:

- `required_approving_review_count = 0`
- Rationale: GitHub does not allow approving your own PR; requiring one approval blocks solo maintenance.
- Safety net: mandatory CI + required conversation resolution + protected branch.

If you add maintainers later, switch to team mode:

- set required approvals to `1+`
- optionally require CODEOWNERS review

## Security Baseline

- Dependabot updates enabled
- Secret scanning + push protection enabled
- CodeQL workflow enabled
- SHA-pinned allowed actions policy enforced

## Documentation Policy

- User-facing behavior changes must update `README.md` and relevant `docs/*` pages.
- New CLI flags must be documented in `docs/commands.md`.
- Release-impacting changes must be added to `CHANGELOG.md`.

## Repository Features

- Issues: enabled (with templates)
- Discussions: enabled
- Projects: enabled
- Wiki: disabled (canonical docs live in `docs/`)

## Repository Operations Automation

- Release drafting by label category
- PR auto-labeling by file scope
- Scheduled stale triage
- First-time contributor greetings
- Label source-of-truth sync (`.github/labels.json`)
