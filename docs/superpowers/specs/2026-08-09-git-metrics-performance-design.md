# Git Metrics Performance Design

Date: 2026-08-09
Branch: `fix/performance`

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
