# Abstractions — `internal/model`

The model package is the shared vocabulary for the scanned codebase: a tree of directories and files, the declarations and commits attached to files, and the metric values attached to all four ([file.go#L1](file.go#L1)).

## Directory

**Purpose.**

- `Directory` is an interior node of the scanned tree: its children (`Dirs`, `Files`), its identity under several roots (`Path`, `RepoPath`, `RepoRoot`, `Name`), and its own metric values ([directory.go#L9](directory.go#L9)).

**Boundary and invariants.**

- `Source` is the `fs.FS` the subtree was read from, and `ReferenceTime` is the run's clock, so time-based metrics do not each call `time.Now()` ([directory.go#L15](directory.go#L15), [directory.go#L16](directory.go#L16)).
- The count fields (`DirectFileCount`, `AllFileCount`, `AllDirCount`) are populated by the scan and are documented as zero on hand-built trees — treat them as a scan artefact, not an invariant ([directory.go#L22](directory.go#L22), [directory.go#L26](directory.go#L26)).
- The root `*Directory` is the single object passed between pipeline stages; providers mutate it in place rather than rebuilding it ([../stages/common.go#L49](../stages/common.go#L49), [../provider/filesystem/metrics.go#L59](../provider/filesystem/metrics.go#L59)).

**Related operations.**

- `WalkFiles` visits direct files before descending and tolerates a nil root or nil child directories; `WalkDeclarations` and `WalkCommits` use that traversal. `WalkDirectories` is post-order (children before parents) and requires non-nil directory nodes, as do `CountFiles`/`CountDirs` ([walk.go](walk.go)).
- `PruneLayers` returns a depth-limited copy of directory nodes for a positive layer budget, cloning metric maps but sharing file objects. A non-positive budget returns the original tree ([walk.go#L44](walk.go#L44), [../donuttree/stages.go](../donuttree/stages.go)).

**Proper-use patterns.**

- Collect values by walking from the root with `WalkFiles` rather than hand-rolling recursion over `Dirs`/`Files` ([../provider/git/metrics.go#L105](../provider/git/metrics.go#L105), [../scatter/data.go#L120](../scatter/data.go#L120)).
- Use `RepoPath` (not `Path`) when talking to git, because it is the repository-relative name ([../provider/git/loader.go#L222](../provider/git/loader.go#L222), [../stages/git_history.go#L220](../stages/git_history.go#L220)).

**Anti-patterns.**

- Do not prune the shared root in place when limiting depth; `PruneLayers` deliberately builds new nodes so other stages keep the full tree ([walk.go#L57](walk.go#L57)).
- Do not treat the pruned result as a fully independent acquisition tree: it shares files and copies only selected directory fields, not repository/source context ([walk.go#L57](walk.go#L57)).

**Source locations.**

- [directory.go#L9](directory.go#L9) — `Directory`.
- [walk.go#L4](walk.go#L4) — traversal and counting helpers.
- [../scatter/data.go](../scatter/data.go) and [../provider/git/metrics.go](../provider/git/metrics.go) — independent model consumers.

## File

**Purpose.**

- `File` is a leaf of the scanned tree: naming (`Path`, `RepoPath`, `SourcePath`, `Name`, `Extension`), content access, the per-file metric values, and the `Declarations` and `Commits` discovered for it ([file.go#L12](file.go#L12)).

**Boundary and invariants.**

- Content is reached through the attached `fs.FS`, never through the OS path: `Open`/`ReadAll` fail explicitly when no source is attached ([file.go#L27](file.go#L27), [file.go#L46](file.go#L46)).
- `Source` and `RepoSource` are separate on purpose — one is the tree being visualized, the other the repository content it came from ([file.go#L20](file.go#L20), [file.go#L21](file.go#L21)).
- `IsBinary` is decided once during the scan and is the flag downstream filtering keys off ([../scan/node_builder.go#L52](../scan/node_builder.go#L52), [../provider/filesystem/metrics.go#L64](../provider/filesystem/metrics.go#L64)).

**Related operations.**

- `Open` and `ReadAll` for content; `MetricContainer` accessors for values; `WalkFiles` for traversal ([file.go#L27](file.go#L27), [walk.go#L4](walk.go#L4)).

**Proper-use patterns.**

- Read file content with `ReadAll()` so the same code works against a working tree and against a git snapshot ([file.go#L46](file.go#L46), [../source/gitfs.go#L39](../source/gitfs.go#L39)).

**Anti-patterns.**

- Do not re-detect binary content per provider; the scan already recorded it on the file ([../scan/node_builder.go#L78](../scan/node_builder.go#L78)).

**Source locations.**

- [file.go#L12](file.go#L12) — `File`, `Open`, `ReadAll`.
- [../provider/filesystem/metrics.go](../provider/filesystem/metrics.go) and [../provider/golang/file_loader.go](../provider/golang/file_loader.go) — content/metric consumers.

## Declaration

**Purpose.**

- `Declaration` is a named thing inside a source file — a function, method, type, constant, or variable — with a visibility and its own metric values ([declaration.go#L17](declaration.go#L17), [declaration.go#L6](declaration.go#L6)).

**Boundary and invariants.**

- `Kind` and `Visibility` are open strings, but the recognised kind vocabulary is declared as constants here, and visibility is "public" or "private" ([declaration.go#L6](declaration.go#L6), [declaration.go#L25](declaration.go#L25)).
- `MatchesFilter` is the declaration-level implementation of `metric.FilterName`; an unrecognised filter matches nothing rather than everything ([declaration.go#L25](declaration.go#L25), [declaration.go#L31](declaration.go#L31)).
- Declarations are populated by a language provider and are replaced wholesale for a file, not merged ([../provider/golang/declarations.go#L51](../provider/golang/declarations.go#L51)).

**Related operations.**

- `WalkDeclarations` visits each declaration with its owning file ([walk.go#L125](walk.go#L125)).

**Proper-use patterns.**

- Use the `DeclKind*` constants when emitting declarations from a language provider instead of ad-hoc strings ([declaration.go#L6](declaration.go#L6)).

**Anti-patterns.**

- Do not test `Visibility` directly for filtered metrics; go through `MatchesFilter` so the filter vocabulary stays in one place ([declaration.go#L25](declaration.go#L25)).

**Source locations.**

- [declaration.go#L17](declaration.go#L17) — `Declaration`, kind constants, `MatchesFilter`.
- [../provider/golang/declarations.go](../provider/golang/declarations.go) and [../stages/aggregation.go](../stages/aggregation.go) — production and aggregation of declarations.

## Commit

**Purpose.**

- `Commit` is a git commit *as attached to a file* — hash, author, date, plus its own metric values ([commit.go#L6](commit.go#L6)).

**Boundary and invariants.**

- It is deliberately minimal: the richer repository-side commit record, with signatures and changed paths, lives in the git provider ([commit.go#L6](commit.go#L6), [../provider/git/commit.go#L29](../provider/git/commit.go#L29)).
- The same commit appears once per file it touched, which is what makes per-file commit metrics possible ([walk.go#L134](walk.go#L134)).

**Related operations.**

- `WalkCommits` visits each commit with its owning file; `CountCommits` totals them ([walk.go#L134](walk.go#L134), [walk.go#L114](walk.go#L114)).

**Proper-use patterns.**

- Treat `File.Commits` as the model-level view for commit-level metrics, and the git provider's history types as the acquisition-side detail ([commit.go#L6](commit.go#L6), [../provider/git/history_range.go#L18](../provider/git/history_range.go#L18)).

**Anti-patterns.**

- Do not derive repository-wide statistics by walking `File.Commits`: commits are duplicated across files by construction ([walk.go#L134](walk.go#L134)).

**Source locations.**

- [commit.go#L6](commit.go#L6) — `Commit`.
- [../stages/git_history.go](../stages/git_history.go) and [../stages/aggregation.go](../stages/aggregation.go) — attachment and commit-level aggregation.

## MetricContainer

**Purpose.**

- `MetricContainer` is the one place metric values are stored: three name-keyed maps, one per `metric.Kind`, guarded by a mutex, embedded by `Directory`, `File`, `Declaration` and `Commit` ([metric_container.go#L10](metric_container.go#L10), [metric_container.go#L12](metric_container.go#L12)).

**Boundary and invariants.**

- Every getter returns `(value, ok)` so "absent" is distinguishable from "zero"; nil maps read as absent and are created lazily on write ([metric_container.go#L20](metric_container.go#L20), [metric_container.go#L89](metric_container.go#L89)).
- Kind is part of the key space, not the value: a name stored as a quantity is not visible through `Measure` ([metric_container.go#L14](metric_container.go#L14)).
- Access is thread-safe by design, because providers populate metrics concurrently ([metric_container.go#L10](metric_container.go#L10), [metric_container.go#L21](metric_container.go#L21)).

**Related operations.**

- `Quantity`/`Measure`/`Classification` and their `Set*` counterparts; `Clone` deep-copies the maps for derived trees ([metric_container.go#L20](metric_container.go#L20), [metric_container.go#L58](metric_container.go#L58)).

**Proper-use patterns.**

- Store an aggregated value under `ResolvedMetric.ResultName` and with its `ResultKind`; the resolved name omits temporal modifiers because this container stores per-snapshot values, not cross-snapshot comparisons ([../provider/resolution.go](../provider/resolution.go), [../stages/aggregation.go](../stages/aggregation.go)).
- Check the `ok` result and record the node as skipped rather than treating a missing metric as zero ([../scatter/data.go#L136](../scatter/data.go#L136), [../scatter/data.go#L142](../scatter/data.go#L142)).

**Anti-patterns.**

- Do not add another metric map to `File` or `Directory`; embedding this container is what keeps storage and locking uniform ([metric_container.go#L10](metric_container.go#L10)).

**Source locations.**

- [metric_container.go#L12](metric_container.go#L12) — `MetricContainer` and its accessors.
- [../stages/aggregation.go](../stages/aggregation.go) and [../inks/inks.go](../inks/inks.go) — computed-value storage and colour lookup.

## DirectorySelection

**Purpose.**

- `DirectorySelection` couples a directory with an explicit file scope so aggregation can distinguish a whole subtree from only the directory's direct files ([directory_selection.go](directory_selection.go)).

**Boundary and invariants.**

- `DirectorySubtree` includes descendants; `DirectoryDirectFiles` excludes them. The zero scope is invalid, and `WalkFiles` returns an error for it or a nil directory ([directory_selection.go#L22](directory_selection.go#L22)).
- `PartitionDirectories` returns disjoint selections keyed by repository-relative directory path. Expansions are relative to the selected root and only take effect when their ancestors are expanded. Direct-file selections with no files are omitted ([directory_selection.go#L41](directory_selection.go#L41), [directory_selection_test.go](directory_selection_test.go)).

**Related operations.**

- `WalkFiles` evaluates the scope; `PartitionDirectories` forms the selections without copying the model; `stages.EvaluateAggregations` returns scoped metric values separately from the directory's stored subtree metrics ([../stages/aggregation.go#L35](../stages/aggregation.go#L35)).

**Proper-use patterns.**

- Shared directory aggregation explicitly chooses `DirectorySubtree`; alluvial acquisition evaluates the partition's supplied scope for each band, including direct-file remainder bands ([../stages/aggregation.go#L92](../stages/aggregation.go#L92), [../alluvial/pipeline.go#L276](../alluvial/pipeline.go#L276)).

**Anti-patterns.**

- Do not substitute the directory's existing roll-up value for a direct-file selection: it includes descendants and would count files in more than one expanded band ([../alluvial/pipeline.go#L281](../alluvial/pipeline.go#L281), [directory_selection_test.go](directory_selection_test.go)).

**Source locations.**

- [directory_selection.go](directory_selection.go) — scopes, selections, and partitioning.
- [../stages/aggregation.go](../stages/aggregation.go) and [../alluvial/pipeline.go](../alluvial/pipeline.go) — independent aggregation and partition consumers.
