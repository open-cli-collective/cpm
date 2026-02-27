# Human Test Plan: PR Artifacts v2

**Implementation plan:** `docs/implementation-plans/2026-02-27-pr-artifacts-v2/`
**Generated:** 2026-02-27

## Prerequisites

- Implementation branch pushed to `origin/brajkovic/pr-artifacts-v2`
- A PR opened against `main`
- Access to the GitHub Actions UI for the repository
- `gh` CLI authenticated with access to the repository

## Phase 1: Workflow Trigger Verification (AC3.3)

| Step | Action | Expected |
|------|--------|----------|
| 1.1 | Open the PR on GitHub. Navigate to the "Checks" tab. | A workflow named "PR Artifacts" appears. No workflow named "CI" appears. |
| 1.2 | Wait for "PR Artifacts" workflow to start. | The workflow run shows three jobs: "Lint", "Test", and "Build Snapshot". |
| 1.3 | After the PR is merged, push a separate commit to `main`. Navigate to the repository Actions tab and filter by "CI" workflow. | The "CI" workflow triggers on the push to `main`. It runs `lint`, `test`, and `build` jobs. |

## Phase 2: Job Dependency Verification (AC3.1, AC3.2, AC3.4)

| Step | Action | Expected |
|------|--------|----------|
| 2.1 | In the "PR Artifacts" workflow run, observe the job graph in the GitHub Actions UI. | "Build Snapshot" appears downstream of both "Lint" and "Test" with dependency arrows. It does not start until both predecessors complete. |
| 2.2 | Compare the `lint` job steps in `pr-artifacts.yml` against `ci.yml`. | Both use: `actions/checkout@v4`, `jdx/mise-action@v2`, `actions/cache@v4` (same cache key), and `mise run lint`. Steps are identical. |
| 2.3 | Compare the `test` job steps in `pr-artifacts.yml` against `ci.yml`. | Both use: `actions/checkout@v4`, `jdx/mise-action@v2`, `actions/cache@v4` (same cache key), `go test -v -race -coverprofile=coverage.out ./...`, and `codecov/codecov-action@v4`. Steps are identical. |
| 2.4 | (Optional) To test failure propagation: create a temporary commit with a lint error (e.g., add an unused import in a Go file), push to the PR branch. | "Lint" job fails. "Build Snapshot" shows status "Skipped" in the Actions UI. |
| 2.5 | (If 2.4 was done) Revert the lint failure commit and push. | "Lint" passes, "Test" passes, and "Build Snapshot" runs. |

## Phase 3: Snapshot Build and Artifacts (AC1.1-AC1.5, AC2.1-AC2.3)

| Step | Action | Expected |
|------|--------|----------|
| 3.1 | In the "Build Snapshot" job logs, expand the "Compute snapshot version" step. | The step outputs a `tag` value matching `v{base}.{count}` (e.g., `v0.2.5`) and a `snapshot` value matching `{base}.{count}-pr.{N}+{sha}` (e.g., `0.2.5-pr.42+abc1234`). |
| 3.2 | In the "Run GoReleaser snapshot" step logs, find where GoReleaser reports the version it is using. | GoReleaser reports the tag from step 3.1 as `GORELEASER_CURRENT_TAG`. The snapshot version in the output includes the correct PR number (confirming `PR_NUMBER` env var was interpolated). |
| 3.3 | In the GoReleaser output, find the list of build targets. | The output lists builds for: `linux/amd64`, `linux/arm64`, `darwin/amd64`, `darwin/arm64`, and `windows/amd64`. Darwin builds are merged into a universal binary. |
| 3.4 | Navigate to the workflow run's "Artifacts" section (bottom of the run summary page). | Five artifacts are listed: `cpm-linux-amd64`, `cpm-linux-arm64`, `cpm-macos-universal`, `cpm-windows-amd64`, `checksums`. |
| 3.5 | Download the `cpm-macos-universal` artifact. Extract the archive inside. | The archive filename matches `cpm_*_darwin_universal.tar.gz`. It contains a single `cpm` binary (not separate amd64 and arm64 archives). |
| 3.6 | Download the `checksums` artifact. | It contains a `checksums.txt` file with SHA256 entries, one per archive file. |
| 3.7 | Verify retention via the GitHub API: run `gh api repos/{owner}/{repo}/actions/artifacts --jq '.artifacts[] \| select(.name \| startswith("cpm-")) \| {name, expires_at}'`. | Each artifact's `expires_at` is approximately 7 days from creation. |

