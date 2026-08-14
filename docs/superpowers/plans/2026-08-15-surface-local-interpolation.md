# Surface Local Interpolation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace global inverse-distance weighting with a compact, globally adaptive reversed-smootherstep kernel and omit mesh geometry outside local data support.

**Architecture:** Add a private interpolation model in `internal/surface` that filters observations, derives one support radius, and evaluates normalized smootherstep weights. Keep the public `Interpolate` signature unchanged, reuse one model throughout each mesh build, and carry a private unsupported flag on generated points so unsupported triangles are removed before rendering.

**Tech Stack:** Go 1.26.1, Gomega, Goldie v2, fogleman/delaunay, gofumpt, Task.

---

## File Map

| File | Responsibility |
| --- | --- |
| `internal/surface/interpolation.go` | Private smootherstep kernel, adaptive radius calculation, interpolation model, and public `Interpolate` wrapper. |
| `internal/surface/interpolation_test.go` | White-box kernel, radius, support, duplicate, and finite-value tests. |
| `internal/surface/types.go` | Remove the obsolete IDW constant and add private point support state. |
| `internal/surface/mesh.go` | Build one interpolation model, assign support to generated points, and omit unsupported triangles. |
| `internal/surface/mesh_test.go` | Update public interpolation behavior and finite-value coverage. |
| `internal/surface/mesh_internal_test.go` | Verify unsupported triangle filtering. |
| `internal/surface/mesh_refinement_test.go` | Adapt private mesh calls and expected-face filtering to the model-aware pipeline. |
| `internal/goldentest/testdata/spiral-surface-*.golden` | Record the four expected PNG/SVG rendering changes. |

No changes are planned for `internal/spiral`, rendering, palettes, CLI/config,
or `samples/`. Spiral already consumes generic `surface.Build` output.

### Task 1: Add The Compact Interpolation Model

**Files:**
- Create: `internal/surface/interpolation.go`
- Create: `internal/surface/interpolation_test.go`
- Modify: `internal/surface/mesh.go:15-43,73-85`
- Modify: `internal/surface/mesh_test.go:11-39,85-116`
- Modify: `internal/surface/types.go:5-12`

- [ ] **Step 1: Replace the public IDW example with smootherstep behavior tests**

In `internal/surface/mesh_test.go`, replace
`TestInterpolate_UsesInverseDistanceWeighting` with a midpoint test. Two
observations four pixels apart derive `R = 8`; the midpoint has equal kernel
weights and must interpolate to `4`:

```go
func TestInterpolate_UsesCompactSmootherstepWeighting(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	originals := []surface.Point{
		{X: 0, Y: 0, Value: 0},
		{X: 4, Y: 0, Value: 8},
	}

	g.Expect(surface.Interpolate(surface.Point{X: 2, Y: 0}, originals)).To(
		gomega.Equal(4.0),
	)
}
```

Keep `TestInterpolate_ReturnsObservedValueAtOriginalLocation`, but remove the
unnecessary `Original: true` fields from its fixture. Exact coordinate matches,
not caller-provided flags, are the measurement-preservation contract.

- [ ] **Step 2: Add failing white-box kernel and radius tests**

Create `internal/surface/interpolation_test.go` in package `surface`. Use the
private APIs that this task will add:

```go
package surface

import (
	"math"
	"testing"

	"github.com/onsi/gomega"
)

func TestSmootherstepWeight_HasSmoothCompactEndpoints(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	g.Expect(smootherstepWeight(0)).To(gomega.Equal(1.0))
	g.Expect(smootherstepWeight(0.5)).To(gomega.Equal(0.5))
	g.Expect(smootherstepWeight(1)).To(gomega.Equal(0.0))
	g.Expect(smootherstepWeight(2)).To(gomega.Equal(0.0))

	const epsilon = 1e-4
	g.Expect(1 - smootherstepWeight(epsilon)).To(gomega.BeNumerically("<", 1e-9))
	g.Expect(smootherstepWeight(1 - epsilon)).To(gomega.BeNumerically("<", 1e-9))
}

func TestNewInterpolationModel_UsesTwiceNearestRankNinetiethPercentile(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	originals := make([]Point, 0, 20)
	for gap := 1; gap <= 10; gap++ {
		base := float64(gap * 100)
		originals = append(originals,
			Point{X: base, Value: float64(gap)},
			Point{X: base + float64(gap), Value: float64(gap)},
		)
	}

	model, ok := newInterpolationModel(originals)
	g.Expect(ok).To(gomega.BeTrue())
	// Distances 1..10 each occur twice; nearest-rank 90% is 9, then multiplied by 2.
	g.Expect(model.radius).To(gomega.Equal(18.0))
}

func TestNewInterpolationModel_SkipsCoincidentAndNonFiniteObservations(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	model, ok := newInterpolationModel([]Point{
		{X: 0, Y: 0, Value: 1},
		{X: 0, Y: 0, Value: 2},
		{X: 4, Y: 0, Value: 3},
		{X: math.NaN(), Y: 0, Value: 4},
		{X: 8, Y: 0, Value: math.Inf(1)},
	})

	g.Expect(ok).To(gomega.BeTrue())
	g.Expect(model.observations).To(gomega.HaveLen(3))
	g.Expect(model.radius).To(gomega.Equal(8.0))
}

func TestNewInterpolationModel_RejectsCoincidentOnlyObservations(t *testing.T) {
	t.Parallel()

	_, ok := newInterpolationModel([]Point{
		{X: 1, Y: 2, Value: 3},
		{X: 1, Y: 2, Value: 4},
	})
	if ok {
		t.Fatal("coincident-only observations must not produce a support radius")
	}
}
```

