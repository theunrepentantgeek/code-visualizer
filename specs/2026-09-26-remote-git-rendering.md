# Rendering a Remote Git Repository

**Date:** 2026-09-26
**Issue:** #759
**Status:** Research / proposed design; no remote input is implemented yet.

## Answer

Yes, with the existing `github.com/go-git/go-git/v5` dependency (v5.19.2).
go-git can fetch a remote into a bare repository without checking out files.
Its commits and trees can then feed the existing `source.NewGitFS` and
`scan.ScanTree` pipeline. This avoids requiring a pre-existing local working
tree, but **does not** mean reading arbitrary remote blobs on demand: Git
transport first transfers the selected repository objects to local storage.
No additional Go dependency or external `git clone` command is required for
HTTPS/SSH remotes.

## What already exists

- `internal/source/gitfs.go` exposes a go-git `object.Tree` as a read-only
  `fs.FS`; the scanner consumes `source.Tree.FS` and `RepoFS`
  (`internal/scan/fs_walker.go`). Blob reads, binary exclusion, filters,
  symlinks, and file-content metrics can use this path unchanged.
- `internal/stages/source.go` already replaces the working filesystem with
  `NewGitFS` for a committed `--until` snapshot and scopes a subtree with
  `git.Snapshot.Subtree`. Snapshot time supplies the reference clock.
- Git metrics, changed-only filtering, and alluvial snapshots use go-git
  history via `internal/provider/git`, not the Git CLI. However,
  `getService` currently opens a **local path** with `PlainOpenWithOptions`,
  and many loaders derive repository-relative paths from `RepoRoot` and
  scanned node paths. Alluvial resolves each reference through the same
  source stage.
- CLI commands currently accept a directory path. `ValidatePaths` calls
  `os.Stat` on it; `ResolveSource` starts with `source.WorkingTree`, and
  Git-required checks discover a local repository. Passing a URL as the
  existing target argument therefore does not work.

## Proposed first implementation

1. Add an explicit remote input option shared by visualization commands and
   presets (for example, `--remote <https-or-ssh-url>`). Keep the existing
   positional target as a **repository-relative directory** in remote mode,
   defaulting to `.` there; local mode retains its current required directory
   and behavior. Reject absolute paths, `..` traversal, and ambiguous
   local/remote combinations. Continue writing images/exports to local output
   paths. Default to the remote's HEAD; retain `--until` (tag, commit ID, or
   date) and `--from` history semantics. An explicit branch selector can be
   added if selecting a non-default branch is required; do not silently
   reinterpret `--until` as a branch name.
2. Once per render, create a private temporary directory and call go-git's
   `PlainCloneContext(ctx, dir, true, &git.CloneOptions{URL: url})` to fetch a
   **bare** repository. Do not check out a worktree. Start with complete
   history (`Depth: 0`) and fetched tags/refs needed by the selected
   revisions: commit counts, authorship, diffs, date selection, and alluvial
   references require their ancestors. Resolve requested references only
   after fetching; report an unavailable ref rather than substituting HEAD.
   Reuse this one fetched repository for every alluvial snapshot and remove
   the temporary directory after the whole render, including on error.
3. Branch `ValidatePaths` and `ResolveSource` on input kind: validate the
   remote URL and repository-relative target without `os.Stat` on that target;
   still validate the output directory locally. Resolve HEAD or the chosen
   `--until` commit against the fetched repository, select its subtree, and
   populate `source.Tree` with `NewGitFS(subtree, commitTime)` and
   `RepoFS: NewGitFS(fullTree, commitTime)`. Set `RepoBase` and a stable,
   absolute logical `RootPath` under the temporary bare repository so
   existing `filepath.Rel` history joins work; set `RepoRoot` to the bare
   repository path. Do not use `WorkingTree` or filesystem existence checks
   for committed content. Set `Snapshot` and `ReferenceNow` consistently
   with current historical scans.
4. Make Git-required checks accept the resolved remote source instead of
   rediscovering a working tree. The existing `getService` explicitly tolerates
   bare repositories; audit its path-based callers and cached service lifetime
   against temporary-directory cleanup. Preserve `--changed-only`, Git
   metrics, authorship, and alluvial history behavior. The local-path pipeline
   must remain unchanged.

This temporary *bare* clone is the smallest adaptation to the current
path-based Git service. If avoiding even temporary on-disk object storage is
required, go-git also supports `CloneContext(memory.NewStorage(), nil, ...)`;
that would require passing the repository handle into the service instead of
reopening a path, and supplying stable logical paths to existing metric
loaders. In-memory storage can exhaust RAM on large repositories and is not
recommended as the first implementation.

## Limits and safeguards

- A full fetch may be expensive in bandwidth, disk, and CPU; no working
  checkout does **not** avoid downloading blobs. A shallow clone (`Depth > 0`)
  may be an opt-in optimization only for snapshot-only rendering, not a
  default for history-dependent metrics, date bounds, or multiple revisions:
  missing ancestors distort results. go-git's clone options do not provide
  Git partial-clone blob filtering. Document resource limits and support
  context cancellation/timeouts for network operations.
- HTTPS and SSH need explicit authentication configuration (for example,
  go-git's HTTP `BasicAuth` with a token or SSH auth with host-key
  verification). Never put tokens in URLs, rendered output, or logs; redact
  remote URLs from errors. Disallow unintended local/file or unauthenticated
  `git://` transports if the feature is exposed to untrusted callers.
  Private remotes must fail clearly without suitable credentials.
- Preserve the current GitFS policy: submodules are omitted, and symlinks
  outside the selected source are not followed. Output and config files are
  local; auto-loading config from the remote repository is not implied.
  A remote fetch produces a committed snapshot, not uncommitted worktree
  changes. References absent from the fetched objects must fail explicitly.

## Acceptance tests for implementation

Use a temporary fixture repository served via a test-controlled transport
(file transport needs a local Git executable) or a local HTTP Git server; no
public network or credentials in tests.

1. Render a simple remote tree and compare its scanned paths and file metrics
   to the same local committed snapshot; verify no worktree checkout.
2. Render a subdirectory and a deleted-at-HEAD directory selected via
   `--until`; reject missing refs and escaping/absolute subdirectory paths.
3. Verify `--from`/`--until`, changed-only, Git metrics, and two alluvial
   references against the same repository's local results; assert one fetch
   per invocation and cleanup on success and failure.
4. Reject inaccessible/unauthorized remotes without logging secrets; keep
   existing local path and output validation tests passing.

## Library references

- go-git v5.19.2 [clone APIs](https://github.com/go-git/go-git/blob/v5.19.2/repository.go)
  and [CloneOptions](https://github.com/go-git/go-git/blob/v5.19.2/options.go)
  (bare clone, depth, reference, context).
- go-git [commit and tree objects](https://github.com/go-git/go-git/tree/v5.19.2/plumbing/object),
  [memory storage](https://github.com/go-git/go-git/blob/v5.19.2/storage/memory/storage.go),
  and [transport registry](https://github.com/go-git/go-git/blob/v5.19.2/plumbing/transport/client/client.go).