## Phase 4: PR Comment (AC4.1-AC4.4)

| Step | Action | Expected |
|------|--------|----------|
| 4.1 | After the first successful "Build Snapshot" run, navigate to the PR conversation tab. | A comment from `github-actions[bot]` is present. |
| 4.2 | Read the comment content. | The comment contains: (1) a `## PR Build` header, (2) a `Version:` line showing the snapshot version from step 3.1, (3) a table listing four platforms (Linux amd64, Linux arm64, macOS universal, Windows amd64), (4) a "Download artifacts" link, (5) a line showing the commit SHA. |
| 4.3 | Click the "Download artifacts" link in the comment. | It navigates to the correct workflow run page. |
| 4.4 | Push a second commit to the PR branch. Wait for the new "Build Snapshot" run to complete. | The PR conversation still has exactly one `## PR Build` comment (not two). The version string and commit SHA in the comment are updated to reflect the new commit. |

## Phase 5: Concurrency (AC5.1, AC5.2)

| Step | Action | Expected |
|------|--------|----------|
| 5.1 | Open two separate PRs against `main` simultaneously. | Each PR triggers its own independent "PR Artifacts" workflow run. The runs do not interfere with each other. |
| 5.2 | On one PR, push two commits in rapid succession (before the first build completes). Observe the Actions tab. | The first workflow run is cancelled. Only the second run completes. |

## End-to-End: Full PR Lifecycle

1. Open a PR with a valid Go change against `main`.
2. Wait for "PR Artifacts" workflow to trigger. Confirm no "CI" workflow triggers.
3. Watch the job graph: "Lint" and "Test" run first, then "Build Snapshot" starts.
4. After "Build Snapshot" completes, verify 5 artifacts in the run summary.
5. Check the PR conversation for the `## PR Build` comment with correct version, platform table, and download link.
6. Click the download link; confirm it points to the correct run.
7. Download `cpm-linux-amd64` artifact. Extract the archive. Run `./cpm --version` and confirm the version includes the snapshot string.
8. Push a follow-up commit. Confirm the existing comment is updated in-place with the new version.
9. Merge the PR. Push a commit to `main`. Confirm the "CI" workflow triggers on push.

## Traceability

| Acceptance Criterion | Automated Test | Manual Step |
|----------------------|----------------|-------------|
| Phase 1 config validity | GoReleaser `check` (PASS) | -- |
| Phase 2 YAML syntax | Python YAML parse (PASS) | -- |
| Phase 3 YAML syntax | Python YAML parse (PASS) | -- |
| AC1.1 Platform archives | -- | 3.3, 3.4 |
| AC1.2 Universal macOS binary | -- | 3.5 |
| AC1.3 Artifacts with 7-day retention | -- | 3.4, 3.7 |
| AC1.4 checksums.txt uploaded | -- | 3.6 |
| AC1.5 Build failure propagation | `if-no-files-found: error` verified | 2.4 (optional) |
| AC2.1 Snapshot version format | -- | 3.1 |
| AC2.2 GORELEASER_CURRENT_TAG set | -- | 3.2 |
| AC2.3 PR_NUMBER env var | -- | 3.2 |
| AC3.1 build-snapshot depends on lint+test | `needs: [lint, test]` verified | 2.1 |
| AC3.2 Lint/test match ci.yml | -- | 2.2, 2.3 |
| AC3.3 ci.yml push-only | YAML parse confirms no `pull_request` key (PASS) | 1.1, 1.3 |
| AC3.4 Build skipped on failure | `needs` dependency verified | 2.4 (optional) |
| AC4.1 PR comment posted | -- | 4.1, 4.2 |
| AC4.2 Comment updated in-place | -- | 4.4 |
| AC4.3 Comment has run link | -- | 4.3 |
| AC4.4 Comment failure non-blocking | `continue-on-error: true` verified | -- |
| AC5.1 Concurrency group per PR | Group key verified | 5.1 |
| AC5.2 Cancel-in-progress | `cancel-in-progress: true` verified | 5.2 |
