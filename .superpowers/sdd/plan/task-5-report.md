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
