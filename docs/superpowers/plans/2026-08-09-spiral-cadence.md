# Adaptive Spiral Cadence Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Adapt daily spiral dots-per-lap to timeline length so midpoint radial and arc spacing are as equal as possible using only 14, 28, 42, or 56 days.

**Architecture:** Add a cadence selector that scores the four daily candidates with the existing Archimedean geometry. Refactor the shared layout parameter calculation to accept an explicit dots-per-lap value, preserve the existing `Resolution`-based public helpers as compatibility wrappers, and have the pipeline store and use the selected value. Hourly continues to pass its fixed 24 dots-per-lap value.

**Tech Stack:** Go 1.26, Gomega, Task.

---

## File Structure

- Create: `internal/spiral/cadence.go` — selects and scores the allowed daily cadence values.
- Create: `internal/spiral/cadence_test.go` — unit tests for allowed candidates, geometry-optimal selection, tie handling, and fixed hourly cadence.
- Modify: `internal/spiral/layout.go` — accepts explicit dots-per-lap internally and exposes cadence-aware layout/disc-radius helpers while retaining `Resolution` wrappers.
- Modify: `internal/spiral/layout_test.go` — tests cadence-aware layout geometry and disc bounds.
- Modify: `internal/spiral/state.go` — stores the selected dots-per-lap value in pipeline state.
- Modify: `internal/spiral/stages.go` — selects cadence after buckets are available and feeds it into layout/disc sizing.
- Modify: `internal/spiral/stages_test.go` — verifies pipeline state selects adaptive daily cadence and keeps hourly fixed.

### Task 1: Select the Daily Cadence

**Files:**
- Create: `internal/spiral/cadence.go`
- Create: `internal/spiral/cadence_test.go`

- [ ] **Step 1: Write failing cadence-selector tests**

```go
func TestDailySpotsPerLapUsesAllowedCadences(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	for _, bucketCount := range []int{1, 14, 28, 100, 365, 730} {
		g.Expect(DailySpotsPerLap(bucketCount, 1920, 1080)).To(
			BeElementOf(14, 28, 42, 56),
		)
	}
}

func TestDailySpotsPerLapMinimizesMidpointGapDifference(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	for _, bucketCount := range []int{28, 100, 365, 730} {
		selected := DailySpotsPerLap(bucketCount, 1920, 1080)
		selectedScore := dailyCadenceScore(bucketCount, 1920, 1080, selected)

		for _, candidate := range dailyCadences {
			g.Expect(selectedScore).To(BeNumerically(
				"<=", dailyCadenceScore(bucketCount, 1920, 1080, candidate),
			))
		}
	}
}

func TestSelectDailyCadencePrefersTwentyEightOnTie(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(selectDailyCadence(map[int]float64{
		14: 0.5, 28: 0.5, 42: 1, 56: 2,
	})).To(Equal(28))
}

func TestSpotsPerLapKeepsHourlyCadence(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(SpotsPerLap(Hourly, 730, 1920, 1080)).To(Equal(24))
}
```

- [ ] **Step 2: Run the selector tests and verify they fail**

Run: `go test ./internal/spiral -run 'Test(DailySpotsPerLap|SelectDailyCadence|SpotsPerLapKeepsHourlyCadence)' -count=1`

Expected: FAIL because `DailySpotsPerLap`, `dailyCadenceScore`, `dailyCadences`, `selectDailyCadence`, and the cadence-aware `SpotsPerLap` function do not exist.

- [ ] **Step 3: Implement the selector**

Create `internal/spiral/cadence.go`:

```go
package spiral

import "math"

var dailyCadences = []int{14, 28, 42, 56}

func SpotsPerLap(resolution Resolution, bucketCount, width, height int) int {
	if resolution != Daily {
		return resolution.SpotsPerLap()
	}

	return DailySpotsPerLap(bucketCount, width, height)
}

func DailySpotsPerLap(bucketCount, width, height int) int {
	scores := make(map[int]float64, len(dailyCadences))
	for _, candidate := range dailyCadences {
		scores[candidate] = dailyCadenceScore(bucketCount, width, height, candidate)
	}

	return selectDailyCadence(scores)
}

func selectDailyCadence(scores map[int]float64) int {
	selected := Daily.SpotsPerLap()
	selectedScore := scores[selected]

	for _, candidate := range dailyCadences {
		if scores[candidate] < selectedScore {
			selected, selectedScore = candidate, scores[candidate]
		}
	}

	return selected
}

func dailyCadenceScore(bucketCount, width, height, spotsPerLap int) float64 {
	params := computeSpiralParams(bucketCount, width, height, spotsPerLap)
	if bucketCount <= 1 || params.b == 0 {
		return math.Inf(1)
	}

	radialGap := 2 * math.Pi * params.b
	midpointRadius := params.a + params.b*params.totalAngle/2
	arcGap := midpointRadius * (2 * math.Pi / float64(spotsPerLap))

	return math.Abs(radialGap - arcGap)
}
```

Add `totalAngle float64` to `spiralParams` and assign it in
`computeSpiralParams`. This permits the selector to measure the same
midpoint geometry as the layout rather than duplicating its formula.

- [ ] **Step 4: Run the selector tests and verify they pass**

Run: `go test ./internal/spiral -run 'Test(DailySpotsPerLap|SelectDailyCadence|SpotsPerLapKeepsHourlyCadence)' -count=1`

Expected: PASS.

- [ ] **Step 5: Commit the selector**

