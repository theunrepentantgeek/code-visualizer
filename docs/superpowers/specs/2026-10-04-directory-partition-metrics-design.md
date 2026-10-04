# Directory Partition Metrics Design

## Status

Ready for user review before implementation planning.

## Intent

Directory expansion must preserve every file exactly once. A directory shown
as one aggregate represents its full subtree. Expanding that directory replaces
the aggregate with one aggregate per child subtree plus, when present, an
aggregate for files directly in the expanded directory.

This rule applies recursively. For example, when the repository root has direct
files, its partition contains `cmd`, `docs`, `internal`, and `.`. Expanding
`internal` replaces its full-subtree band with its package subtrees and an
`internal` direct-files band only when `internal` contains files outside those
packages.

The requirement is not specific to Alluvial rendering. The source-tree model
and metric layer must support reusable directory selections evaluated over
either a full subtree or direct files. Alluvial should consume evaluated bands
without knowing how their source files were selected.

Success means:

- a partition includes every selected file exactly once;
- direct-files remainders appear at the root and at every expanded directory
  that has direct files;
- empty direct-files remainders are omitted;
- all supported metric expressions preserve their existing semantics for both
  full-subtree and direct-files scopes;
- visualization models contain evaluated visualization data rather than
  synthetic source-tree nodes.

## Scope

This change:

- introduces a shared directory-selection abstraction with full-subtree and
  direct-files scopes;
- introduces shared recursive directory partitioning;
- extends shared metric evaluation to evaluate a directory selection on
  demand;
- refactors existing directory aggregation to use the same scoped evaluator;
- replaces Alluvial's `Snapshot.Root` and `Snapshot.DirectFiles` fields with
  evaluated path-keyed bands;
- applies root and recursive expansion remainders to Alluvial;
- updates tests and Alluvial documentation.

This change does not:

- retain empty source directories that the scanner currently prunes;
- change file filtering, provider execution, or history loading;
- require other visualizations to adopt directory partitions immediately;
- persist direct-only metrics on every `model.Directory`;
- add an `Other` label: a direct-files remainder uses the expanded directory's
  repository-relative path.

## Existing problem

`model.Directory` stores direct files and child directories, but its aggregated
metrics represent all descendant source nodes. That is correct for a collapsed
directory. It is insufficient for a partition because the direct files of an
expanded directory need a separate aggregate.

The first implementation in this PR adds a synthetic
`Snapshot.DirectFiles *model.Directory` for the target root. This has two
problems:

1. It handles only the target root, not every recursively expanded directory.
2. It represents an evaluated remainder as a second source-tree directory,
   exposing an implementation workaround in the Alluvial snapshot interface.

Using the full directory aggregate for the remainder would double-count child
files. Subtracting child aggregates from the parent works only for additive
metrics and is invalid for mean, min, max, range, distinct, mode, and similar
aggregations. Direct-only values must therefore be evaluated from their source
nodes.

## Options considered

### 1. Shared selection and eager evaluation

A shared `DirectorySelection` identifies one source directory and an evaluation
scope. A shared partition function produces selections. Alluvial acquisition
evaluates those selections immediately and stores only path-keyed metric
containers in each snapshot.

This is the selected option. It separates the source tree, structural
selection, metric evaluation, and visualization data. Other visualizations can
reuse partitioning or scoped evaluation without depending on Alluvial.

### 2. Lazy definitions in `alluvial.Snapshot`

Each snapshot could store `map[string]DirectorySelection` and defer evaluation
until `BuildData`. This keeps source-tree references and evaluation strategy in
the rendering data model. It also requires the rendering stage to receive
resolved expressions and understand provider-backed metric evaluation.

Reject this option because it moves acquisition responsibilities across the
rendering seam and makes `Snapshot` expose more than callers need.

### 3. Persist both aggregate scopes on `model.Directory`

Every directory could retain full-subtree and direct-files metric containers.
Consumers would have simple lookups, but every scan would compute and retain
values that most visualizations do not request. It also couples the source-tree
model to a particular evaluation strategy.

Reject this option in favor of evaluating `(directory, scope)` on demand.

## Shared directory selections

The source-tree model gains a structural selection type:

```go
type DirectoryScope uint8

const (
    DirectoryScopeInvalid DirectoryScope = iota
    DirectorySubtree
    DirectoryDirectFiles
)

type DirectorySelection struct {
    Directory *Directory
    Scope     DirectoryScope
}
```

The zero value is invalid. An unknown scope must return an explicit error rather
than silently behaving like either valid scope.

`DirectorySelection` owns scoped file traversal:

- `DirectorySubtree` walks `Directory.Files` and every descendant directory;
- `DirectoryDirectFiles` walks only `Directory.Files`.

Declaration- and commit-level evaluation derives from the files selected by
that traversal. This keeps the meaning of scope consistent across every metric
source level.

The abstraction belongs in `internal/model` because it describes a view over a
source tree. Metric expression evaluation remains in `internal/stages`.

## Recursive directory partitioning

A shared partition function accepts a root directory and repository-relative
expansion paths:

