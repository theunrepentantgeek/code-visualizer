# Abstractions — `internal/provider/git`

## Commit

**Purpose.**

- `Commit` is a single commit in the project history with enough metadata for any downstream consumer — timeline, churn, authorship, message-mining — rather than a per-metric subset ([commit.go#L21](commit.go#L21), [commit.go#L29](commit.go#L29)).
- `Signature` is the author/committer record captured at commit time ([commit.go#L13](commit.go#L13), [commit.go#L15](commit.go#L15)).

**Boundary and invariants.**

- Immutable after loading: once `BulkCommitHistory` returns, no field of any returned commit is mutated, so consumers may hold `*Commit` references for the lifetime of the slice ([commit.go#L26](commit.go#L26), [../../stages/common.go#L59](../../stages/common.go#L59)).
- `Changes` contains `FileChange` records with slash-separated, repository-relative paths and added/removed line counts, restricted to the tracked path set. It is not a list of every repository change ([commit.go#L30](commit.go#L30), [commit.go#L40](commit.go#L40)).
- This is the acquisition-side record; the per-file commit attached to the model tree is a separate, smaller type ([../../model/commit.go#L6](../../model/commit.go#L6)).

**Related operations.**

- `BulkCommitHistory` and its ranged/prewarm variants acquire shared history; `CommitTotal` and `CommitTotalInHistoryRange` count commits, with context-aware ranged entry points available for cancellation ([bulk_history.go](bulk_history.go), [commit.go](commit.go)).

**Proper-use patterns.**

- Load history once into `CommonState.GitHistory` and derive per-file views from it, instead of querying the repository per metric ([../../stages/git_history.go#L28](../../stages/git_history.go#L28), [../../stages/providers.go#L40](../../stages/providers.go#L40)).

**Anti-patterns.**

- Do not mutate loaded commits or their slices; the no-mutation invariant is what makes shared `*Commit` references safe ([commit.go#L26](commit.go#L26)).

**Source locations.**

- [commit.go#L15](commit.go#L15) — `Signature`.
- [commit.go#L29](commit.go#L29) — `Commit` and its invariant.
- [../../stages/git_history.go](../../stages/git_history.go) and [../../spiral/bucketing.go](../../spiral/bucketing.go) — per-file history and timeline consumers.

## HistoryRange

**Purpose.**

- `HistoryRange` is the user's selection of which commits count: a `From` lower bound and an `Until` upper bound, each given as `HEAD`, a tag, commit ID, date, `^` predecessor, or `ago(Nd)` relative date ([history_range.go#L17](history_range.go#L17), [../../../cmd/codeviz/treemap_cmd.go#L17](../../../cmd/codeviz/treemap_cmd.go#L17)).

**Boundary and invariants.**

- The zero value selects history reachable from HEAD. A revision lower bound excludes that revision and its ancestors; an upper revision chooses the effective tip. Date bounds filter author timestamps, not committer timestamps ([history_range.go](history_range.go)).
- Resolution to concrete graph positions and timestamps is internal; callers pass reference strings, including explicit `tag:`, `sha:`, or `date:` forms when disambiguation is needed ([history_reference.go](history_reference.go)).
- `TimeNow` supplies the shared clock for `ago(Nd)` bounds. CLI construction captures it once for the run; a zero clock is filled by each range-resolution call, so independently resolved zero-clock ranges need not share an instant ([history_range.go#L26](history_range.go#L26), [../../../cmd/codeviz/main.go](../../../cmd/codeviz/main.go)).
- It travels with the run as part of the shared flag bundle rather than being re-parsed per metric ([../../stages/common.go#L29](../../stages/common.go#L29)).

**Related operations.**

- The `*InHistoryRange` family — commit totals, bulk history, author history, changed paths — all take one ([commit.go#L44](commit.go#L44), [author_history.go#L87](author_history.go#L87), [../../stages/changed_only.go#L34](../../stages/changed_only.go#L34)).

**Proper-use patterns.**

- Build the range once from the command's `--from`/`--until` flags and pass it down, so every git query in the run sees the same window ([../../stages/common.go#L29](../../stages/common.go#L29)).

**Anti-patterns.**

- Do not add a second range-like parameter to git APIs; the ranged variants already carry both bounds ([commit.go#L94](commit.go#L94)).

**Source locations.**

- [history_range.go#L18](history_range.go#L18) — `HistoryRange`.
- [history_reference.go](history_reference.go), [../../stages/changed_only.go](../../stages/changed_only.go), and [../../stages/source.go](../../stages/source.go) — reference syntax, change filtering, and source selection.

## Snapshot

**Purpose.**

- `Snapshot` identifies the committed tree selected by an `--until` value: the repository root, the chosen commit, and its tree ([snapshot.go#L15](snapshot.go#L15), [snapshot.go#L16](snapshot.go#L16)).
- It is what lets the tool visualize the codebase as it was at a point in history instead of the working tree ([../../stages/source.go#L64](../../stages/source.go#L64)).

**Boundary and invariants.**

- Resolving requires a git repository; without one the `--until` request is an error, not a silent fallback ([../../stages/source.go#L61](../../stages/source.go#L61), [snapshot.go#L23](snapshot.go#L23)).
- `Subtree` scopes the snapshot to the repo-relative directory being visualized, so a snapshot of a subdirectory is still a real git tree ([snapshot.go#L115](snapshot.go#L115), [../../stages/source.go#L69](../../stages/source.go#L69)).
- A date reference selects the latest reachable commit by author time at or before the cutoff. After selection, the commit's committer time (falling back to author time) becomes the source and metric reference clock; selection time and evaluation time are distinct ([snapshot.go](snapshot.go), [../../stages/source.go](../../stages/source.go)).

**Related operations.**

- `ResolveSnapshot` to build one; `Subtree` to scope it; `source.NewGitFS` to expose it as an `fs.FS` ([snapshot.go#L23](snapshot.go#L23), [snapshot.go#L115](snapshot.go#L115), [../../source/gitfs.go#L39](../../source/gitfs.go#L39)).

**Proper-use patterns.**

- Substitute the snapshot for the working tree at the source-resolution stage, so downstream scanning and metric code is unaware of the difference ([../../stages/source.go#L79](../../stages/source.go#L79)).

**Anti-patterns.**

- Do not read snapshot content through OS paths; the snapshot only exists as git objects, reached through the `fs.FS` built from its tree ([../../stages/source.go#L79](../../stages/source.go#L79)).

**Source locations.**

- [snapshot.go#L16](snapshot.go#L16) — `Snapshot`, `ResolveSnapshot`, `Subtree`.
- [../../stages/source.go](../../stages/source.go) and [../../alluvial/pipeline.go](../../alluvial/pipeline.go) — single-source substitution and ordered-snapshot acquisition.

## AuthorshipParams

**Purpose.**

- `AuthorshipParams` bundles the configurable thresholds shared by the authorship metric family — activity and recency windows, early-window fraction, significance and bus-factor thresholds, identity top-k, and mailmap handling ([authorship_metrics.go#L21](authorship_metrics.go#L21), [authorship_metrics.go#L24](authorship_metrics.go#L24)).

**Boundary and invariants.**

- `DefaultAuthorshipParams` supplies the baseline; the stages layer overlays only configured fields before calling the Git metric loader ([authorship_metrics.go](authorship_metrics.go), [../../stages/providers.go#L354](../../stages/providers.go#L354)).
- Identity metrics return sentinel classifications rather than empty strings: `Unmaintained` when no qualifying author exists, `OtherContributor` for contributors beyond top-k ([authorship_metrics.go#L10](authorship_metrics.go#L10), [authorship_metrics.go#L14](authorship_metrics.go#L14)).
- `IdentityTopK` exists to bound legend size, not to filter data ([authorship_metrics.go#L42](authorship_metrics.go#L42)).

**Related operations.**

- `DefaultAuthorshipParams` for the baseline; the stages layer merges the user's `config.AuthorshipConfig` over it ([authorship_metrics.go#L53](authorship_metrics.go#L53), [../../stages/providers.go#L174](../../stages/providers.go#L174)).

**Proper-use patterns.**

- Start from the defaults and apply only values the user set, then pass the result to `LoadAuthorshipMetricsInHistoryRange` so one run's settings do not become mutable global provider state ([../../stages/providers.go#L354](../../stages/providers.go#L354), [authorship_loader.go#L32](authorship_loader.go#L32)).

**Anti-patterns.**

- Do not use a zero-valued `AuthorshipParams` as the default or overwrite omitted options with zero; neither preserves the metric family's configured thresholds ([authorship_metrics.go](authorship_metrics.go), [../../stages/providers.go#L354](../../stages/providers.go#L354)).

**Source locations.**

- [authorship_metrics.go#L10](authorship_metrics.go#L10) — `Unmaintained`, `OtherContributor`.
- [authorship_metrics.go#L24](authorship_metrics.go#L24) — `AuthorshipParams`, `DefaultAuthorshipParams`.
- [../../stages/providers.go#L354](../../stages/providers.go#L354) and [authorship_loader.go](authorship_loader.go) — config translation and parameterized loading.
