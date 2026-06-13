# Scatter Logarithmic Axis Scales

**Date:** 2026-06-13
**Issue:** #396

## Summary

Add per-axis logarithmic scale options to the scatter visualization.
Log scaling spreads out data points that cluster at low values by positioning
them according to `ln(value)` while labeling ticks with original values.

## Configuration

### Config YAML

Add two new optional fields to the `scatter:` section:

```yaml
scatter:
  xAxis: file-size
  yAxis: comment-ratio
  xScale: log       # new — "linear" (default) or "log"
  yScale: linear    # new — "linear" (default) or "log"
  size: declaration-count
```

### CLI Flags

```
--x-scale=log    X-axis scale (linear or log). Default: linear.
--y-scale=log    Y-axis scale (linear or log). Default: linear.
```

## Types

New `ScaleType` in `internal/scatter/axis.go`:

```go
type ScaleType int

const (
    Linear ScaleType = iota
    Log
)
```

`AxisSpec` gains a `Scale ScaleType` field.

`config.Scatter` gains:

```go
XScale *string `yaml:"xScale,omitempty" json:"xScale,omitempty"`
YScale *string `yaml:"yScale,omitempty" json:"yScale,omitempty"`
```

With corresponding `OverrideXScale` / `OverrideYScale` methods.

`ScatterCmd` gains:

```go
XScale string `default:"" enum:",linear,log" help:"X-axis scale (linear or log)." name:"x-scale"`
YScale string `default:"" enum:",linear,log" help:"Y-axis scale (linear or log)." name:"y-scale"`
```

## Validation

Log scale requires all data values on that axis to be strictly positive (`> 0`).
Validation occurs after dataset collection (in `BuildInksStage`), since actual
values are not known until providers have run.

A new function:

```go
func validateLogScale(dataset Dataset, xAxis, yAxis AxisSpec) error
```

Returns an error identifying the first file with a non-positive value:

```
log scale on x-axis requires all values to be positive; file "foo.go" has value 0
```

## Log Tick Generation

New function `logNumericTicks(minValue, maxValue float64, plot PlotRect, direction axisDirection) []AxisTick`:

1. Compute `logMin = ln(min)`, `logMax = ln(max)`.
2. Choose tick values at powers of 10 within `[min, max]` (1, 10, 100, 1000, …).
3. If the resulting tick count is fewer than 3, supplement with 2× and 5×
   intermediate ticks per decade (e.g., 20, 50, 200, 500) until at least 4 ticks
   are present.
4. Position each tick: `norm = (ln(tickValue) - logMin) / (logMax - logMin)`.
5. Labels show original numeric values via `formatTick`.

## Position Mapping

`positionForValue` gains a log branch when `axis.Numeric.Scale == Log`:

```go
norm = (math.Log(value.Numeric) - math.Log(minValue)) / (math.Log(maxValue) - math.Log(minValue))
```

The `NumericAxis` struct stores the `Scale` so `positionForValue` can branch:

```go
type NumericAxis struct {
    Min   float64
    Max   float64
    Scale ScaleType
    Ticks []AxisTick
}
```

## Axis Resolution Changes

`resolveAxis` receives the `ScaleType` from `AxisSpec.Scale` and:

- For `Log`: calls `logNumericTicks` instead of `numericTicks`.
- Sets `axis.Numeric.Scale = spec.Scale` so downstream code can branch.

## Pipeline Wiring

1. `resolveAxisSpec` in `stages.go` parses the scale string from config and sets
   `AxisSpec.Scale`.
2. `applyOverrides` in `scatter_cmd.go` calls `cfg.Scatter.OverrideXScale(c.XScale)`
   and `cfg.Scatter.OverrideYScale(c.YScale)`.
3. `BuildInksStage` calls `validateLogScale` after `CollectDataset`.

## Testing

1. **`logNumericTicks` unit tests**: verify tick values and positions for ranges
   like 1–1000, 50–500, 1–10000.
2. **`positionForValue` log tests**: verify log-proportional positioning.
3. **`validateLogScale` tests**: error on zero/negative, pass on all-positive.
4. **Golden-file integration test**: render a scatter plot with `xScale: log` and
   verify via Goldie snapshot.
5. **Config round-trip test**: verify `xScale`/`yScale` serialize and deserialize
   correctly.
