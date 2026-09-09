# Historical Filesystem Snapshot Design

## Goal

Make visualizations constrained by `--until` internally consistent. The files,
directories, and file contents must come from the same historical commit that
defines the upper end of Git history.

When `--until` is omitted, the current working tree remains authoritative,
including modified and untracked files.

## User-Visible Semantics

Supplying `--until` selects both:

- the inclusive upper bound for Git history; and
- the committed filesystem snapshot used by every visualization.

Tags and commit IDs identify the snapshot commit directly. A date identifies the
reachable commit with the latest author timestamp at or before the bound. A
date-only value includes the entire date by normalizing it to the final
nanosecond of that day in its interpreted timezone.

If multiple reachable commits have the same latest author timestamp, commit
hash ordering provides a stable tie-break so repeated runs select the same
snapshot.

`--from` continues to constrain Git history but does not select the filesystem
snapshot. Supplying `--from` without `--until` therefore retains the live
working tree.

This intentionally changes existing `--until` behavior. No compatibility flag
is added.

## Architecture

Introduce a read-only `source.Tree` abstraction backed by the standard
`io/fs` interfaces. It provides directory traversal and file reads while
retaining stable repository-relative file identities.

Two implementations share the contract:

- **Working tree source:** wraps the operating-system filesystem and preserves
  current behavior when `--until` is absent.
- **Git snapshot source:** exposes a selected go-git `object.Tree` directly as
  an `fs.FS`, without checking out files or creating temporary storage.

The scanner consumes a `source.Tree` instead of walking operating-system paths
directly. Files retain logical paths for labels, filters, extensions, data
exports, and Git lookups. Providers that inspect file contents read through the
source rather than calling `os.Open` or `os.ReadFile`.

Shared pipeline state carries the content source, real repository root,
repository-relative target path, and selected snapshot commit. Git providers
use the real repository and logical identities; they never attempt repository
discovery from a virtual path.

Rendering, layout, and metric aggregation remain independent of the storage
implementation.

## Git Tree Filesystem Adapter

Add a small internal adapter over go-git v5's `object.Tree`. It implements:

- `fs.FS`;
- `fs.ReadDirFS`;
- `fs.ReadFileFS`;
- `fs.StatFS`; and
- `fs.ReadLinkFS`.

The adapter maps Git directories, regular files, executable files, and
symlinks to standard `io/fs` entries. Git trees do not contain per-file
timestamps, so entry modification times use the selected commit's committer
timestamp, falling back to its author timestamp if needed.

Gitlinks are skipped. A parent commit records only a submodule commit ID, not
the referenced repository's file tree, so presenting a populated directory
would invent content that is absent from the snapshot.

The implementation is validated with `fstest.TestFS` and focused tests for
Git-specific modes and error cases. Existing external adapters are not adopted:
the closest direct adapter targets go-git v6 alpha, while go-git v5 alternatives
clone or check out repositories and do not accept an existing arbitrary commit.

## Snapshot Resolution

Snapshot resolution runs before filesystem scanning whenever `--until` is
non-empty:

1. Resolve the target path to its containing repository and repository-relative
   subdirectory.
2. Resolve a tag or SHA directly, or select the latest reachable commit within
   a date bound.
3. Load the commit and root Git tree.
4. Scope the tree to the target subdirectory.
5. Create the Git-backed content source and scan it.

Date selection considers the full commit graph reachable from `HEAD`, matching
the application's existing history traversal rather than switching to
first-parent history. The selected commit supplies the filesystem snapshot and
reference clock, but date-filtered history continues to include every reachable
commit within the date bound, including commits that are not ancestors of the
snapshot commit.

For a tag or SHA, the selected revision remains the effective history tip. An
upper revision outside the current `HEAD` ancestry remains valid, matching
existing `--until` revision behavior.

If the requested target subdirectory did not exist at the selected commit, the
command fails with a message identifying the target and snapshot.

## Scanner and Provider Data Flow

The effective pipeline is:

1. validate CLI paths and resolve configuration;
2. resolve the content source and optional snapshot;
3. scan that source using existing include/exclude and binary rules;
4. optionally apply changed-file filtering;
5. load filesystem, language, and Git metrics;
6. aggregate, lay out, and render.

Filesystem metrics and Go analysis consume the selected source's bytes. A
historical snapshot therefore changes line counts, file sizes, declarations,
imports, complexity, and other content-derived metrics along with its
directory structure.

Git metrics continue to use the selected `HistoryRange`. Repository-relative
file identities connect snapshot files to their historical records.

## Changed-Only Interaction

When `--changed-only` and `--until` are combined, filtering starts from the
historical snapshot and retains paths changed in the selected Git history
range.

The changed-path provider must intersect changes with paths in the selected
snapshot rather than paths in the current Git index. Consequently:

- files present at `--until` but deleted later remain eligible;
- files added after `--until` cannot appear;
- renamed destinations are eligible only when they exist in the snapshot and
  changed in range; and
- untracked working-tree files are irrelevant to historical snapshots.

Without `--until`, `--changed-only` retains its existing current-tree behavior.

## Reference Clock

Historical visualizations use the selected snapshot commit time as their
reference clock. Age, freshness, and authorship-window calculations are
therefore evaluated as of the snapshot rather than as of the day the command
is run.

Live-working-tree visualizations continue using the existing current or
repository clock behavior.

## Symlinks

Scanner behavior remains consistent across both source types:

- directory symlinks are skipped;
- file symlinks are followed only when their fully resolved target remains
  inside the selected source root;
- broken links, links that escape the root, and directory targets are logged
  and skipped; and
- historical resolution is cycle-checked and depth-limited.

The Git adapter exposes symlink metadata and target text through
`fs.ReadLinkFS`; the scanner owns follow-or-skip policy.

## Errors

Return clear user-facing errors for:

- using `--until` outside a Git repository;
- missing or invalid tags, commits, or dates;
- no reachable commit at or before a date bound;
- a target subtree absent from the selected commit;
- invalid or corrupt Git tree objects; and
- no files remaining after standard or changed-only filtering.

Repository and object-read failures preserve their underlying causes. Invalid
Git tree paths and corrupt objects fail the scan instead of being silently
omitted.

Inaccessible, broken, or unsafe symlinks follow existing scanner behavior:
they are logged and skipped.

The design creates no temporary files and requires no cleanup lifecycle.

## Documentation

Update usage and visualization documentation to state:

- `--until` controls both history and the filesystem snapshot;
- date-only bounds include the full named date;
- date bounds select the latest reachable matching commit;
- `--from` alone retains the live working tree;
- uncommitted and untracked files are visible only when `--until` is omitted;
  and
- historical submodules are not expanded.

## Testing

### Adapter Tests

- `fstest.TestFS` conformance;
- regular and executable files;
- nested directories and empty directories;
- binary reads;
- symlink metadata and target reads;
- submodule exclusion;
- malformed paths; and
- missing or corrupt objects.

### Resolution Tests

- tag and SHA selection;
- revision outside current `HEAD` ancestry;
- inclusive timestamp bounds;
- date-only end-of-day normalization;
- latest matching commit across merged branches;
- deterministic equal-timestamp tie-breaking;
- no matching commit; and
- missing target subtree.

### Scanner and Provider Tests

- working-tree and Git-tree sources produce equivalent models for equivalent
  content;
- safe internal file symlinks are followed;
- directory, broken, cyclic, and escaping symlinks are skipped;
- historical additions, deletions, and content changes affect structure and
  content-derived metrics together;
- include/exclude and binary filters operate on historical paths and bytes;
- Git metrics use repository-relative snapshot identities; and
- age, freshness, and authorship windows use the snapshot clock.

### Integration Tests

- historical `--changed-only` intersects the snapshot rather than the current
  index;
- target subdirectories remain correctly scoped;
- omitting `--until` preserves modified and untracked working-tree files;
- every direct visualization command forwards the source semantics; and
- preset rendering behaves identically to the corresponding direct command.

Golden tests are added only where they protect a meaningful externally visible
change that lower-level structural and metric assertions cannot cover.
