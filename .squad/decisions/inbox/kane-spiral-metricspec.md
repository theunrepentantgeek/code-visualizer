# Spiral Config Uses MetricSpec (Not Separate Fields)

**Author:** Kane
**Date:** 2026-07-19
**Status:** Implemented

## Decision

The `config.Spiral` struct uses `*MetricSpec` for Fill and Border fields (matching Treemap, Radial, and Bubbletree post-issue #118), instead of the architecture doc's original proposal of separate `*string` fields for Fill/FillPalette/Border/BorderPalette.

## Rationale

The architecture proposal was written before MetricSpec consolidation (#118/#120). All three existing visualization config structs now use `*MetricSpec`. Keeping spiral consistent avoids a special case and ensures the CLI `--fill metric,palette` syntax works the same way everywhere.

## Impact

- `cmd/codeviz/spiral_cmd.go` uses `config.MetricSpec` for Fill/Border CLI flags
- `internal/config/spiral.go` uses `*MetricSpec` for Fill/Border config fields
- No `FillPalette`/`BorderPalette` separate fields exist on spiral config or CLI
