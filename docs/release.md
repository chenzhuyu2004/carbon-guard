# Release Process

## Pre-Release Checklist

1. Update `CHANGELOG.md` under `[Unreleased]`.
2. Run local checks:

```bash
make test
make lint
make build
```

3. Confirm CI is green on `main`.
4. Confirm branch protection checks are up to date.

## Tag and Release

```bash
git checkout main
git pull origin main
git tag vX.Y.Z
git push origin vX.Y.Z
```

Then create GitHub Release:

```bash
gh release create vX.Y.Z --title "vX.Y.Z" --notes-file CHANGELOG.md
```

## Post-Release

1. Verify release page and artifacts.
2. Validate README examples against released tag.
3. Update roadmap/issues for next milestone.
