---
summary: "Release checklist for spogo (GitHub release binaries via GoReleaser)"
---

# Releasing `spogo`

Always do **all** steps below (CI + changelog + tag + GitHub release assets). No partial releases.

Shortcut (if you want scripts later): create them to mirror this doc.

Assumptions:
- Repo: `openclaw/spogo`
- Binary: `spogo`
- GoReleaser config: `.goreleaser.yaml`

## 0) Prereqs
- Clean working tree on `main`.
- Go toolchain installed (version from `go.mod`).
- CI is green.

## 1) Verify build is green
```sh
./scripts/lint.sh
./scripts/check-coverage.sh 90
```

Confirm GitHub Actions `CI` is green for the commit you’re tagging:
```sh
gh run list -L 5 --branch main
```

## 2) Update changelog
- Update `CHANGELOG.md` for the version you’re releasing.

Example heading:
- `## 0.1.0 - 2026-01-02`

## 3) Merge the release prep
```sh
git checkout main
git pull --ff-only

# Commit the dated changelog section and any release tweaks on a PR.
# Merge only after required CI is green.
```

Do not create or push the release tag manually. The unified workflow freezes the release source and creates the tag.

## 4) Dispatch and verify the unified release

Run the thin caller with the version without a `v` prefix:

```sh
gh workflow run release-unified.yml -f version=X.Y.Z
gh run list -L 5 --workflow release-unified.yml
gh run watch <run-id> --exit-status
```

Watch validation, tag creation, build/sign/notarize, draft creation, independent verification, publication, Homebrew handoff, and closeout. The legacy workflow in `release-legacy.yml` is manual-only and is not the production release path.

After success:

- Download every published asset listed in `SHA256SUMS` and verify its SHA-256 digest.
- Run `codesign --verify --strict --check-notarization` on every macOS binary and confirm its designated requirement is stable.
- Confirm published release notes are byte-identical to the dated changelog section.
- Confirm the `spogo` formula in the Homebrew tap points at the new version and matching SHA-256 hashes.
- Confirm the closeout PR merged and opened the next `Unreleased` changelog section.

## Notes
- GoReleaser publishes binaries for macOS, Linux, and Windows.
- macOS binaries are signed with the OpenClaw Foundation Developer ID and notarized by Apple.