```go
func PartitionDirectories(
    root *Directory,
    expansions []string,
) map[string]DirectorySelection
```

The returned map is keyed by repository-relative path. Each key occurs at most
once and identifies the label/alignment identity used by a consumer.

Partitioning follows these rules:

1. The target root is always partitioned into:
   - one `DirectorySubtree` selection for each child directory; and
   - one `DirectoryDirectFiles` selection for the root when it has direct
     files.
2. Expanding a currently selected child replaces its `DirectorySubtree`
   selection with:
   - one `DirectorySubtree` selection for each child directory; and
   - one `DirectoryDirectFiles` selection for the expanded directory when it
     has direct files.
3. The replacement rule repeats for reachable nested expansion paths.
4. A direct-files selection is omitted when `len(Directory.Files) == 0`.
5. An expansion path that is not currently reachable has no effect, preserving
   existing Alluvial behavior.
6. A nil or file-empty tree produces an empty partition rather than a
   synthetic band.

The direct-files selection intentionally reuses the expanded directory's path.
It replaces the full-subtree selection at that path, so no identity collision
exists in one partition.

Although the interface returns a map, consumers that require deterministic
ordering must sort its keys. The map expresses uniqueness and supports
cross-snapshot alignment; it does not define presentation order.

## Scoped metric evaluation

The shared metric module gains an evaluator that accepts a
`DirectorySelection` and resolved expressions and returns a fresh
`*model.MetricContainer`.

The evaluator collects source values only through the selection's scoped file
traversal. It supports the same file-, declaration-, and commit-level
expressions and result kinds as existing directory aggregation. It must return
an explicit error for an invalid scope or unsupported expression.

`ComputeAggregations` is refactored to use this evaluator with
`DirectorySubtree` selections when populating real directories. This preserves
existing output while preventing separate implementations for ordinary
directory metrics and partition metrics.

Evaluation remains source-grounded. It never calculates a direct-files value by
subtracting child aggregates or by combining aggregate results.

## Alluvial acquisition and snapshot model

After scanning a historical reference, Alluvial continues to load providers,
populate declarations, and prepare commit data before selecting bands. It then:

1. calls `PartitionDirectories(snapshotRoot, cfg.Expand)`;
2. evaluates every returned selection using the requested resolved
   expressions;
3. stores each evaluated metric container under its path.

The snapshot model becomes:

```go
type Snapshot struct {
    Reference string
    Bands     map[string]*model.MetricContainer
}
```

`Snapshot` no longer contains `Root` or `DirectFiles`. Exporting the first
snapshot continues to use the acquisition state's real root and is independent
of the Alluvial snapshot model.

`BuildData` receives already evaluated bands. It sorts paths when constructing
columns, looks up width and fill metrics from each metric container, and keeps
the existing transition, temporal fill, constant-band, layout, and rendering
logic. `Options.Expand` is removed because expansion is complete before data
construction.

## Missing and empty references

A target may be absent or file-empty at an individual historical reference.
Acquisition represents that reference with an empty source root, partitioning
produces no bands, and transition construction tapers paths to or from zero.

The command returns an error only when every reference has an empty band map.
No direct-files band is created merely to carry a zero metric value. A
non-empty selection whose chosen metric evaluates to zero remains a valid band,
preserving existing zero-width behavior.

Shared scanner behavior remains unchanged. The Alluvial acquisition path owns
its existing tolerance for missing or empty historical targets.

## Verification

### Shared partition tests

- repository root with direct files and child directories;
- root without direct files;
- expansion with both direct files and children;
- expansion without direct files;
- recursively nested expansions with remainders at multiple levels;
- unreachable expansion paths;
- unique repository-relative keys and complete file partitioning;
- nil and file-empty roots.

### Shared evaluator tests

- sum and count over direct files versus a full subtree;
- non-additive mean, min/max, range, distinct, and mode;
- declaration-level values from direct files only;
- commit-level values from direct files only;
- invalid scope errors;
- unchanged `ComputeAggregations` full-subtree behavior.

### Alluvial tests

- snapshot construction contains evaluated bands and no source-tree
  directories;
- repository-root direct-files band;
- leaf target as one direct-files band;
- expanded directory with a direct-files remainder;
- expanded directory without direct files;
- nested expansion remainders;
- introductions and removals across references;
- all references empty;
- width, fill, temporal fill, and constant-band behavior over evaluated bands;
- end-to-end SVG labels for root and nested expansion cases.

Documentation examples will state that expansion partitions a directory into
child subtrees and an optional direct-files remainder labelled with the
directory's repository-relative path.

## Migration of the current PR

The implementation plan will replace, rather than layer on top of, the current
`DirectFiles` workaround:

- remove `Snapshot.Root` and `Snapshot.DirectFiles`;
- remove `directFilesAggregate`;
- remove direct-files special cases from `selectedDirectories`;
- add shared partitioning and scoped evaluation;
- move expansion from `BuildData` into snapshot acquisition;
- rewrite the current regression tests against the new snapshot interface and
  add recursive expansion coverage.

This keeps the PR focused on issue #767 while correcting the abstraction before
merge.
