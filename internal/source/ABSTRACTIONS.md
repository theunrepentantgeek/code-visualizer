# Abstractions — `internal/source`

## Tree

**Purpose.**

- A `Tree` is the read-only content source a scan walks, together with the repository context needed to name what it finds: the scanned filesystem, the enclosing repository filesystem, the root name/path, the repository root and base, and the reference clock ([tree.go#L15](tree.go#L15), [tree.go#L1](tree.go#L1)).
- It is the seam that makes "scan the working directory" and "scan a historical commit" the same operation ([../stages/source.go#L16](../stages/source.go#L16), [../stages/source.go#L79](../stages/source.go#L79)).

**Boundary and invariants.**

- Paths inside a `Tree` are `fs.FS` names (slash-separated, relative to `FS`); `RepoPath` rebases such a name onto `RepoBase` to produce a repository-relative path, treating an empty or `"."` base as "already repo-relative" ([tree.go#L39](tree.go#L39)).
- A tree is populated in stages: `WorkingTree` fills only `FS`, `RootName` and `RootPath`, while the repository fields and `Clock` are set later — `Clock` to the snapshot commit time when scanning history, so age calculations are relative to the snapshot rather than "now" ([tree.go#L22](tree.go#L22), [tree.go#L26](tree.go#L26), [../stages/source.go#L74](../stages/source.go#L74)).
- The tree is read-only: `NewGitFS` exposes a committed `object.Tree` as an `fs.FS` whose file modes and modification times come from the commit, never from disk ([gitfs.go#L39](gitfs.go#L39), [gitfs.go#L228](gitfs.go#L228)).

**Related operations.**

- `WorkingTree` builds an OS-backed tree; `NewGitFS` builds the historical `fs.FS` substituted into `FS`/`RepoFS` for `--until` ([tree.go#L26](tree.go#L26), [gitfs.go#L39](gitfs.go#L39), [../stages/source.go#L79](../stages/source.go#L79)).
- `RepoPath` is the only supported way to derive repository-relative names ([tree.go#L39](tree.go#L39)).

**Proper-use patterns.**

- Accept a `source.Tree` (not a path) in scanning code, so the same walker serves live and historical scans ([../scan/scanner.go#L24](../scan/scanner.go#L24), [../scan/fs_walker.go#L32](../scan/fs_walker.go#L32)).
- Populate `model` nodes' `RepoPath` via `tree.RepoPath(name)` while walking ([../scan/fs_walker.go#L44](../scan/fs_walker.go#L44), [../scan/fs_walker.go#L147](../scan/fs_walker.go#L147)).
- Switch to history by replacing `FS`/`RepoFS`/`Clock` on an already-resolved tree rather than constructing a different kind of source ([../stages/source.go#L79](../stages/source.go#L79)).

**Anti-patterns.**

- Do not reach past the tree to the OS (`os.Open`, `filepath.Walk`) while scanning: a historical tree has no on-disk representation ([gitfs.go#L39](gitfs.go#L39)).
- Do not join `RepoBase` onto names by hand; `RepoPath` already handles the empty and `"."` base cases ([tree.go#L41](tree.go#L41)).
- Do not use wall-clock time for age-like values when a tree was resolved from a snapshot — use the tree's `Clock` ([tree.go#L22](tree.go#L22), [../stages/source.go#L81](../stages/source.go#L81)).

**Source locations.**

- [tree.go#L15](tree.go#L15) — `Tree`.
- [tree.go#L26](tree.go#L26) — `WorkingTree`; [tree.go#L39](tree.go#L39) — `RepoPath`.
- [gitfs.go#L39](gitfs.go#L39) — `NewGitFS`, the committed-tree `fs.FS`.