- [ ] **Step 3: Add failing interpolation support tests**

Append these tests to `internal/surface/interpolation_test.go`:

```go
func TestInterpolationModel_UsesOnlyObservationsInsideRadius(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	model := interpolationModel{
		observations: []Point{
			{X: 0, Y: 0, Value: 2},
			{X: 20, Y: 0, Value: 100},
		},
		radius: 10,
	}

	value, supported := model.interpolate(Point{X: 5, Y: 0})
	g.Expect(supported).To(gomega.BeTrue())
	g.Expect(value).To(gomega.Equal(2.0))
}

func TestInterpolationModel_ReportsUnsupportedPoint(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	model := interpolationModel{
		observations: []Point{{X: 0, Y: 0, Value: 2}},
		radius:       10,
	}

	value, supported := model.interpolate(Point{X: 10, Y: 0})
	g.Expect(supported).To(gomega.BeFalse())
	g.Expect(value).To(gomega.Equal(0.0))
}

func TestInterpolationModel_UsesFirstCoincidentObservation(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	model, ok := newInterpolationModel([]Point{
		{X: 0, Y: 0, Value: 3},
		{X: 0, Y: 0, Value: 7},
		{X: 4, Y: 0, Value: 11},
	})
	g.Expect(ok).To(gomega.BeTrue())

	value, supported := model.interpolate(Point{X: 0, Y: 0})
	g.Expect(supported).To(gomega.BeTrue())
	g.Expect(value).To(gomega.Equal(3.0))
}

func TestInterpolate_ReturnsZeroOutsideSupport(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	value := Interpolate(Point{X: 100, Y: 100}, []Point{
		{X: 0, Y: 0, Value: 2},
		{X: 4, Y: 0, Value: 6},
	})
	g.Expect(value).To(gomega.Equal(0.0))
}
```

- [ ] **Step 4: Run the interpolation tests to verify they fail**

Run:

```bash
go test ./internal/surface -run 'Test(SmootherstepWeight|NewInterpolationModel|InterpolationModel|Interpolate_)' -count=1
```

Expected: compilation fails because `smootherstepWeight`,
`newInterpolationModel`, and `interpolationModel` do not exist; the external
midpoint test still uses the old implementation.

- [ ] **Step 5: Implement the private model and public wrapper**

Create `internal/surface/interpolation.go`:

```go
package surface

import (
	"math"
	"sort"
)

const (
	supportRadiusPercentile = 0.90
	supportRadiusMultiplier = 2.0
)

type interpolationModel struct {
	observations []Point
	radius       float64
}

func newInterpolationModel(originals []Point) (interpolationModel, bool) {
	observations := observedPoints(originals)
	radius, ok := interpolationSupportRadius(observations)
	if !ok {
		return interpolationModel{}, false
	}

	return interpolationModel{observations: observations, radius: radius}, true
}

func interpolationSupportRadius(observations []Point) (float64, bool) {
	distances := make([]float64, 0, len(observations))
	for index, observation := range observations {
		nearest := math.Inf(1)
		for otherIndex, other := range observations {
			if index == otherIndex {
				continue
			}

			distance := Distance(observation, other)
			if distance > 0 && distance < nearest {
				nearest = distance
			}
		}

		if isFinite(nearest) {
			distances = append(distances, nearest)
		}
	}

	if len(distances) == 0 {
		return 0, false
	}

	sort.Float64s(distances)
	rank := int(math.Ceil(supportRadiusPercentile*float64(len(distances)))) - 1
	radius := distances[rank] * supportRadiusMultiplier

	return radius, isFinite(radius) && radius > 0
}

func smootherstepWeight(t float64) float64 {
	if !isFinite(t) || t >= 1 {
		return 0
	}

	if t <= 0 {
		return 1
	}

	return 1 - t*t*t*(t*(t*6-15)+10)
}

func (m interpolationModel) interpolate(point Point) (float64, bool) {
	var weightedValue, totalWeight float64

	for _, observation := range m.observations {
		distance := Distance(point, observation)
		if distance == 0 {
			return observation.Value, true
		}

		weight := smootherstepWeight(distance / m.radius)
		weightedValue += observation.Value * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0, false
	}

	return weightedValue / totalWeight, true
}

// Interpolate estimates a point's value from spatially local observed points.
func Interpolate(point Point, originals []Point) float64 {
	model, ok := newInterpolationModel(originals)
	if !ok {
		return 0
	}

	value, _ := model.interpolate(point)

	return value
}
```

Remove the old `Interpolate` implementation and its `math.Pow` use from
`internal/surface/mesh.go`. Remove `IDWPower` from the constants in
`internal/surface/types.go`.

Update `observedPoints` in `internal/surface/mesh.go` so metric values are
filtered as required by the design:

```go
func observedPoints(originals []Point) []Point {
	observed := make([]Point, 0, len(originals))
	for _, original := range originals {
		if !isFinitePoint(original) || !isFinite(original.Value) {
			continue
		}

		original.Original = true
		observed = append(observed, original)
	}

	return observed
}
```

- [ ] **Step 6: Extend the existing non-finite build test to cover values**

In `TestBuild_IgnoresNonFiniteOriginalCoordinates`, rename it to
`TestBuild_IgnoresNonFiniteOriginals` and add these invalid observations:

```go
{X: 5, Y: 5, Value: math.NaN()},
{X: 6, Y: 6, Value: math.Inf(1)},
```

After the existing coordinate assertions, assert all emitted values are
finite:

```go
g.Expect(math.IsNaN(point.Value) || math.IsInf(point.Value, 0)).To(gomega.BeFalse())
```

- [ ] **Step 7: Format and run the focused tests**

Run:

```bash
gofumpt -w internal/surface/interpolation.go internal/surface/interpolation_test.go internal/surface/mesh.go internal/surface/mesh_test.go internal/surface/types.go
go test ./internal/surface -run 'Test(SmootherstepWeight|NewInterpolationModel|InterpolationModel|Interpolate_|Build_IgnoresNonFinite)' -count=1
```

Expected: PASS.

- [ ] **Step 8: Run the complete surface package**

Run:

```bash
go test ./internal/surface -count=1
```

Expected: PASS. Mesh construction still calls the public wrapper repeatedly in
this intermediate commit; Task 2 replaces that with one reused model.

- [ ] **Step 9: Commit the interpolation model**

```bash
git add internal/surface/interpolation.go internal/surface/interpolation_test.go internal/surface/mesh.go internal/surface/mesh_test.go internal/surface/types.go
git commit -m "feat(surface): add compact interpolation kernel"
```

### Task 2: Propagate Support Through Mesh Construction

**Files:**
- Modify: `internal/surface/types.go:13-18`
- Modify: `internal/surface/interpolation.go`
- Modify: `internal/surface/mesh.go:45-137,176-197`
- Modify: `internal/surface/mesh_internal_test.go`
- Modify: `internal/surface/mesh_refinement_test.go:58-145`

- [ ] **Step 1: Add a failing unsupported-triangle test**

Append to `internal/surface/mesh_internal_test.go`:

```go
func TestRegionTriangles_OmitsTriangleWithUnsupportedVertex(t *testing.T) {
	t.Parallel()

	g := gomega.NewWithT(t)
	points := []Point{
		{X: 0, Y: 0, Value: 1},
		{X: 1, Y: 0, Value: 2, unsupported: true},
		{X: 0, Y: 1, Value: 3},
	}

	triangles, complete := regionTriangles(
		Rect{MinX: 0, MinY: 0, MaxX: 1, MaxY: 1},
		points,
		[]int{0, 1, 2},
	)

	g.Expect(complete).To(gomega.BeTrue())
	g.Expect(triangles).To(gomega.BeEmpty())
}
```

