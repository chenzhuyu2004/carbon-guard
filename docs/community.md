# Community Guide

This document explains how repository channels and automation are used.

## Channels

- Issues: reproducible bugs and actionable feature requests.
- Discussions: usage questions, adoption ideas, and architecture topics.
- Pull Requests: scoped implementation changes with tests/docs.
- Security policy: private vulnerability disclosure only.

## Templates and Standards

- Issue templates: bug report, feature request, and question routing forms.
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

## Automation Workflows

- `release-drafter.yml`: maintain draft release notes from merged PR labels.
- `labeler.yml`: auto-apply labels to PRs by changed files.
- `stale.yml`: mark/close inactive issues and PRs on schedule.
- `greetings.yml`: first-contribution greeting on issues and PRs.
- `sync-labels.yml`: sync repository labels from `.github/labels.json`.

## Projects and Wiki

- Projects: enabled for roadmap/task planning.
- Wiki: disabled to keep canonical docs in repository (`docs/`).
