# Release bootstrap — `hop-top/ben`

Status sweep of the dotgithub SKILL §"Bootstrap checklist (first-time
setup)" for this repo. CI-side wiring is committed; web-side actions
(org secrets, mirror repos, App installation) are flagged below.

Reference: `~/.w/ideacrafterslabs/dotgithub/SKILL.md`.

## Naming + tag shape

- [x] Repo follows the naming convention — Go-only repo, bare-name
      `hop-top/ben`. No polyglot source, no `<name>-go` (Go always
      takes the bare slot).
- [x] Every `component` in `release-please-config.json` is a single
      segment — `component: "ben"` (no `/`), so tags will match
      `tags: ['*/v*']`.
- [x] Three-way name alignment per SKILL — release-please component
      `ben` + mirror repo basename `ben`. Note: `publish.yml`
      `ecosystems` key is the ecosystem-typed `go` (matching the SKILL
      Quick-start verbatim), not the component name. If
      `publish-on-tag.yml@v0` rejects this with `Unknown component
      'ben'` at first dry-run, swap to `ben: { dir: ., ecosystem: go,
      mirror: hop-top/ben }` and re-tag.

## Prerelease channel

- [x] All four pieces present in `release-please-config.json`:
      `prerelease: true`, `prerelease-type: "alpha.0"`,
      `versioning: "prerelease"`, `bump-minor-pre-major: true`.
- [x] Manifest seed prerelease-shaped: `{".": "0.2.0-alpha.0"}`.
- [ ] Verified with a release-please dry-run — requires
      `gh auth token` and the repo to exist on GitHub. Run after
      mirror repo + App install land:
      ```sh
      npx release-please@latest release-pr \
        --token "$(gh auth token)" \
        --repo-url https://github.com/hop-top/ben \
        --config-file .github/release-please-config.json \
        --manifest-file .github/.release-please-manifest.json \
        --target-branch main \
        --dry-run | grep '^title:'
      ```

## Secrets (org-level)

CI cannot create org secrets. Human/web action required:

- [ ] `GH_MIRROR_PAT` (fine-grained, `Administration: RW` +
      `Contents: RW` on `hop-top/ben`, `hop-top/homebrew-tap`,
      `hop-top/scoop-bucket`). Likely already exists org-wide.
- [ ] `RELEASE_BOT_APP_ID` + `RELEASE_BOT_PRIVATE_KEY` — release-bot
      GitHub App credentials. Per SKILL: "already set across the org".
      Verify reachable from this repo before first release.
- [x] `NPM_REGISTRY_TOKEN` — N/A (Go-only).
- [x] `CARGO_REGISTRY_TOKEN` — N/A (Go-only).
- [x] `PYPI_REGISTRY_TOKEN` — N/A (Go-only).

## Registry pre-registration

- [x] npm — N/A.
- [x] crates.io — N/A.
- [x] PyPI — N/A.
- [x] Packagist — N/A.
- [x] Go (proxy.golang.org) — pulls from git tags automatically. No
      pre-registration. Watch for SKILL §Go module proxy: ghost
      versions if `hop.top/ben` had a prior incarnation.

## Repo hygiene

- [x] Rust target/ — N/A.
- [x] Rust feature-gated tests — N/A.
- [x] TS native bindings — N/A.

## Workflow setup

- [x] `release-please.yml` declares `workflow_dispatch: {}` for manual
      retrigger after sibling-PR conflicts.
- [x] `publish.yml` uses `@v0` (rolling major), not `@main`, not pinned
      `@v0.x.y`.
- [x] `goreleaser-on-tag.yml` also uses `@v0` (rolling major).

## Mirror repos

CI cannot create mirror repos. Human/web action required:

- [ ] **Create `hop-top/ben` mirror repo on GitHub** (empty / fresh).
      The `mirror-subtree` workflow auto-archives it after the first
      sync. Without this, the mirror push step in `publish.yml` fails.
- [ ] **Install `release-bot` GitHub App on `hop-top/ben`** (source
      repo) — required for `release-please.yml` and
      `goreleaser-on-tag.yml` to mint App tokens. Per SKILL: App must
      be installed on every source repo + every package-manager target
      repo.
- [ ] **Confirm `hop-top/homebrew-tap` exists** and the `release-bot`
      App has `Contents: RW` there (org-wide tap; goreleaser pushes
      formula updates). Likely already in place.
- [ ] **Confirm `hop-top/scoop-bucket` exists** and the `release-bot`
      App has `Contents: RW` there. Likely already in place.

## Post-merge sanity (run AFTER first release-please PR merges)

- [ ] `gh run list --workflow publish.yml --repo hop-top/ben` shows a
      run triggered by the `ben/v0.2.0-alpha.0` tag push.
- [ ] `gh run list --workflow goreleaser.yml --repo hop-top/ben`
      similarly shows a parallel run on the same tag.
- [ ] If either failed: fix on `main` then **delete + recreate the
      tag** (do NOT `gh run rerun` — `publish.yml` snapshots from the
      tag's commit, not current main). See SKILL §Re-triggering a
      failed publish.

## Summary of outstanding human/web actions

1. Create `hop-top/ben` GitHub repo (empty source repo — this worktree
   tracks `chore/dotgithub-pipeline` off a yet-to-be-created `main`).
2. Create `hop-top/ben` mirror repo (separate empty repo; could share
   the same name with subtree push pattern — see SKILL §Go: root-
   component caveats).
3. Install `release-bot` App on source + mirror + `homebrew-tap` +
   `scoop-bucket` repos.
4. Confirm org-level `GH_MIRROR_PAT`, `RELEASE_BOT_APP_ID`,
   `RELEASE_BOT_PRIVATE_KEY` reachable from this repo.
5. Run the release-please dry-run command above to verify the first
   proposed PR title is `chore(main): release ben 0.2.0-alpha.1`
   (counter-incrementing from the manifest seed `0.2.0-alpha.0`).
