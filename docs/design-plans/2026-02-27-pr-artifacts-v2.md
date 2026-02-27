# PR Artifacts v2 Design

## Summary

Replace the manual `go build` matrix in the PR artifacts workflow with GoReleaser snapshot builds to produce universal macOS binaries matching the release pipeline, add a PR comment with artifact download links that updates on each push, and chain builds after CI passes with cancel-in-progress concurrency.

## Definition of Done

The PR artifacts workflow is replaced so that:

1. **GoReleaser snapshot builds** produce the same artifact set as releases — including a universal macOS binary — by running `goreleaser release --snapshot --clean` with the existing `.goreleaser.yml`
2. **Snapshot versioning** uses the existing `version.txt` + commit-count scheme, producing tags like `v0.2.5-pr.42+abc1234`
3. **Build depends on CI** — snapshot builds only run after lint+test pass
4. **PR comment** is created (or updated on subsequent pushes) with the build version and a link to download artifacts
5. **Concurrency** — a cancel-in-progress group ensures only the latest push builds

**Out of scope:** Changing action version pinning (tags → SHAs), adding change detection (paths-filter), or modifying the release workflow.

## Acceptance Criteria

### DoD 1: GoReleaser snapshot builds

- **pr-artifacts-v2.AC1.1** (success): When a PR is opened against main and CI passes, the `build-snapshot` job runs `goreleaser release --snapshot --clean` and produces archives for linux/amd64, linux/arm64, darwin/universal, and windows/amd64
- **pr-artifacts-v2.AC1.2** (success): The macOS artifact is a single universal binary (not separate amd64 + arm64), named `cpm_*_darwin_universal.tar.gz`
- **pr-artifacts-v2.AC1.3** (success): All archives are uploaded as GitHub Actions artifacts with 7-day retention
- **pr-artifacts-v2.AC1.4** (success): A checksums.txt file is uploaded alongside the archives
- **pr-artifacts-v2.AC1.5** (failure): If GoReleaser build fails, the job fails and no artifacts are uploaded

### DoD 2: Snapshot versioning

- **pr-artifacts-v2.AC2.1** (success): The snapshot version follows the pattern `{base}.{count}-pr.{pr_number}+{short_sha}` (e.g., `0.2.5-pr.42+abc1234`), derived from `version.txt` + commit count since last version.txt change
- **pr-artifacts-v2.AC2.2** (success): `GORELEASER_CURRENT_TAG` is set to the computed base version tag (e.g., `v0.2.5`) so GoReleaser resolves `{{ .Version }}` correctly
- **pr-artifacts-v2.AC2.3** (success): The `PR_NUMBER` environment variable is passed to GoReleaser so the snapshot template can interpolate it

### DoD 3: Build depends on CI

- **pr-artifacts-v2.AC3.1** (success): The `build-snapshot` job has `needs: [lint, test]` and only runs after both pass
- **pr-artifacts-v2.AC3.2** (success): The lint and test jobs are defined within `pr-artifacts.yml`, matching the existing `ci.yml` configuration
- **pr-artifacts-v2.AC3.3** (success): `ci.yml` is scoped to `push` only (the `pull_request` trigger is removed) since PR CI is now handled by `pr-artifacts.yml`
- **pr-artifacts-v2.AC3.4** (failure): If lint or test fails, the build-snapshot job is skipped entirely

### DoD 4: PR comment

- **pr-artifacts-v2.AC4.1** (success): On first successful build, a new comment is posted to the PR with the build version, platform list, and artifact download link
- **pr-artifacts-v2.AC4.2** (success): On subsequent pushes to the same PR, the existing comment is found (by `body-includes` marker) and updated in-place rather than creating a new comment
- **pr-artifacts-v2.AC4.3** (success): The comment includes a link to `https://github.com/{repo}/actions/runs/{run_id}` for artifact download
- **pr-artifacts-v2.AC4.4** (failure): If comment creation/update fails, the build still succeeds (comment is non-blocking)

### DoD 5: Concurrency

- **pr-artifacts-v2.AC5.1** (success): A concurrency group keyed on `workflow-pr_number` ensures only one build runs per PR at a time
- **pr-artifacts-v2.AC5.2** (success): `cancel-in-progress: true` cancels the running build when a new push arrives

## Architecture

### Workflow Structure

The redesigned PR pipeline consolidates CI and artifact building into a single workflow file with three chained jobs:

```
pr-artifacts.yml
├── lint (runs golangci-lint via mise)
├── test (runs go test with race detector)
└── build-snapshot (needs: [lint, test])
    ├── Calculate version from version.txt
    ├── GoReleaser snapshot build
    ├── Upload artifacts (5 uploads: linux-amd64, linux-arm64, macos-universal, windows-amd64, checksums)
    └── Create/update PR comment
```

