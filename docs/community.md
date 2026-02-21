# Community Guide

This document explains how repository features are used.

## Channels

- Issues: reproducible bugs and actionable feature requests.
- Discussions: usage questions, adoption ideas, and architecture topics.
- Pull Requests: scoped implementation changes with tests/docs.
- Security policy: private vulnerability disclosure only.

## Templates and Standards

- Issue templates: bug report and feature request forms.
- PR template: checklist for tests, lint, docs, and changelog.
- Code owners: default owner routing via `.github/CODEOWNERS`.

## Maintainer Modes

### Solo-maintainer mode

- Required approvals: `0`
- Required checks: `build`, `action-smoke`, `Analyze (go)`, `docs-links`
- Required conversation resolution: enabled

### Team mode (recommended when maintainers > 1)

- Required approvals: `1+`
- Optionally require code-owner reviews.

## Projects and Wiki

- Projects: enabled for roadmap/task planning.
- Wiki: disabled to keep canonical docs in repository (`docs/`).
