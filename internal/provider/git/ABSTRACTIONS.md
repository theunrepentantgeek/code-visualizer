# Abstractions — `internal/provider/git`

## Commit

**Purpose.**

- `Commit` is a single commit in the project history with enough metadata for any downstream consumer — timeline, churn, authorship, message-mining — rather than a per-metric subset ([commit.go#L21](commit.go#L21), [commit.go#L29](commit.go#L29)).
- `Signature` is the author/committer record captured at commit time ([commit.go#L13](commit.go#L13), [commit.go#L15](commit.go#L15)).

**Boundary and invariants.**

- Immutable after loading: once `BulkCommitHistory` returns, no field of any returned commit is mutated, so consumers may hold `*Commit` references for the lifetime of the slice ([commit.go#L26](commit.go#L26), [../../stages/common.go#L59](../../stages/common.go#L59)).
- `ChangedPaths` is slash-separated and repo-relative, and is restricted to the tracked path set passed in, which keeps each commit's memory bounded ([commit.go#L23](commit.go#L23), [commit.go#L34](commit.go#L34)).
- This is the acquisition-side record; the per-file commit attached to the model tree is a separate, smaller type ([../../model/commit.go#L6](../../model/commit.go#L6)).

**Related operations.**

- `BulkCommitHistory` and its range/prewarm variants walk the graph once for a whole run; `CommitTotal` counts reachable commits ([commit.go#L85](commit.go#L85), [commit.go#L120](commit.go#L120), [commit.go#L39](commit.go#L39)).

**Proper-use patterns.**

- Load history once into `CommonState.GitHistory` and derive per-file views from it, instead of querying the repository per metric ([../../stages/git_history.go#L28](../../stages/git_history.go#L28), [../../stages/providers.go#L40](../../stages/providers.go#L40)).

**Anti-patterns.**

- Do not mutate loaded commits or their slices; the no-mutation invariant is what makes shared `*Commit` references safe ([commit.go#L26](commit.go#L26)).

**Source locations.**

- [commit.go#L15](commit.go#L15) — `Signature`.
- [commit.go#L29](commit.go#L29) — `Commit` and its invariant.

## HistoryRange

**Purpose.**

- `HistoryRange` is the user's selection of which commits count: a `From` lower bound and an `Until` upper bound, each given as a tag, commit id, or date ([history_range.go#L17](history_range.go#L17), [../../../cmd/codeviz/treemap_cmd.go#L17](../../../cmd/codeviz/treemap_cmd.go#L17)).

**Boundary and invariants.**

- The zero value means "all history", and every unranged entry point delegates to the ranged one with an empty range ([commit.go#L39](commit.go#L39), [author_history.go#L69](author_history.go#L69)).
- Resolution to concrete graph positions and timestamps is internal; callers pass the user-facing strings and never plumbing hashes ([history_range.go#L23](history_range.go#L23), [history_range.go#L30](history_range.go#L30)).
- It travels with the run as part of the shared flag bundle rather than being re-parsed per metric ([../../stages/common.go#L29](../../stages/common.go#L29)).

**Related operations.**

- The `*InHistoryRange` family — commit totals, bulk history, author history, changed paths — all take one ([commit.go#L44](commit.go#L44), [author_history.go#L87](author_history.go#L87), [../../stages/changed_only.go#L34](../../stages/changed_only.go#L34)).

**Proper-use patterns.**

- Build the range once from the command's `--from`/`--until` flags and pass it down, so every git query in the run sees the same window ([../../stages/common.go#L29](../../stages/common.go#L29)).

**Anti-patterns.**

- Do not add a second range-like parameter to git APIs; the ranged variants already carry both bounds ([commit.go#L94](commit.go#L94)).

**Source locations.**

- [history_range.go#L18](history_range.go#L18) — `HistoryRange`.

## Snapshot

**Purpose.**

- `Snapshot` identifies the committed tree selected by an `--until` value: the repository root, the chosen commit, and its tree ([snapshot.go#L15](snapshot.go#L15), [snapshot.go#L16](snapshot.go#L16)).
- It is what lets the tool visualize the codebase as it was at a point in history instead of the working tree ([../../stages/source.go#L64](../../stages/source.go#L64)).

**Boundary and invariants.**

- Resolving requires a git repository; without one the `--until` request is an error, not a silent fallback ([../../stages/source.go#L61](../../stages/source.go#L61), [snapshot.go#L23](snapshot.go#L23)).
- `Subtree` scopes the snapshot to the repo-relative directory being visualized, so a snapshot of a subdirectory is still a real git tree ([snapshot.go#L115](snapshot.go#L115), [../../stages/source.go#L69](../../stages/source.go#L69)).
- The snapshot commit's date becomes the run's reference clock, so time-based metrics are computed relative to the snapshot rather than to now ([../../stages/source.go#L74](../../stages/source.go#L74), [../../stages/source.go#L83](../../stages/source.go#L83)).

**Related operations.**

- `ResolveSnapshot` to build one; `Subtree` to scope it; `source.NewGitFS` to expose it as an `fs.FS` ([snapshot.go#L23](snapshot.go#L23), [snapshot.go#L115](snapshot.go#L115), [../../source/gitfs.go#L39](../../source/gitfs.go#L39)).

**Proper-use patterns.**

- Substitute the snapshot for the working tree at the source-resolution stage, so downstream scanning and metric code is unaware of the difference ([../../stages/source.go#L79](../../stages/source.go#L79)).

**Anti-patterns.**

- Do not read snapshot content through OS paths; the snapshot only exists as git objects, reached through the `fs.FS` built from its tree ([../../stages/source.go#L79](../../stages/source.go#L79)).

**Source locations.**

- [snapshot.go#L16](snapshot.go#L16) — `Snapshot`, `ResolveSnapshot`, `Subtree`.

## AuthorRecord

**Purpose.**

- `AuthorRecord` is one author's aggregate contribution to a single file, plus the per-commit `ContributionPoint` list that window-based metrics filter by calendar time ([author_history.go#L20](author_history.go#L20), [author_history.go#L23](author_history.go#L23)).

**Boundary and invariants.**

- Contribution weight is `Added + Removed`: removals count equally, so the model rewards improving or deleting code, not just adding volume ([author_history.go#L21](author_history.go#L21), [author_history.go#L58](author_history.go#L58)).
- Records are per file *and* per author; `FirstSeen`/`LastSeen` bound that author's activity on that file ([author_history.go#L28](author_history.go#L28), [author_history.go#L29](author_history.go#L29)).
- `Contributions` is retained deliberately — aggregate totals alone cannot answer initial-developer, current-maintainer, orphan-risk, or knowledge-handoff questions ([author_history.go#L10](author_history.go#L10), [author_history.go#L30](author_history.go#L30)).

**Related operations.**

- `BulkAuthorHistory` produces them; author identity can be normalised through the repository `.mailmap` before aggregation ([author_history.go#L69](author_history.go#L69), [authorship_metrics.go#L46](authorship_metrics.go#L46)).

**Proper-use patterns.**

- Derive shares and rankings from the records' weights rather than from raw line counts ([authorship_metrics.go#L76](authorship_metrics.go#L76)).

**Anti-patterns.**

- Do not drop `Contributions` to save memory; the windowed metrics need per-commit timestamps ([authorship_metrics.go#L152](authorship_metrics.go#L152)).

**Source locations.**

- [author_history.go#L14](author_history.go#L14) — `ContributionPoint`.
- [author_history.go#L23](author_history.go#L23) — `AuthorRecord`, `FileAuthorRecords`.

## AuthorHistoryResult

**Purpose.**

- `AuthorHistoryResult` is the whole-repository authorship picture produced by one graph walk: per-file records, a repo-wide author last-active map, and the HEAD commit date ([author_history.go#L42](author_history.go#L42), [author_history.go#L43](author_history.go#L43)).

**Boundary and invariants.**

- `HeadDate` is the global clock for every authorship window calculation, so results do not drift with wall-clock time ([author_history.go#L48](author_history.go#L48), [../../stages/common.go#L66](../../stages/common.go#L66)).
- `LastActive` covers all authors in the repository, not only those touching tracked files, because orphan-risk asks whether an author is still active anywhere ([author_history.go#L48](author_history.go#L48), [author_history.go#L54](author_history.go#L54)).
- It is produced by a single walk over the commit graph and then treated as read-only run state ([author_history.go#L55](author_history.go#L55), [../../stages/author_history.go#L40](../../stages/author_history.go#L40)).

**Related operations.**

- `BulkAuthorHistory` / `BulkAuthorHistoryInHistoryRange` build it; the authorship stage stores it on the shared state ([author_history.go#L69](author_history.go#L69), [../../stages/author_history.go#L15](../../stages/author_history.go#L15)).

**Proper-use patterns.**

- Compute all nine authorship metrics from one stored result instead of re-walking history per metric ([../../stages/author_history.go#L15](../../stages/author_history.go#L15)).

**Anti-patterns.**

- Do not use `time.Now()` for authorship windows; use `HeadDate` so output is reproducible ([author_history.go#L48](author_history.go#L48), [authorship_metrics.go#L244](authorship_metrics.go#L244)).

**Source locations.**

- [author_history.go#L43](author_history.go#L43) — `AuthorHistoryResult`.

## AuthorshipParams

**Purpose.**

- `AuthorshipParams` bundles the configurable thresholds shared by the authorship metric family — activity and recency windows, early-window fraction, significance and bus-factor thresholds, identity top-k, and mailmap handling ([authorship_metrics.go#L21](authorship_metrics.go#L21), [authorship_metrics.go#L24](authorship_metrics.go#L24)).

**Boundary and invariants.**

- Every field has a specified default supplied by `DefaultAuthorshipParams`; config only overrides individual values ([authorship_metrics.go#L53](authorship_metrics.go#L53), [../../stages/providers.go#L173](../../stages/providers.go#L173)).
- Identity metrics return sentinel classifications rather than empty strings: `Unmaintained` when no qualifying author exists, `OtherContributor` for contributors beyond top-k ([authorship_metrics.go#L10](authorship_metrics.go#L10), [authorship_metrics.go#L14](authorship_metrics.go#L14)).
- `IdentityTopK` exists to bound legend size, not to filter data ([authorship_metrics.go#L42](authorship_metrics.go#L42)).

**Related operations.**

- `DefaultAuthorshipParams` for the baseline; the stages layer merges the user's `config.AuthorshipConfig` over it ([authorship_metrics.go#L53](authorship_metrics.go#L53), [../../stages/providers.go#L174](../../stages/providers.go#L174)).

**Proper-use patterns.**

- Start from the defaults and apply only the values the user set, so unset config keys keep spec behaviour ([../../stages/providers.go#L173](../../stages/providers.go#L173)).

**Anti-patterns.**

- Do not treat the sentinel strings as real author names; they are deliberately bracketed placeholders ([authorship_metrics.go#L14](authorship_metrics.go#L14)).

**Source locations.**

- [authorship_metrics.go#L10](authorship_metrics.go#L10) — `Unmaintained`, `OtherContributor`.
- [authorship_metrics.go#L24](authorship_metrics.go#L24) — `AuthorshipParams`, `DefaultAuthorshipParams`.
