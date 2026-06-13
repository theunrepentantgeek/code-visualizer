# Scatter Logarithmic Axis Scales Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add per-axis logarithmic scale options (`--x-scale=log`, `--y-scale=log`) to scatter visualizations so clustered low-range data spreads out.

**Architecture:** Add `ScaleType` (Linear/Log) to `AxisSpec`. Thread the scale through config → CLI → axis resolution → tick generation → position mapping. Log positioning uses `ln(value)` normalized to [0,1]; tick labels show original values. Validate all values positive when log is active.

**Tech Stack:** Go 1.26.1 + Kong (CLI), Gomega (assertions), existing `internal/scatter` axis resolution code.

---

## File Map

| File | Action | Responsibility |
|------|--------|---------------|
| `internal/scatter/axis.go` | Modify | Add `ScaleType` type and constants; add `Scale` field to `AxisSpec` |
| `internal/config/scatter.go` | Modify | Add `XScale`/`YScale` fields + override methods |
| `cmd/codeviz/scatter_cmd.go` | Modify | Add `--x-scale`/`--y-scale` CLI flags; wire overrides |
| `internal/scatter/axis_resolve.go` | Modify | Add `logNumericTicks`; branch `resolveAxis` and `positionForValue` on scale |
| `internal/scatter/resolved_axis.go` | Modify | Add `Scale` field to `NumericAxis` |
| `internal/scatter/stages.go` | Modify | Parse scale from config in `resolveAxisSpec`; add `validateLogScale` |
| `internal/scatter/layout_test.go` | Modify | Add tests for log tick generation, log positioning, validation |
| `internal/scatter/stages_test.go` | Modify | Add test for scale parsing in `ResolveMetrics` |

---

### Task 1: Add ScaleType and Config Fields

**Files:**
- Modify: `internal/scatter/axis.go`
- Modify: `internal/config/scatter.go`

- [ ] **Step 1: Add `ScaleType` to `axis.go`**

Add after the `AxisBand` struct at the end of `internal/scatter/axis.go`:

```go
// ScaleType controls how numeric values are mapped to axis positions.
type ScaleType int

const (
	Linear ScaleType = iota
	Log
)
```

Add a `Scale` field to `AxisSpec`:

```go
type AxisSpec struct {
	Metric metric.Name
	Kind   metric.Kind
	Scale  ScaleType
}
```

- [ ] **Step 2: Add `Scale` field to `NumericAxis`**

In `internal/scatter/resolved_axis.go`, add `Scale ScaleType` to `NumericAxis`:

```go
type NumericAxis struct {
	Min   float64
	Max   float64
	Scale ScaleType
	Ticks []AxisTick
}
```

- [ ] **Step 3: Add config fields**

In `internal/config/scatter.go`, add after the `Border` field:

```go
XScale *string `yaml:"xScale,omitempty" json:"xScale,omitempty"`
YScale *string `yaml:"yScale,omitempty" json:"yScale,omitempty"`
```

Add override methods:

```go
// OverrideXScale sets XScale to v if v is non-empty.
func (s *Scatter) OverrideXScale(v string) { overrideString(&s.XScale, v) }

// OverrideYScale sets YScale to v if v is non-empty.
func (s *Scatter) OverrideYScale(v string) { overrideString(&s.YScale, v) }
```

- [ ] **Step 4: Verify build**

Run: `task build`
Expected: clean compile (no test changes needed yet — `Scale` zero-value is `Linear`).

- [ ] **Step 5: Commit**

```bash
git add internal/scatter/axis.go internal/scatter/resolved_axis.go internal/config/scatter.go
git commit -m "feat(scatter): add ScaleType and config fields for log axis (#396)"
```

---

### Task 2: Wire CLI Flags and Config Parsing

**Files:**
- Modify: `cmd/codeviz/scatter_cmd.go`
- Modify: `internal/scatter/stages.go`

- [ ] **Step 1: Add CLI flags to `ScatterCmd`**

In `cmd/codeviz/scatter_cmd.go`, add after the `Size` field:

