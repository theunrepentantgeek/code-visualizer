# Git Metrics Performance Design

Date: 2026-08-09
Branch: `fix/performance`

## Revision: shared spiral history and metric acquisition

The first implementation removed repeated tree diffs and stopped unrequested
metric writes, but the acceptance benchmark still had a 75.85-second median.
Profiling showed two separate full Git-history traversals: 33.68 seconds for
Git metric prewarming and 32.05 seconds for spiral history loading. The
following extension replaces those two passes with one pass for spiral renders.

## Goal

Restore practical Git-metric loading performance across every visualization.
The acceptance benchmark is three cold runs of:

```sh
codeviz spiral --config samples/spiral/code-visualizer.yml \
  /home/bevan/github/azure-service-operator --output <temporary.svg>
```

The median complete-process runtime, including SVG output, must be under
60 seconds.

## Problem

The Git metrics are served by one consolidated loader. Selecting any one Git
metric activates the loader, which currently:

1. Prewarms all per-file Git history in one commit-graph traversal.
2. Calculates all seven Git metrics for every scanned file, regardless of the
   metrics selected by the visualization.
3. Calculates line-churn statistics for every changed tracked file. The current
   line-churn path performs a full tree diff once per changed file.

The acceptance command selects only `commit-count`, but it still pays for
`total-lines-added` and `total-lines-removed`. On the benchmark repository,
one cold baseline run took 1,022.99 seconds.

## Scope

- Optimise the shared Git metric loader for all visualizations.
- Preserve one-pass traversal of the Git commit graph.
- Avoid line-churn work unless a selected metric requires it.
- Remove repeated full-tree diffs when line-churn metrics are selected.
- Add regression coverage for selection-aware loading and the efficient
  churn path.

## Non-goals

- Changing CLI or configuration syntax.
- Replacing go-git with Git command-line subprocesses.
- Changing Git history, merge, or TREESAME semantics.
- Caching data across separate `codeviz` processes.
- Optimising non-Git portions of rendering.

## Architecture

### Loader requests

`provider.RunLoaders` will calculate the subset of requested metric names
served by each selected `BaseMetricLoader`. It will pass that subset to the
loader invocation.

The consolidated Git loader receives only its requested Git metric names and
derives an internal requirements value containing:

- the processors that write requested values to `model.File`;
- `needsLineStats`, true only when `total-lines-added` or
  `total-lines-removed` is requested.

The loader continues to be registered as one loader for the complete Git
metric family. Loader selection, dependency ordering, and parallel execution
remain unchanged.

### Git prewarm

The loader builds cache entries for every scanned path and walks the commit
graph once. For every commit, it:

1. Computes each required parent/tree diff once.
2. Filters those changes to scanned paths using existing TREESAME semantics,
   including the existing all-parent rule for merges.
3. Updates timestamps, author identities, and commit counts for each tracked
   changed path.
4. When `needsLineStats` is true, derives that path's patch from the
   already-computed diff and accumulates additions and removals.

When no churn metric is requested, step 4 is skipped entirely. No patch is
generated, and no additions/removals are accumulated.

### File writes

After prewarming, the loader visits scanned files once and applies only the
requested Git metric processors. Unrequested Git metric fields remain unset.
The `commitData` cache remains complete enough for each requested metric: it
always has timestamp, author, and count data, and has line statistics only
when requested.

For the acceptance benchmark, only `commit-count` is requested. It therefore
performs one history walk without patch generation, churn accumulation, or
unused metric writes.

## Compatibility and errors

Public metric names, configuration, output semantics, and Git loader
registration do not change. Churn values retain their current meaning, but
reuse each commit's already-computed diff instead of recomputing it for each
changed file.

Repository-open, HEAD, history-iteration, and no-tracked-history errors retain
their current behavior and wrapped messages. A tree, diff, or patch failure
continues to omit churn statistics for only the affected change, matching the
current best-effort per-file behavior rather than turning a successful render
into an error.

Progress still counts changed tracked files. Its count does not vary with the
requested metric subset.

## Testing

Unit and package tests will cover:

- loader scheduling passes each loader only the subset it serves;
- selecting `commit-count` writes `commit-count` without populating unrequested
  Git metrics;
- selecting either churn metric produces correct additions and removals;
- a commit that changes several tracked files uses the single-diff churn path;
- existing merge-history and progress behavior remains correct.

The normal repository test suite validates functional behavior. The external
performance acceptance test builds the binary, runs the supplied command as
three cold processes with temporary SVG outputs, removes generated artifacts,
and calculates the median elapsed runtime. The result passes only when that
median is under 60 seconds.

## Expected outcome

The Git loader continues to scale with one commit-graph walk, while a
non-churn request avoids the expensive per-file patch path entirely. Requests
for churn metrics retain correct values and no longer repeat full tree diffs
for every changed file.

## Shared spiral acquisition

### Goal

For spiral visualizations, use one Git commit traversal to produce both the
rich history required for time buckets and the cache entries required by the
selected Git metrics. The Azure Service Operator acceptance benchmark remains
three cold SVG renders with a median under 60 seconds.

### Data flow

After scanning and Git-requirement validation, `spiral.AcquireData` loads Git
history before running providers. `stages.LoadGitHistory` supplies
`CommonState.Requested.BaseMetrics` to a new Git history API.

That API walks the commit graph once and, for each tracked change:

1. Produces the same `git.Commit` value and `ChangedPaths` used by
   `GroupGitHistoryByFile`.
2. Updates the corresponding cached `commitData` using the selected Git metric
   requirements.
3. Generates patch statistics only if a selected churn metric requires them.

The API atomically publishes both the completed history slice and the completed
metric cache. `RunProviders` then runs as before. Its Git loader finds complete
cache entries and writes selected values without another history traversal;
filesystem and Go providers are unchanged.

When a spiral requests no Git metric, the history API still returns the rich
history required by the visualization but performs no metric-cache work.
Other visualizations retain their current provider-first behavior and public
Git APIs retain their existing semantics.

### Compatibility and failure handling

The returned history records, file-history grouping, metric names, values,
progress callbacks, and rendered output do not change. The stage retains its
current wrapped errors for repository resolution, history traversal, and empty
history. Cache publication remains atomic: metadata-only work must never
overwrite cached line statistics.

### Tests

Tests will prove that a shared spiral pass produces the same history and metric
values as separate passes for commit-count and churn requests; that subsequent
Git provider loading reuses the warmed cache without a second traversal; and
that stage ordering still loads history before providers. Existing merge and
first-parent churn tests remain authoritative. The external acceptance
benchmark is the three-run cold median under 60 seconds.
