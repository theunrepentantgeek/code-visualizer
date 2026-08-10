# Unify Git History Traversal Design

## Goal

Make one internal traversal responsible for walking tracked Git history, producing
timeline commits, and optionally warming per-file metric data. Preserve the
existing public `BulkCommitHistory` API and all metric results.

## Design

Introduce an internal walker that:

1. Opens and iterates the repository history once.
2. Determines tracked changes for each commit through the existing shared
   `trackedChangesInCommit` helper.
3. Invokes the progress callback once per examined commit.
4. Optionally appends the commit's timeline representation.
5. Optionally updates a supplied prewarm cache using the requested metric
   requirements.

`BulkCommitHistory` calls this walker without cache warming.
`BulkCommitHistoryAndPrewarm` calls it with a cache and publishes the completed
cache only after a successful walk. `bulkPrewarm` delegates to the same walker
without collecting timeline commits.

## Constraints

- Preserve TREESAME filtering, tracked-path normalization, callback timing, and
  cache-completeness semantics.
- Do not expose the internal walker or change caller-facing APIs.
- Keep prewarm cache publication atomic with respect to a completed traversal.

## Verification

Existing history and prewarm tests must cover the wrapper paths and the shared
walker. The full Go test suite must pass.
