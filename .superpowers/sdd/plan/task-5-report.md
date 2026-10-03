# Task 5 Report: Alluvial per-snapshot label-only metrics

Implemented Alluvial label metrics from acquisition through rendering:

- Resolves configured labels with Alluvial directory-expression rules and requests them for snapshot acquisition.
- Formats ordered quantity, measure, classification, and missing (`-`) values per directory in each snapshot.
- Preserves those values through `Data.Value` and layout `Band`.
- Renders path, width, explicit fill, then additional values; muted bands remain unlabeled.
- Uses the complete label line count for font suppression and edge extension.
- Appends resolved label metric names to the square thumbnail only, without adding colour-key entries.

Tests were written first and observed failing before implementation. Focused and complete `internal/alluvial` package tests pass.

Self-review found no changes outside Task 5. The pre-existing modifications to `.custom-gcl.yml` and `internal/progress/tty.go` were not touched or staged.

## Review round 1

- Added `TestBuildData_MergePreservesPerColumnLabelsForChangedPaths`, covering distinct per-column label values on a changed, unmuted band when constant bands are merged.
- Updated `mergedColumnValues` to reuse the original `Value` for single-member, unmuted groups, preserving its labels. Merged muted groups continue to use a synthesized value without labels.

Commands and outcomes:

- `./dev.sh -c 'go test ./internal/alluvial -run TestBuildData_MergePreservesPerColumnLabelsForChangedPaths -count=1'` — FAIL as expected before the fix: the changed band had `Labels: nil` instead of `["12"]`.
- `./dev.sh -c 'go test ./internal/alluvial -run TestBuildData_MergePreservesPerColumnLabelsForChangedPaths -count=1'` — PASS after the fix (`ok`, 0.005s).
- `./dev.sh -c 'go test ./internal/alluvial -count=1'` — PASS (`ok`, 0.009s).