- [ ] **Step 2: Run the new test to verify it fails**

Run:

```bash
go test ./internal/surface -run '^TestRegionTriangles_OmitsTriangleWithUnsupportedVertex$' -count=1
```

Expected: compilation fails because `Point` has no `unsupported` field.

- [ ] **Step 3: Add private support state and a point assignment helper**

Add a private field to `Point` in `internal/surface/types.go`. Naming the
negative state preserves zero-value compatibility for existing point literals:

```go
type Point struct {
	X           float64
	Y           float64
	Value       float64
	Original    bool
	unsupported bool
}
```

Add this helper to `internal/surface/interpolation.go`:

```go
func (m interpolationModel) assign(point Point) Point {
	value, supported := m.interpolate(point)
	point.Value = value
	point.unsupported = !supported

	return point
}
```

- [ ] **Step 4: Build and reuse one model across the mesh pipeline**

Replace the beginning of `Build` in `internal/surface/mesh.go` with:

```go
func Build(region Region, originals []Point, seed uint64) []Triangle {
	if !isValidRegion(region) {
		return nil
	}

	model, ok := newInterpolationModel(originals)
	if !ok || len(model.observations) < 3 {
		return nil
	}

	points, complete := meshPoints(region, model, seed)
	if !complete || len(points) < 3 {
		return nil
	}
```

Keep the existing triangulation and `regionTriangles` tail unchanged.

Change `meshPoints` and `refineMeshPoints` to accept the model and assign
support once per generated point:

```go
func meshPoints(region Region, model interpolationModel, seed uint64) ([]Point, bool) {
	boundary := boundarySamples(region, model.observations)
	for index := range boundary {
		boundary[index] = model.assign(boundary[index])
	}

	points := append([]Point(nil), model.observations...)
	points = append(points, boundary...)
	samplingSources := append([]Point(nil), points...)

	infill := Sample(region, samplingSources, PoissonMinDistance, seed)
	for _, point := range infill {
		points = append(points, model.assign(point))
	}

	return refineMeshPoints(region, points, model)
}

func refineMeshPoints(
	region Region,
	points []Point,
	model interpolationModel,
) ([]Point, bool) {
	limit := refinementPointLimit(region, len(points))

	for {
		triangulation, err := delaunay.Triangulate(delaunayPoints(points))
		if err != nil {
			return nil, false
		}

		candidates, oversized := refinementPoints(region, points, triangulation.Triangles)
		if !oversized {
			return points, true
		}

		if len(candidates) == 0 || len(points)+len(candidates) > limit {
			return nil, false
		}

		for _, candidate := range candidates {
			points = append(points, model.assign(candidate))
		}
	}
}
```

- [ ] **Step 5: Filter unsupported triangles before calculating values**

Add this helper in `internal/surface/mesh.go`:

```go
func triangleIsUnsupported(triangle Triangle) bool {
	for _, point := range triangle.Points {
		if point.unsupported {
			return true
		}
	}

	return false
}
```

In `regionTriangles`, extend the existing early filter:

```go
if !ok || isDegenerateTriangle(triangle) || triangleIsUnsupported(triangle) {
	continue
}
```

Do this before calculating `triangle.Value`. Supported zero values remain
valid because filtering uses the private flag rather than the numeric value.

- [ ] **Step 6: Adapt white-box refinement tests to the new model signature**

In both call sites in `internal/surface/mesh_refinement_test.go`, replace the
direct `observedPoints` argument with:

```go
model, ok := newInterpolationModel(originals)
if !ok {
	t.Fatal("failed to create interpolation model")
}

points, complete := meshPoints(region, model, 42)
```

In `inRegionDelaunayTriangles`, skip unsupported triangles so its expected set
matches `Build`:

```go
if !triangleInRegion(region, triangle) || triangleIsUnsupported(triangle) {
	continue
}
```

- [ ] **Step 7: Run the unsupported-triangle test**

Run:

```bash
gofumpt -w internal/surface/interpolation.go internal/surface/types.go internal/surface/mesh.go internal/surface/mesh_internal_test.go internal/surface/mesh_refinement_test.go
go test ./internal/surface -run '^TestRegionTriangles_OmitsTriangleWithUnsupportedVertex$' -count=1
```

Expected: PASS.

- [ ] **Step 8: Run all surface tests and repair only locality-related expectations**

Run:

```bash
go test ./internal/surface -count=1
```