```go
XScale string `default:"" enum:",linear,log" help:"X-axis scale (linear or log)." name:"x-scale"` //nolint:revive,nolintlint // kong struct tags require long lines
YScale string `default:"" enum:",linear,log" help:"Y-axis scale (linear or log)." name:"y-scale"` //nolint:revive,nolintlint // kong struct tags require long lines
```

- [ ] **Step 2: Wire overrides in `applyOverrides`**

In `cmd/codeviz/scatter_cmd.go`, in `applyOverrides`, add after `cfg.Scatter.OverrideSize(string(c.Size))`:

```go
cfg.Scatter.OverrideXScale(c.XScale)
cfg.Scatter.OverrideYScale(c.YScale)
```

- [ ] **Step 3: Parse scale in `resolveAxisSpec`**

In `internal/scatter/stages.go`, change `resolveAxisSpec` to accept a scale string:

```go
func resolveAxisSpec(name *string, scale *string) (AxisSpec, error) {
	metricName := metric.Name(stages.PtrString(name))
	descriptor, ok := provider.GetDescriptor(metricName)

	if !ok {
		return AxisSpec{}, eris.Errorf("unknown axis metric %q", metricName)
	}

	spec := AxisSpec{Metric: metricName, Kind: descriptor.Kind}

	switch stages.PtrString(scale) {
	case "", "linear":
		spec.Scale = Linear
	case "log":
		spec.Scale = Log
	default:
		return AxisSpec{}, eris.Errorf("unknown scale %q; must be \"linear\" or \"log\"", stages.PtrString(scale))
	}

	return spec, nil
}
```

- [ ] **Step 4: Update `ResolveMetrics` callers**

In `internal/scatter/stages.go`, update the two calls to `resolveAxisSpec` in `ResolveMetrics`:

```go
xAxis, err := resolveAxisSpec(cfg.XAxis, cfg.XScale)
```

```go
yAxis, err := resolveAxisSpec(cfg.YAxis, cfg.YScale)
```

- [ ] **Step 5: Verify build and existing tests pass**

Run: `task build && task test`
Expected: all pass (existing tests use `nil` scale which maps to `Linear`).

- [ ] **Step 6: Commit**

```bash
git add cmd/codeviz/scatter_cmd.go internal/scatter/stages.go
git commit -m "feat(scatter): wire --x-scale/--y-scale CLI flags through config (#396)"
```

---

### Task 3: Add Scale Parsing Test

**Files:**
- Modify: `internal/scatter/stages_test.go`

- [ ] **Step 1: Write test for log scale parsing**

Add to `internal/scatter/stages_test.go`:

```go
func TestResolveMetrics_ParsesLogScale(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	common := &stages.CommonState{}
	viz := &scatter.State{}
	cfg := &config.Scatter{
		XAxis:  new("file-lines"),
		YAxis:  new("file-size"),
		Size:   new("file-size"),
		XScale: new("log"),
		YScale: new("linear"),
	}

	err := scatter.ResolveMetrics(common, viz, cfg)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(viz.XAxis.Scale).To(Equal(scatter.Log))
	g.Expect(viz.YAxis.Scale).To(Equal(scatter.Linear))
}
```

- [ ] **Step 2: Run and verify test passes**

