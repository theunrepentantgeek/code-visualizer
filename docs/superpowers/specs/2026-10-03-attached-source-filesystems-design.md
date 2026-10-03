# Attached Source Filesystems Design

## Problem

CodeViz has two file-content contracts:

- `scan.ScanTree` attaches an `fs.FS` to every `model.File`, allowing providers
  to read content through `File.Open` and `File.ReadAll`.
- `scan.Scan` uses a separate OS walker whose files have no attached source, so
  providers detect that case and open `File.Path` through the operating system.

The split makes providers depend on scanner provenance, duplicates traversal
behavior, and allows new providers to accidentally bypass the abstraction
required by in-memory and historical Git sources.

## Goals

- Make `scan.ScanTree` the single scanner implementation.
- Ensure files returned by both scanner entry points have a usable attached
  content source.
- Make `model.File.Open` and `model.File.ReadAll` the only provider-facing
  content-access API.
- Preserve `scan.Scan` path naming, filtering, binary handling, directory
  counts, progress callbacks, cancellation, permission handling, and empty-tree
  behavior.
- Preserve Go module discovery through attached source and repository
  filesystems for live, in-memory, and historical trees.
- Reject source-less hand-built files with a clear content-source error.

## Non-goals

- Supporting provider-level fallback reads from `File.Path`.
- Adding a new OS-path adapter for hand-built `model.File` values.
- Following symlinks to files outside the requested scan root.
- Changing CLI commands or user-facing visualization behavior.

## Architecture

### Canonical scanner

`scan.Scan` remains as the compatibility entry point for callers that provide an
OS path. It performs the early context-cancellation check, constructs a
`source.Tree` with `source.WorkingTree`, and delegates to `scan.ScanTree`.

`ScanTree` and `fsWalker` become the only traversal path. The legacy `walker`
implementation and the traversal-specific portion of `nodeBuilder` are removed.
Shared scanner helpers, including `hasFiles` and `FilterBinaryFiles`, remain.

The working-tree adapter resolves the root to an absolute path and exposes it
through `os.DirFS`. Consequently, the canonical walker continues to produce
absolute display paths while also attaching:

- `Source`: the filesystem that owns the content;
- `SourcePath`: the path accepted by that filesystem;
- `RepoSource`: the containing repository filesystem when repository context
  was supplied; and
- `RepoPath`: the corresponding repository-relative identity.

### Provider content access

Providers no longer inspect `File.Source` to choose an implementation.
Filesystem line counting, Go declaration loading, and Go metric loading call
`File.Open` or `File.ReadAll` unconditionally.

The OS-path analysis functions and caches that exist only to support source-less
files are removed. Source-backed parsing remains testable through byte-oriented
helpers such as `analyzeSource` and `analyzeDeclarationSource`.

`File.Open` retains its explicit `file has no content source` error. Provider
loaders keep their existing per-file failure policy: they log the wrapped error,
omit the unavailable metric or declarations, and continue processing other
files. They do not silently reconstruct an OS filesystem from `File.Path`.

### Go module discovery

Go module lookup starts at the file's `SourcePath` within `Source`. When a
`RepoSource` is attached, lookup instead starts at `RepoPath` within that
repository filesystem so a scoped source can find a parent `go.mod`.

There is no OS-path fallback. Callers that scan a scoped tree and need context
above its root must supply `RepoSource` and `RepoPath` through a `source.Tree`
passed to `ScanTree`. This is already how the main pipeline represents live Git
working trees and historical Git snapshots.

## Data Flow

For `scan.Scan`:

1. Reject an already-cancelled context.
2. Resolve the requested path through `source.WorkingTree`.
3. Pass the resulting tree and all existing options to `ScanTree`.
4. Traverse with `fsWalker`.
5. Build `model.Directory` and `model.File` values with attached filesystems.
6. Return the same no-files error when filtering or binary exclusion empties the
   tree.

For provider loading:

1. Walk model files.
2. Skip files irrelevant to the provider, such as binary files or non-Go files.
3. Read relevant content through `File.Open` or `File.ReadAll`.
4. Resolve repository-scoped metadata, including `go.mod`, through attached
   filesystems.
5. Populate metrics or declarations, or log a path-qualified read/parse error.

## Compatibility

The following `scan.Scan` behavior remains stable:

- root and child display paths are absolute OS paths;
- file and directory names and extensions are unchanged;
- file filters receive equivalent source-relative paths;
- binary files are classified and included or excluded according to
  `includeBinary`;
- directory symlinks are skipped;
- in-root file symlinks retain their link identity while content is read from
  the target;
- direct and recursive file/directory counts are populated;
- progress callbacks occur after each fully processed directory;
- cancellation, inaccessible entries, disappearing files, and empty results
  retain their semantic outcomes.

One behavior changes intentionally: file symlinks that resolve outside the
requested root are skipped. The attached source filesystem cannot address those
targets without weakening the requested content boundary. `Scan` documentation
will state this behavior and direct callers to select a wider root when those
files are intended inputs.

`RepoPath`, `Source`, and `SourcePath` become populated for `Scan` results. This
is additive metadata and enables the canonical content-access contract.

## Testing

### Scanner compatibility

- Verify every file returned by `Scan` has a non-nil source, a source-relative
  path, and readable content.
- Scan the same OS working tree through `Scan` and `ScanTree` and compare tree
  shape, paths, metadata, filtering, binary flags, and counts.
- Retain coverage for progress callbacks, early and mid-scan cancellation,
  permission failures, disappearing files, empty trees, and filter pruning.
- Verify in-root file symlinks remain readable under the link identity.
- Add explicit coverage that outside-root file symlinks are skipped.

### Provider contract

- Convert source-less provider fixtures to attached `fstest.MapFS` or
  `os.DirFS` fixtures.
- Verify filesystem line metrics, Go metrics, and Go declarations read attached
  content even when the display path is not an OS-readable path.
- Verify module lookup in the local source and in an attached repository source.
- Verify a missing source returns the model error and does not result in metrics
  or declarations.
- Retain working-tree, in-memory, and historical Git end-to-end coverage.

### Validation

Run focused tests for `internal/model`, `internal/scan`,
`internal/provider/filesystem`, and `internal/provider/golang`, then run the
repository CI task through the required delegated runner.

## Documentation and Migration

Update `scan.Scan` documentation to describe attached sources and the
outside-root symlink rule. Update `model.File` documentation to state that
callers constructing files manually must attach `Source` and, when different
from the display path, `SourcePath`.

Callers that currently construct source-less files must either:

- obtain files from `scan.Scan` or `scan.ScanTree`; or
- attach an appropriate `fs.FS` and `SourcePath` themselves.

No provider-level OS-path compatibility path remains.