```bash
git add internal/spiral/cadence.go internal/spiral/cadence_test.go internal/spiral/layout.go
git commit -m "feat(spiral): select adaptive daily cadence"
```

### Task 2: Pass the Selected Cadence Through Layout and Pipeline State

**Files:**
- Modify: `internal/spiral/layout.go`
- Modify: `internal/spiral/layout_test.go`
- Modify: `internal/spiral/state.go`
- Modify: `internal/spiral/stages.go`
- Modify: `internal/spiral/stages_test.go`

- [ ] **Step 1: Write failing layout and stage tests**

Add to `internal/spiral/layout_test.go`:

```go
func TestLayoutWithCadenceUsesProvidedSpotsPerLap(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	layout := LayoutWithCadence(makeBuckets(56, Daily), 1920, 1080, 56)
	g.Expect(layout.Nodes[56-1].Angle).To(
		BeNumerically("~", 55*(2*math.Pi/56), 0.001),
	)
}

func TestMaxDiscRadiusWithCadenceUsesProvidedSpotsPerLap(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(MaxDiscRadiusWithCadence(365, 1920, 1080, 56)).To(
		BeNumerically(">", MaxDiscRadiusWithCadence(365, 1920, 1080, 14)),
	)
}
```

Add to `internal/spiral/stages_test.go`:

```go
func TestLayoutStageSelectsAdaptiveDailyCadence(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	common := &stages.CommonState{
		Width: 1920, Height: 1080,
		DrawingBounds: stages.DrawingBounds{MaxX: 1920, MaxY: 1080},
	}
	viz := &spiral.State{
		Resolution: spiral.Daily,
		Buckets:    make([]spiral.TimeBucket, 365),
	}

	g.Expect(spiral.LayoutStage(common, viz)).To(Succeed())
	g.Expect(viz.SpotsPerLap).To(Equal(spiral.DailySpotsPerLap(365, 1920, 1080)))
}

func TestLayoutStageKeepsHourlyCadence(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	common := &stages.CommonState{
		Width: 1920, Height: 1080,
		DrawingBounds: stages.DrawingBounds{MaxX: 1920, MaxY: 1080},
	}
	viz := &spiral.State{
		Resolution: spiral.Hourly,
		Buckets:    make([]spiral.TimeBucket, 720),
	}

	g.Expect(spiral.LayoutStage(common, viz)).To(Succeed())
	g.Expect(viz.SpotsPerLap).To(Equal(24))
}
```

- [ ] **Step 2: Run the layout and stage tests and verify they fail**

Run: `go test ./internal/spiral -run 'Test(LayoutWithCadence|MaxDiscRadiusWithCadence|LayoutStageSelectsAdaptive|LayoutStageKeepsHourly)' -count=1`

Expected: FAIL because `LayoutWithCadence`, `MaxDiscRadiusWithCadence`, and `State.SpotsPerLap` do not exist.

- [ ] **Step 3: Implement cadence-aware geometry and pipeline wiring**

In `internal/spiral/layout.go`, make `computeSpiralParams` accept
`spotsPerLap int`. Keep existing public wrappers and add:

```go
func LayoutWithCadence(
	buckets []TimeBucket,
	width, height, spotsPerLap int,
) SpiralLayout {
	if len(buckets) == 0 {
		return SpiralLayout{}
	}

	nodes := make([]SpiralNode, len(buckets))
	params := computeSpiralParams(len(buckets), width, height, spotsPerLap)
	for i, bucket := range buckets {
		nodes[i] = positionNode(i, bucket, params)
	}

	return newSpiralLayout(nodes, params)
}

func MaxDiscRadiusWithCadence(
	bucketCount, width, height, spotsPerLap int,
) float64 {
	if bucketCount == 0 {
		return defaultDiscRadius
	}

	return computeSpiralParams(bucketCount, width, height, spotsPerLap).maxDisc
}
```

Make `Layout` call `LayoutWithCadence` with `resolution.SpotsPerLap()`, and
make `MaxDiscRadius` call `MaxDiscRadiusWithCadence` with the same fixed
resolution cadence. Extract the existing `SpiralLayout` construction into
`newSpiralLayout(nodes []SpiralNode, params spiralParams)` so both entry points
return the identical layout metadata.

In `internal/spiral/state.go`, add:

```go
SpotsPerLap int
```

after `Resolution`.

In `internal/spiral/stages.go`, replace the existing geometry calls in
`LayoutStage`:

```go
p.SpotsPerLap = SpotsPerLap(p.Resolution, len(p.Buckets), c.Width, availH)
layout := LayoutWithCadence(p.Buckets, c.Width, availH, p.SpotsPerLap)
maxDisc := MaxDiscRadiusWithCadence(
	len(p.Buckets), c.Width, availH, p.SpotsPerLap,
)
```

Keep the bounds offset, disc sizing, label construction, and all rendering
logic unchanged.

- [ ] **Step 4: Run targeted tests and verify they pass**

Run: `go test ./internal/spiral -count=1`

Expected: PASS, including existing layout, disc-size, surface, render, and
stage tests.

- [ ] **Step 5: Run repository validation**

Run: `task test`

Expected: PASS.

Run via an explore subagent: `task lint`

Expected: exit status 0 and no linter issues.

- [ ] **Step 6: Commit the pipeline integration**

```bash
git add internal/spiral/layout.go internal/spiral/layout_test.go \
  internal/spiral/state.go internal/spiral/stages.go \
  internal/spiral/stages_test.go
git commit -m "feat(spiral): apply adaptive cadence to layout"
```