Run: `go test -run TestResolveMetrics_ParsesLogScale ./internal/scatter/ -count=1 -v`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/scatter/stages_test.go
git commit -m "test(scatter): add test for log scale parsing in ResolveMetrics (#396)"
```

---

### Task 4: Implement Log Tick Generation

**Files:**
- Modify: `internal/scatter/axis_resolve.go`
- Modify: `internal/scatter/layout_test.go`

- [ ] **Step 1: Write failing test for `logNumericTicks`**

Add to `internal/scatter/layout_test.go`:

```go
func TestLogNumericTicks_SpansMultipleDecades(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	plot := PlotRect{X: 0, Y: 0, W: 800, H: 600}
	ticks := logNumericTicks(1, 10000, plot, horizontalAxis)

	// Expect ticks at powers of 10: 1, 10, 100, 1000, 10000
	g.Expect(ticks).To(HaveLen(5))
	g.Expect(ticks[0].Value).To(BeNumerically("~", 1, 1e-9))
	g.Expect(ticks[1].Value).To(BeNumerically("~", 10, 1e-9))
	g.Expect(ticks[2].Value).To(BeNumerically("~", 100, 1e-9))
	g.Expect(ticks[3].Value).To(BeNumerically("~", 1000, 1e-9))
	g.Expect(ticks[4].Value).To(BeNumerically("~", 10000, 1e-9))

	// Positions should be logarithmically spaced (equal increments in log space)
	for i := 1; i < len(ticks); i++ {
		g.Expect(ticks[i].Position).To(BeNumerically(">", ticks[i-1].Position))
	}

	// Each gap should be the same size (equal decades = equal spacing)
	gap := ticks[1].Position - ticks[0].Position
	for i := 2; i < len(ticks); i++ {
		g.Expect(ticks[i].Position - ticks[i-1].Position).To(BeNumerically("~", gap, 1e-6))
	}
}

func TestLogNumericTicks_NarrowRange_AddsIntermediateTicks(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	plot := PlotRect{X: 0, Y: 0, W: 800, H: 600}
	ticks := logNumericTicks(50, 500, plot, horizontalAxis)

	// Range spans ~1 decade, so intermediate ticks (2x, 5x) are added.
	// Expect at least 4 ticks.
	g.Expect(len(ticks)).To(BeNumerically(">=", 4))

	// All tick values should be within [50, 500]
	for _, tick := range ticks {
		g.Expect(tick.Value).To(BeNumerically(">=", 50))
		g.Expect(tick.Value).To(BeNumerically("<=", 500))
	}

	// Positions should be monotonically increasing
	for i := 1; i < len(ticks); i++ {
		g.Expect(ticks[i].Position).To(BeNumerically(">", ticks[i-1].Position))
	}
}