Expected: PASS. Existing tests for observed values, annular boundaries,
maximum edge length, refinement, deterministic builds, and subdivision remain
green. A failure at this gate must be diagnosed before proceeding; do not
weaken support filtering or add fallback values.

- [ ] **Step 9: Verify spiral surface geometry remains available**

Run:

```bash
go test ./internal/spiral -run 'Surface' -count=1
```

Expected: PASS, including short spirals, annulus constraints, band subdivision,
layer ordering, and fixed-ink fallback.

- [ ] **Step 10: Commit mesh support propagation**

```bash
git add internal/surface/interpolation.go internal/surface/types.go internal/surface/mesh.go internal/surface/mesh_internal_test.go internal/surface/mesh_refinement_test.go
git commit -m "feat(surface): omit unsupported mesh regions"
```

### Task 3: Update And Inspect Spiral Surface Goldens

**Files:**
- Modify: `internal/goldentest/testdata/spiral-surface-shared-png.golden`
- Modify: `internal/goldentest/testdata/spiral-surface-shared-svg.golden`
- Modify: `internal/goldentest/testdata/spiral-surface-distinct-png.golden`
- Modify: `internal/goldentest/testdata/spiral-surface-distinct-svg.golden`

- [ ] **Step 1: Verify only surface goldens mismatch**

Run:

```bash
go test ./internal/goldentest -run '^TestGolden_SpiralSurface(Shared|Distinct)$' -count=1
```

Expected: FAIL with Goldie mismatches for the changed PNG/SVG surface output,
not a panic or pipeline error.

- [ ] **Step 2: Regenerate only the four surface goldens**

Run:

```bash
GOLDIE_UPDATE=1 go test ./internal/goldentest -run '^TestGolden_SpiralSurface(Shared|Distinct)$' -count=1
```

Expected: PASS and exactly four modified files under
`internal/goldentest/testdata/`.

- [ ] **Step 3: Inspect the golden scope and visual output**

Run:

```bash
git status --short internal/goldentest/testdata
git --no-pager diff --stat -- internal/goldentest/testdata/spiral-surface-*.golden
```

Expected: only the four files listed in this task changed. Use the image viewer
on both PNG goldens. Inspect both SVG diffs or render them for comparison.
Confirm:

- nearby peaks blend rather than forming inverse-square spikes;
- broad areas are no longer pulled toward a global mean;
- no visible cutoff rings or step boundaries appear;
- unsupported areas show background rather than fabricated colour;
- track, discs, labels, legends, and annulus boundaries remain intact.

Do not regenerate or stage `samples/`; existing sample modifications belong to
the user and are outside this feature.

- [ ] **Step 4: Re-run the focused golden tests without update mode**

Run:

```bash
go test ./internal/goldentest -run '^TestGolden_SpiralSurface(Shared|Distinct)$' -count=1
```

Expected: PASS.

- [ ] **Step 5: Commit the golden snapshots**

```bash
git add internal/goldentest/testdata/spiral-surface-shared-png.golden internal/goldentest/testdata/spiral-surface-shared-svg.golden internal/goldentest/testdata/spiral-surface-distinct-png.golden internal/goldentest/testdata/spiral-surface-distinct-svg.golden
git commit -m "test(surface): update local interpolation goldens"
```

### Task 4: Final Verification

**Files:**
- Verify only; no planned modifications.

- [ ] **Step 1: Check formatting without rewriting unrelated files**

Run:

```bash
gofumpt -l internal/surface
```

Expected: no output.

- [ ] **Step 2: Run focused packages**

Run:

```bash
go test ./internal/surface ./internal/spiral ./internal/goldentest -count=1
```

Expected: PASS.

- [ ] **Step 3: Run the complete test suite**

Run:

```bash
task test
```

Expected: PASS.

- [ ] **Step 4: Run repository CI through an Explore subagent**

Per `.github/copilot-instructions.md`, dispatch an `Explore` subagent to run
`task ci` and return only the exit status, failing test/linter identities,
offending file and message, or a one-line clean result.

Expected: exit status `0`, no failing tests, and no lint issues. Locally,
`verify-no-changes` may warn about the user's pre-existing sample image and
`.superpowers/` changes; it must not introduce additional source changes.

- [ ] **Step 5: Verify final change scope**

Run:

```bash
git --no-pager diff origin/copilot/enhance-radial-visualization...HEAD --stat
git status --short
```

Expected: feature commits contain the design, plan, surface implementation and
tests, plus four spiral surface goldens. Pre-existing modified sample images
and untracked `.superpowers/` remain unstaged and unchanged.