### GoReleaser Snapshot Configuration

A `snapshot` section is added to `.goreleaser.yml`:

```yaml
snapshot:
  version_template: "{{ .Version }}-pr.{{ .Env.PR_NUMBER }}+{{ .ShortCommit }}"
```

This template is only used when `--snapshot` is passed. The existing release builds are unaffected. GoReleaser computes `{{ .Version }}` from `GORELEASER_CURRENT_TAG` and `{{ .ShortCommit }}` from the git HEAD.

### Version Computation

The same shell logic from `auto-release.yml` is reused:

```bash
BASE_VERSION=$(cat version.txt | tr -d '\n')        # e.g., "0.2"
LAST_VERSION_COMMIT=$(git log -1 --format=%H version.txt)
COMMIT_COUNT=$(git rev-list --count ${LAST_VERSION_COMMIT}..HEAD)
TAG="v${BASE_VERSION}.${COMMIT_COUNT}"               # e.g., "v0.2.5"
```

`GORELEASER_CURRENT_TAG` is set to `$TAG`, and the snapshot template appends the PR context.

### PR Comment Strategy

Uses two community actions in sequence:
1. `peter-evans/find-comment` — searches for an existing comment containing a known marker string (`## PR Build`)
2. `peter-evans/create-or-update-comment` — creates a new comment if none found, or updates the existing one via `edit-mode: replace`

The marker string in the comment body acts as an identifier — only comments from this workflow contain it.

## Existing Patterns

| Pattern | Where | How We Follow It |
|---------|-------|-----------------|
| `version.txt` + commit count | `auto-release.yml` | Reuse the same version computation shell script |
| GoReleaser for builds | `release.yml` + `.goreleaser.yml` | Use GoReleaser snapshot mode with existing config |
| Tag-based action versions | All workflow files | Keep `@v4`/`@v6` style, don't switch to SHA pinning |
| mise for tool setup | `ci.yml`, current `pr-artifacts.yml` | Continue using `jdx/mise-action@v2` |
| Actions cache for Go modules | All workflows | Same cache key pattern |
| PR comment pattern | unquote `tui-pr.yml` | Adapt `peter-evans/find-comment` + `create-or-update-comment` |

## Implementation Phases

### Phase 1: GoReleaser snapshot config

Add the `snapshot` section to `.goreleaser.yml`. This is a safe, isolated change that has zero effect on release builds (the template is only used with `--snapshot`).

**Files:** `.goreleaser.yml`

### Phase 2: Rewrite PR artifacts workflow

Replace the contents of `.github/workflows/pr-artifacts.yml` with the new lint → test → build-snapshot pipeline. This includes:
- Concurrency group
- Lint and test jobs (mirroring current `ci.yml`)
- GoReleaser snapshot build with version computation
- Artifact uploads (linux-amd64, linux-arm64, macos-universal, windows-amd64, checksums)
- PR comment with find + create-or-update pattern

**Files:** `.github/workflows/pr-artifacts.yml`

### Phase 3: Scope ci.yml to push-only

Remove the `pull_request` trigger from `ci.yml` so PR CI is handled exclusively by `pr-artifacts.yml`. This avoids redundant CI runs on PRs.

**Files:** `.github/workflows/ci.yml`

## Additional Considerations

### GoReleaser `before.hooks`

The existing `.goreleaser.yml` runs `go mod tidy` and `go test ./...` as pre-build hooks. In snapshot mode, `go test` will run again after the lint+test jobs already passed. This is redundant but harmless. Removing the test hook would speed up PR builds but also affects releases, which is out of scope for this design.

### Archive naming with `+` in version

The snapshot version contains a `+` character (semver build metadata separator). GoReleaser handles this correctly in archive filenames. The upload-artifact glob patterns (`dist/cpm_*_linux_amd64.tar.gz`) match regardless of the version string.

### Permissions

The workflow needs `contents: read` (checkout) and `pull-requests: write` (comment). These are the same permissions the current `pr-artifacts.yml` already declares.

## Glossary

| Term | Definition |
|------|-----------|
| **GoReleaser snapshot** | A GoReleaser build mode (`--snapshot`) that produces all artifacts without creating a GitHub release or pushing to Homebrew. Used for pre-release/preview builds. |
| **Universal binary** | A macOS Mach-O binary containing code for multiple architectures (amd64 + arm64). Users don't need to choose the right download for their Mac. |
| **`GORELEASER_CURRENT_TAG`** | Environment variable that tells GoReleaser what version tag to use when no actual git tag exists (e.g., in snapshot builds). |
| **Concurrency group** | GitHub Actions feature that groups workflow runs and optionally cancels older runs when a new one starts, keyed by a user-defined string. |