func TestLogNumericTicks_SingleValue_ReturnsCenterTick(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	plot := PlotRect{X: 0, Y: 0, W: 800, H: 600}
	ticks := logNumericTicks(42, 42, plot, horizontalAxis)

	g.Expect(ticks).To(HaveLen(1))
	g.Expect(ticks[0].Value).To(BeNumerically("~", 42, 1e-9))
	g.Expect(ticks[0].Position).To(BeNumerically("~", 400, 1e-6)) // center of 800-wide plot
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -run TestLogNumericTicks ./internal/scatter/ -count=1 -v 2>&1 | tail -10`
Expected: FAIL — `logNumericTicks` undefined.

- [ ] **Step 3: Implement `logNumericTicks`**

Add to `internal/scatter/axis_resolve.go`:

```go
func logNumericTicks(minValue, maxValue float64, plot PlotRect, direction axisDirection) []AxisTick {
	if minValue == maxValue {
		return []AxisTick{{
			Value:    minValue,
			Label:    formatTick(minValue, 0),
			Position: direction.center(plot),
		}}
	}

	logMin := math.Log(minValue)
	logMax := math.Log(maxValue)

	// Collect candidate tick values at powers of 10 within [minValue, maxValue]
	candidates := logTickCandidates(minValue, maxValue)

	// If fewer than 4 ticks, add 2x and 5x intermediate values per decade
	if len(candidates) < 4 {
		candidates = logTickCandidatesWithSubdivisions(minValue, maxValue)
	}

	ticks := make([]AxisTick, 0, len(candidates))
	for _, value := range candidates {
		norm := (math.Log(value) - logMin) / (logMax - logMin)
		ticks = append(ticks, AxisTick{
			Value:    value,
			Label:    formatTick(value, 0),
			Position: direction.position(plot, norm),
		})
	}

	return ticks
}

// logTickCandidates returns powers of 10 within [minValue, maxValue].
func logTickCandidates(minValue, maxValue float64) []float64 {
	startExp := math.Floor(math.Log10(minValue))
	endExp := math.Ceil(math.Log10(maxValue))

	candidates := make([]float64, 0, int(endExp-startExp)+1)
	for exp := startExp; exp <= endExp; exp++ {
		value := math.Pow(10, exp)
		if value >= minValue && value <= maxValue {
			candidates = append(candidates, value)
		}
	}

	return candidates
}

// logTickCandidatesWithSubdivisions returns powers of 10 plus 2x and 5x
// subdivisions within [minValue, maxValue].
func logTickCandidatesWithSubdivisions(minValue, maxValue float64) []float64 {
	startExp := math.Floor(math.Log10(minValue))
	endExp := math.Ceil(math.Log10(maxValue))
	multipliers := []float64{1, 2, 5}

	candidates := make([]float64, 0, int(endExp-startExp)*3+1)
	for exp := startExp; exp <= endExp; exp++ {
		base := math.Pow(10, exp)
		for _, m := range multipliers {
			value := base * m
			if value >= minValue && value <= maxValue {
				candidates = append(candidates, value)
			}
		}
	}

	return candidates
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -run TestLogNumericTicks ./internal/scatter/ -count=1 -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/scatter/axis_resolve.go internal/scatter/layout_test.go
git commit -m "feat(scatter): implement logNumericTicks with decade/subdivision ticks (#396)"
```

---

### Task 5: Branch `resolveAxis` and `positionForValue` on Scale

**Files:**
- Modify: `internal/scatter/axis_resolve.go`
- Modify: `internal/scatter/layout_test.go`

- [ ] **Step 1: Write failing test for log positioning**

Add to `internal/scatter/layout_test.go`:

```go
func TestLayout_LogScalePositionsPointsLogarithmically(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	small := scatterTestFile("small.go")
	small.SetQuantity(filesystem.FileLines, 10)
	small.SetQuantity(filesystem.FileSize, 100)

	medium := scatterTestFile("medium.go")
	medium.SetQuantity(filesystem.FileLines, 100)
	medium.SetQuantity(filesystem.FileSize, 100)

	large := scatterTestFile("large.go")
	large.SetQuantity(filesystem.FileLines, 1000)
	large.SetQuantity(filesystem.FileSize, 100)

	root := &model.Directory{Files: []*model.File{small, medium, large}}
	xAxis := AxisSpec{Metric: filesystem.FileLines, Kind: metric.Quantity, Scale: Log}
	yAxis := AxisSpec{Metric: filesystem.FileSize, Kind: metric.Quantity, Scale: Linear}

	dataset := CollectDataset(root, xAxis, yAxis, filesystem.FileSize)
	layout := Layout(dataset, 800, 600, xAxis, yAxis)

	points := map[string]ScatterPoint{}
	for _, point := range layout.Points {
		points[point.File.Name] = point
	}

	// With log scale, the gap between 10→100 should equal the gap between 100→1000
	// (both are one decade)
	gap1 := points["medium.go"].X - points["small.go"].X
	gap2 := points["large.go"].X - points["medium.go"].X
	g.Expect(gap1).To(BeNumerically("~", gap2, 1.0))

	// All X values should be within the plot area
	g.Expect(points["small.go"].X).To(BeNumerically(">=", scatterPlotLeftMargin))
	g.Expect(points["large.go"].X).To(BeNumerically("<=", 800-scatterPlotRightMargin))
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test -run TestLayout_LogScalePositionsPointsLogarithmically ./internal/scatter/ -count=1 -v 2>&1 | tail -10`
Expected: FAIL — log scale not yet used in `resolveAxis`/`positionForValue`.

- [ ] **Step 3: Update `resolveAxis` to branch on scale**

In `internal/scatter/axis_resolve.go`, modify `resolveAxis`:

```go
func resolveAxis(points []PointDatum, plot PlotRect, spec AxisSpec, direction axisDirection) ResolvedAxis {
	axis := ResolvedAxis{Spec: spec, Title: string(spec.Metric)}
	if spec.Kind == metric.Classification {
		axis.Categorical = &CategoricalAxis{Bands: categoricalBands(points, plot, direction)}

		return axis
	}

	minValue, maxValue := numericExtent(points, direction)

	if spec.Scale == Log {
		axis.Numeric = &NumericAxis{
			Min:   minValue,
			Max:   maxValue,
			Scale: Log,
			Ticks: logNumericTicks(minValue, maxValue, plot, direction),
		}
	} else {
		axis.Numeric = &NumericAxis{
			Min:   minValue,
			Max:   maxValue,
			Scale: Linear,
			Ticks: numericTicks(minValue, maxValue, plot, direction),
		}
	}

	return axis
}
```

- [ ] **Step 4: Update `positionForValue` for log scale**

In `internal/scatter/axis_resolve.go`, modify `positionForValue`:

```go
func positionForValue(value AxisValue, axis ResolvedAxis, plot PlotRect, direction axisDirection) float64 {
	if axis.Categorical != nil {
		for _, band := range axis.Categorical.Bands {
			if band.Label == value.Category {
				return band.Center
			}
		}

		return direction.center(plot)
	}

	minValue := axis.Numeric.Min
	maxValue := axis.Numeric.Max

	if minValue == maxValue {
		return direction.center(plot)
	}

	var norm float64
	if axis.Numeric.Scale == Log {
		norm = (math.Log(value.Numeric) - math.Log(minValue)) / (math.Log(maxValue) - math.Log(minValue))
	} else {
		norm = (value.Numeric - minValue) / (maxValue - minValue)
	}

	return direction.position(plot, norm)
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test -run TestLayout_LogScalePositionsPointsLogarithmically ./internal/scatter/ -count=1 -v`
Expected: PASS

- [ ] **Step 6: Run all tests to verify nothing is broken**

Run: `task test`
Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add internal/scatter/axis_resolve.go internal/scatter/layout_test.go
git commit -m "feat(scatter): branch resolveAxis and positionForValue on log scale (#396)"
```

---

### Task 6: Add Log Scale Validation

**Files:**
- Modify: `internal/scatter/stages.go`
- Modify: `internal/scatter/layout_test.go`

- [ ] **Step 1: Write failing test for validation**

Add to `internal/scatter/layout_test.go`:

```go
func TestValidateLogScale_ErrorsOnZeroValue(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	zero := scatterTestFile("zero.go")
	zero.SetQuantity(filesystem.FileLines, 0)
	zero.SetQuantity(filesystem.FileSize, 100)

	positive := scatterTestFile("positive.go")
	positive.SetQuantity(filesystem.FileLines, 10)
	positive.SetQuantity(filesystem.FileSize, 50)

	root := &model.Directory{Files: []*model.File{zero, positive}}
	xAxis := AxisSpec{Metric: filesystem.FileLines, Kind: metric.Quantity, Scale: Log}
	yAxis := AxisSpec{Metric: filesystem.FileSize, Kind: metric.Quantity, Scale: Linear}

	dataset := CollectDataset(root, xAxis, yAxis, filesystem.FileSize)

	err := ValidateLogScale(dataset, xAxis, yAxis)
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("x-axis"))
	g.Expect(err.Error()).To(ContainSubstring("zero.go"))
}

func TestValidateLogScale_PassesWhenAllPositive(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	a := scatterTestFile("a.go")
	a.SetQuantity(filesystem.FileLines, 10)
	a.SetQuantity(filesystem.FileSize, 100)

	b := scatterTestFile("b.go")
	b.SetQuantity(filesystem.FileLines, 200)
	b.SetQuantity(filesystem.FileSize, 50)

	root := &model.Directory{Files: []*model.File{a, b}}
	xAxis := AxisSpec{Metric: filesystem.FileLines, Kind: metric.Quantity, Scale: Log}
	yAxis := AxisSpec{Metric: filesystem.FileSize, Kind: metric.Quantity, Scale: Log}

	dataset := CollectDataset(root, xAxis, yAxis, filesystem.FileSize)

	err := ValidateLogScale(dataset, xAxis, yAxis)
	g.Expect(err).NotTo(HaveOccurred())
}

func TestValidateLogScale_SkipsLinearAxes(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	zero := scatterTestFile("zero.go")
	zero.SetQuantity(filesystem.FileLines, 0)
	zero.SetQuantity(filesystem.FileSize, 100)

	root := &model.Directory{Files: []*model.File{zero}}
	xAxis := AxisSpec{Metric: filesystem.FileLines, Kind: metric.Quantity, Scale: Linear}
	yAxis := AxisSpec{Metric: filesystem.FileSize, Kind: metric.Quantity, Scale: Linear}

	dataset := CollectDataset(root, xAxis, yAxis, filesystem.FileSize)

	err := ValidateLogScale(dataset, xAxis, yAxis)
	g.Expect(err).NotTo(HaveOccurred())
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -run TestValidateLogScale ./internal/scatter/ -count=1 -v 2>&1 | tail -10`
Expected: FAIL — `ValidateLogScale` undefined.

- [ ] **Step 3: Implement `ValidateLogScale`**

Add to `internal/scatter/stages.go`:

```go
// ValidateLogScale checks that all data values are positive when log scale is used.
func ValidateLogScale(dataset Dataset, xAxis, yAxis AxisSpec) error {
	if xAxis.Scale == Log {
		for _, point := range dataset.Points {
			if point.X.Numeric <= 0 {
				return eris.Errorf(
					"log scale on x-axis requires all values to be positive; file %q has value %g",
					point.File.Name, point.X.Numeric,
				)
			}
		}
	}

	if yAxis.Scale == Log {
		for _, point := range dataset.Points {
			if point.Y.Numeric <= 0 {
				return eris.Errorf(
					"log scale on y-axis requires all values to be positive; file %q has value %g",
					point.File.Name, point.Y.Numeric,
				)
			}
		}
	}

	return nil
}
```

- [ ] **Step 4: Call `ValidateLogScale` from `BuildInksStage`**

In `internal/scatter/stages.go`, in `BuildInksStage`, add after the `CollectDataset` call:

```go
func BuildInksStage(c *stages.CommonState, x *State) error {
	x.Dataset = CollectDataset(c.Root, x.XAxis, x.YAxis, x.Size)

	if err := ValidateLogScale(x.Dataset, x.XAxis, x.YAxis); err != nil {
		return err
	}

	x.Inks = BuildInks(x.Dataset, x.FillMetric, x.FillPalette, x.BorderMetric, x.BorderPalette)

	slog.Info("Rendering image", "output", c.Output, "width", c.Width, "height", c.Height)

	return nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test -run TestValidateLogScale ./internal/scatter/ -count=1 -v`
Expected: PASS

- [ ] **Step 6: Run full test suite**

Run: `task test`
Expected: all pass.

- [ ] **Step 7: Commit**

```bash
git add internal/scatter/stages.go internal/scatter/layout_test.go
git commit -m "feat(scatter): validate positive values required for log scale axes (#396)"
```

---

### Task 7: Final Verification

**Files:** None (verification only)

- [ ] **Step 1: Run full CI checks**

Run: `task ci`
Expected: build, test, lint all pass.

- [ ] **Step 2: Verify sample config works with log scale**

Run:
```bash
go run ./cmd/codeviz scatter . -o /tmp/scatter-log-test.png \
  --x-axis file-size --y-axis file-lines --size file-size \
  --x-scale log --y-scale log
```
Expected: produces `/tmp/scatter-log-test.png` without errors (some files may be skipped if they have 0 lines — that's an expected validation error which tells us the feature works).

If validation errors about zero values, run with only `--x-scale log` on a metric known to be positive (e.g., `file-size`):

```bash
go run ./cmd/codeviz scatter . -o /tmp/scatter-log-test.png \
  --x-axis file-size --y-axis file-lines --size file-size \
  --x-scale log
```

- [ ] **Step 3: Verify help output includes new flags**

Run: `go run ./cmd/codeviz scatter --help 2>&1 | grep -A1 'scale'`
Expected: shows `--x-scale` and `--y-scale` with `linear` and `log` enum values.
