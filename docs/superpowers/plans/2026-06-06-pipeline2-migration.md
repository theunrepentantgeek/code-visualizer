# Pipeline2 Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the legacy `pipeline.Run` + `pipeline.Stage[S]` model across all five viz commands with the type-keyed `*pipeline.State` model from `pipeline2.go`, and delete the legacy API.

**Architecture:** Each viz pipeline carries exactly three typed values in `*pipeline.State`: `*stages.CommonState`, the per-viz `*config.<Viz>`, and the slimmed per-viz `*State` (with no embedded `CommonState`, no `Config` field, no `Common()`, no `IncludeBinary()`). Stages become plain functions whose parameters spell out the typed values they consume; orchestrators compose them via `pipeline.ApplyFunc*` helpers. `FilterBinaryFiles` and `ApplyCanvasBlockLabels` move out of `internal/stages` into the per-viz packages (since they need concrete viz types).

**Tech Stack:** Go 1.26+, Gomega, `github.com/sebdah/goldie/v2`, `eris`, `task` build runner. Spec: [docs/superpowers/specs/2026-06-06-pipeline2-migration-design.md](../specs/2026-06-06-pipeline2-migration-design.md).

---

## Conventions used by this plan

- **Stage parameter order:** When a stage takes multiple typed values, the order is **always** `(*stages.CommonState, *<viz>.State, *config.<Viz>)`. The compiler does not enforce this; reviewers do.
- **Receiver naming:** stage parameters are named `c` for `*stages.CommonState`, `t`/`b`/`r`/`p`/`x` for the viz state (treemap/bubbletree/radialtree/sPiral/scatter — pick the obvious one and stay consistent within a file), `cfg` for the viz config.
- **Branch:** `improve/migrate-pipeline` (already created).
- **Verification command (lightweight, per task):** `go build ./... && go test ./internal/pipeline/...` (or the package being changed).
- **Verification command (gate, end of plan):** `task ci`. Per repo memory, dispatch `task lint` and `task ci` via the `Explore` subagent and ask it to return exit status + only the failing-lint / failing-test summary.

---

## File map

**`internal/pipeline/`**
- Modify: [pipeline2.go](../../../internal/pipeline/pipeline2.go) — add `ApplyFuncXY`, `ApplyFuncXYZ`, export `Store`.
- Modify: [state.go](../../../internal/pipeline/state.go) — `NewState(values ...any)`; remove unexported `store`.
- Modify: [state_test.go](../../../internal/pipeline/state_test.go) — adapt to variadic `NewState`; add panic-on-nil and panic-on-duplicate-type tests.
- Modify: [pipeline2_test.go](../../../internal/pipeline/pipeline2_test.go) — swap `store(...)` for `Store(...)`; add tests for `ApplyFuncXY`, `ApplyFuncXYZ`.
- Delete: [pipeline.go](../../../internal/pipeline/pipeline.go).
- Delete: [pipeline_test.go](../../../internal/pipeline/pipeline_test.go).

**`internal/stages/`**
- Modify: [common.go](../../../internal/stages/common.go) — delete `VizState` interface.
- Modify: [paths.go](../../../internal/stages/paths.go) — `ValidatePaths(c *CommonState) error`.
- Modify: [filter.go](../../../internal/stages/filter.go) — `BuildFilterRules(c *CommonState) error`.
- Modify: [scan.go](../../../internal/stages/scan.go) — `ScanFilesystem(c *CommonState) error`.
- Modify: [git.go](../../../internal/stages/git.go) — `CheckGitRequirement(c *CommonState) error`.
- Modify: [providers.go](../../../internal/stages/providers.go) — `RunProviders(c *CommonState) error`.
- Modify: [dimensions.go](../../../internal/stages/dimensions.go) — `ResolveDimensions(c *CommonState) error`.
- Modify: [export.go](../../../internal/stages/export.go) — `ExportConfig(c *CommonState) error`, `ExportData(c *CommonState) error`.
- Modify: [canvas.go](../../../internal/stages/canvas.go) — `ApplyFooter(c *CommonState) error`, `WriteCanvas(c *CommonState) error`.
- Modify: [git_history.go](../../../internal/stages/git_history.go) — `LoadGitHistory`, `GroupGitHistoryByFile`, `ExtractFileHistory` all take `c *CommonState`.
- Modify: [binary.go](../../../internal/stages/binary.go) — delete `BinaryFilterToggler` and the generic `FilterBinaryFiles`; expose `FilterBinaryFiles(c *CommonState, include bool) error` (unexported helper becomes the public stage helper).
- Delete: [labels.go](../../../internal/stages/labels.go) — `ApplyCanvasBlockLabels` moves to `internal/treemap`.
- Delete: [labels_test.go](../../../internal/stages/labels_test.go) — its tests move with the function.
- Modify: [paths_test.go](../../../internal/stages/paths_test.go) — `fakeState` deleted, tests now exercise the concrete signatures directly. (`ValidatePathsHelper` tests stay; they were already on the helper, not the stage.)
- Modify: [binary_test.go](../../../internal/stages/binary_test.go) — delete `fakeBinaryState`; rewrite tests to call `FilterBinaryFiles(c, include)` directly with a `*CommonState`.
- Modify: [canvas_test.go](../../../internal/stages/canvas_test.go) — drop `[*fakeState]` type params; call `ApplyFooter(c)` / `WriteCanvas(c)` with a `*CommonState`.

**`internal/treemap/`**
- Modify: [state.go](../../../internal/treemap/state.go) — slim per spec (drop embedded `CommonState`, `Config` field, `Common()`, `IncludeBinary()`, `CanvasLabels()`).
- Modify: [stages.go](../../../internal/treemap/stages.go) — new signatures; delete `var _ pipeline.Stage[*State] = ...` block.
- Create: [labels_stage.go](../../../internal/treemap/labels_stage.go) — `ApplyCanvasBlockLabels(c *stages.CommonState, t *State) error`. Also add `FilterBinaryFiles(c *stages.CommonState, t *State) error` here (one-liner that calls `stages.FilterBinaryFiles(c, t.IncludeBinaryFiles)`).
- Create: [labels_stage_test.go](../../../internal/treemap/labels_stage_test.go) — relocated `ApplyCanvasBlockLabels` test content.
- Modify: [stages_test.go](../../../internal/treemap/stages_test.go) — split combined-state literals; delete the `BinaryFilterToggler` assertion and `TestState_CommonReturnsEmbeddedPointer` and `TestState_IncludeBinary` tests; add a tiny test for the new `treemap.FilterBinaryFiles`.

**`internal/bubbletree/`, `internal/radialtree/`, `internal/spiral/`, `internal/scatter/`** — same pattern as treemap (minus the `BlockLabels` work):
- Modify each `state.go` (slim).
- Modify each `stages.go` (new signatures; drop assertion block).
- Create per-viz `binary_filter.go` (one-liner `FilterBinaryFiles`) and `binary_filter_test.go`.
- Modify each `stages_test.go` (split state literals, delete `BinaryFilterToggler` assertions, update `Common()`/`Config` accesses).

**`cmd/codeviz/`**
- Modify: [treemap_cmd.go](../../../cmd/codeviz/treemap_cmd.go), [bubbletree_cmd.go](../../../cmd/codeviz/bubbletree_cmd.go), [radialtree_cmd.go](../../../cmd/codeviz/radialtree_cmd.go), [spiral_cmd.go](../../../cmd/codeviz/spiral_cmd.go), [scatter_cmd.go](../../../cmd/codeviz/scatter_cmd.go) — rewrite each `Run` to use `pipeline.NewState(common, cfg, viz) + pipeline.ApplyFunc*`.

---

## Task 1: Add `Store`, `ApplyFuncXY`, `ApplyFuncXYZ`; variadic `NewState`

**Files:**
- Modify: `internal/pipeline/state.go`
- Modify: `internal/pipeline/pipeline2.go`
- Modify: `internal/pipeline/state_test.go`
- Modify: `internal/pipeline/pipeline2_test.go`

- [ ] **Step 1: Write failing tests for variadic `NewState`**

Append to `internal/pipeline/state_test.go`:

```go
func TestNewState_GivenMultipleValues_StoresAll(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	k := Kind{name: "k"}
	c := Color{name: "c"}

	state := NewState(k, c)

	kv, kok := Lookup[Kind](state)
	cv, cok := Lookup[Color](state)
	g.Expect(kok).To(BeTrue())
	g.Expect(cok).To(BeTrue())
	g.Expect(kv).To(Equal(k))
	g.Expect(cv).To(Equal(c))
}

func TestNewState_GivenNilValue_Panics(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(func() { NewState(nil) }).To(Panic())
}

func TestNewState_GivenDuplicateType_Panics(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	a := Kind{name: "a"}
	b := Kind{name: "b"}

	g.Expect(func() { NewState(a, b) }).To(Panic())
}

func TestStore_StoresValue(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	state := NewState()
	Store(state, Kind{name: "x"})

	v, ok := Lookup[Kind](state)
	g.Expect(ok).To(BeTrue())
	g.Expect(v.name).To(Equal("x"))
}
```

Existing tests in this file still pass single values to `NewState`. They'll be updated below as part of changing the API; for now they will fail to compile against the new signature.

- [ ] **Step 2: Update existing tests in `state_test.go` to use variadic API**

In `internal/pipeline/state_test.go`, every existing call of the form `NewState(alpha)` (where `alpha` is a value) already works under the new signature unchanged — variadic accepts one arg. The only test that needs editing is `TestState_Store_WhenValuePresent_OverwritesValue`: replace its internal `store(state, beta)` (if present) with `Store(state, beta)`.

Run: `grep -n "store(" internal/pipeline/state_test.go` and replace each lowercase `store(` with `Store(`.

- [ ] **Step 3: Write failing tests for `ApplyFuncXY` and `ApplyFuncXYZ`**

Append to `internal/pipeline/pipeline2_test.go`:

```go
/*
 * ApplyFuncXY Tests (in-place mutation, no return value)
 */

func Test_ApplyFuncXY_WhenStateMissingX_Panics(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	var k Kind
	state := NewState(k)

	g.Expect(func() {
		ApplyFuncXY(state, func(Color, Kind) error { return nil })
	}).To(PanicWith(ContainSubstring("Color")))
}

func Test_ApplyFuncXY_WhenBothPresent_CallsFunc(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	state := NewState(Kind{name: "k"}, Color{name: "c"})
	called := false

	ApplyFuncXY(state, func(Kind, Color) error {
		called = true
		return nil
	})
	g.Expect(state.Err()).ToNot(HaveOccurred())
	g.Expect(called).To(BeTrue())
}

func Test_ApplyFuncXY_WhenFuncReturnsError_SetsStateErr(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	state := NewState(Kind{}, Color{})
	ApplyFuncXY(state, func(Kind, Color) error { return errors.New("boom") })
	g.Expect(state.Err()).To(MatchError(ContainSubstring("boom")))
}

func Test_ApplyFuncXY_WhenAlreadyErrored_ShortCircuits(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	state := NewState(Kind{}, Color{})
	state.setErr(errors.New("prior"))

	called := false
	ApplyFuncXY(state, func(Kind, Color) error { called = true; return nil })
	g.Expect(called).To(BeFalse())
}

/*
 * ApplyFuncXYZ Tests
 */

func Test_ApplyFuncXYZ_WhenStateMissingZ_Panics(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	state := NewState(Kind{}, Color{})

	g.Expect(func() {
		ApplyFuncXYZ(state, func(Kind, Color, Texture) error { return nil })
	}).To(PanicWith(ContainSubstring("Texture")))
}

func Test_ApplyFuncXYZ_WhenAllPresent_CallsFunc(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	state := NewState(Kind{name: "k"}, Color{name: "c"}, Texture{name: "t"})
	called := false

	ApplyFuncXYZ(state, func(Kind, Color, Texture) error {
		called = true
		return nil
	})
	g.Expect(state.Err()).ToNot(HaveOccurred())
	g.Expect(called).To(BeTrue())
}

func Test_ApplyFuncXYZ_WhenFuncReturnsError_SetsStateErr(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	state := NewState(Kind{}, Color{}, Texture{})
	ApplyFuncXYZ(state, func(Kind, Color, Texture) error { return errors.New("boom") })
	g.Expect(state.Err()).To(MatchError(ContainSubstring("boom")))
}
```

Also in `internal/pipeline/pipeline2_test.go`, replace every occurrence of `store(state, k)` (or other lowercase `store(`) with `Store(state, k)`. There is currently one such call in `Test_ApplyFuncXYR_WhenStateContainsXAndY_StoresResultInState`.

- [ ] **Step 4: Run the new tests; confirm they fail to compile**

Run: `cd /home/bevan/github/code-visualizer && go test ./internal/pipeline/...`
Expected: compile errors — `Store` undefined, `ApplyFuncXY` undefined, `ApplyFuncXYZ` undefined, `NewState` arity mismatch.

- [ ] **Step 5: Replace `NewState` and add `Store`**

Edit `internal/pipeline/state.go`. Replace the file's body so it reads:

```go
package pipeline

import "reflect"

// State is a simple key-value store where the key is the type of the value.
type State struct {
	content map[reflect.Type]any
	err     error
}

// NewState creates a new State pre-populated with the given values. Each
// value is keyed by its dynamic Go type. Panics if any value is nil (no
// usable type information) or if the same type is supplied twice.
func NewState(values ...any) *State {
	s := &State{content: map[reflect.Type]any{}}
	for _, v := range values {
		if v == nil {
			panic("pipeline.NewState: nil value has no type")
		}
		key := reflect.TypeOf(v)
		if _, exists := s.content[key]; exists {
			panic("pipeline.NewState: duplicate value for type " + key.String())
		}
		s.content[key] = v
	}
	return s
}

// Lookup retrieves a value of type S from the state.
// It returns the value and a boolean indicating whether the value was found.
func Lookup[S any](s *State) (S, bool) {
	var zero S

	key := keyOf[S]()
	if v, ok := s.content[key]; ok {
		//nolint:revive // Invariant is that this value will be of type S
		return v.(S), true
	}

	return zero, false
}

// Store saves a value of type S in the state, overwriting any existing
// value of the same type.
func Store[S any](s *State, value S) {
	key := keyOf[S]()
	s.content[key] = value
}

// keyOf returns a key to use for the specified type.
func keyOf[T any]() reflect.Type {
	return reflect.TypeFor[T]()
}

// setErr sets the error in the state. This is used to store any error that occurred during pipeline execution.
func (s *State) setErr(err error) {
	s.err = err
}

// Err returns the error stored in the state, if any.
func (s *State) Err() error {
	return s.err
}
```

- [ ] **Step 6: Add `ApplyFuncXY` and `ApplyFuncXYZ`**

Append to `internal/pipeline/pipeline2.go`:

```go
// ApplyFuncXY updates pipeline state by applying an error-returning function
// that consumes two typed inputs and mutates them in place. Panics if either
// input type is absent from the state. Short-circuits when state already
// holds an error.
func ApplyFuncXY[X any, Y any](
	s *State,
	f func(X, Y) error,
) {
	if s.Err() != nil {
		return
	}

	vx, ok := Lookup[X](s)
	if !ok {
		panic(fmt.Sprintf("state does not contain value of type %s", keyOf[X]()))
	}

	vy, ok := Lookup[Y](s)
	if !ok {
		panic(fmt.Sprintf("state does not contain value of type %s", keyOf[Y]()))
	}

	if err := f(vx, vy); err != nil {
		s.setErr(err)
	}
}

// ApplyFuncXYZ is the three-input variant of ApplyFuncXY.
func ApplyFuncXYZ[X any, Y any, Z any](
	s *State,
	f func(X, Y, Z) error,
) {
	if s.Err() != nil {
		return
	}

	vx, ok := Lookup[X](s)
	if !ok {
		panic(fmt.Sprintf("state does not contain value of type %s", keyOf[X]()))
	}

	vy, ok := Lookup[Y](s)
	if !ok {
		panic(fmt.Sprintf("state does not contain value of type %s", keyOf[Y]()))
	}

	vz, ok := Lookup[Z](s)
	if !ok {
		panic(fmt.Sprintf("state does not contain value of type %s", keyOf[Z]()))
	}

	if err := f(vx, vy, vz); err != nil {
		s.setErr(err)
	}
}
```

- [ ] **Step 7: Run the new tests; confirm they pass**

Run: `cd /home/bevan/github/code-visualizer && go test ./internal/pipeline/...`
Expected: PASS (existing `pipeline_test.go` for `Run`/`Stage` continues to pass because we have not deleted them yet).

- [ ] **Step 8: Commit**

```bash
cd /home/bevan/github/code-visualizer
git add internal/pipeline/state.go internal/pipeline/state_test.go \
        internal/pipeline/pipeline2.go internal/pipeline/pipeline2_test.go
git commit -m "feat(pipeline): add Store, ApplyFuncXY/XYZ, variadic NewState"
```

---

## Task 2: Slim the per-viz state types

This task strips `CommonState`, `Config`, `Common()`, `IncludeBinary()`, and (for treemap) `CanvasLabels()` from each viz state. After this task, the per-viz packages will not compile — that is fixed in Task 3. **Do not run tests between Task 2 and Task 3.** Commit at the end of Task 3.

**Files:**
- Modify: `internal/treemap/state.go`
- Modify: `internal/bubbletree/state.go`
- Modify: `internal/radialtree/state.go`
- Modify: `internal/spiral/state.go`
- Modify: `internal/scatter/state.go`

- [ ] **Step 1: Slim `internal/treemap/state.go`**

Replace the file body with:

```go
package treemap

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// State is the viz-specific pipeline state for the treemap visualization.
// Shared state lives in *stages.CommonState; treemap config in *config.Treemap.
type State struct {
	IncludeBinaryFiles bool
	Flat               bool

	Size          metric.Name
	FillMetric    metric.Name
	FillPalette   palette.PaletteName
	BorderMetric  metric.Name
	BorderPalette palette.PaletteName
	Inks          Inks
	Root          TreemapRectangle
	LegendConfig  *canvas.LegendConfig
	BlockLabels   []canvas.BlockLabel
}
```

- [ ] **Step 2: Slim `internal/bubbletree/state.go`**

Replace the file body with:

```go
package bubbletree

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// State is the viz-specific pipeline state for the bubbletree visualization.
type State struct {
	IncludeBinaryFiles bool
	Flat               bool

	Size          metric.Name
	FillMetric    metric.Name
	FillPalette   palette.PaletteName
	BorderMetric  metric.Name
	BorderPalette palette.PaletteName
	Labels        LabelMode
	Inks          Inks
	Nodes         BubbleNode
	LegendConfig  *canvas.LegendConfig
}
```

- [ ] **Step 3: Slim `internal/radialtree/state.go`**

Replace the file body with:

```go
package radialtree

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// State is the viz-specific pipeline state for the radial tree visualization.
type State struct {
	IncludeBinaryFiles bool

	DiscSize      metric.Name
	FillMetric    metric.Name
	FillPalette   palette.PaletteName
	BorderMetric  metric.Name
	BorderPalette palette.PaletteName
	Labels        LabelMode
	Inks          Inks
	Nodes         RadialNode
	LegendConfig  *canvas.LegendConfig
}
```

- [ ] **Step 4: Slim `internal/spiral/state.go`**

Replace the file body with:

```go
package spiral

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// State is the viz-specific pipeline state for the spiral visualization.
type State struct {
	IncludeBinaryFiles bool

	Size          metric.Name
	FillMetric    metric.Name
	FillPalette   palette.PaletteName
	BorderMetric  metric.Name
	BorderPalette palette.PaletteName
	Resolution    Resolution
	Labels        LabelMode

	Buckets      []TimeBucket
	Inks         Inks
	Layout       SpiralLayout
	LegendConfig *canvas.LegendConfig
}
```

- [ ] **Step 5: Slim `internal/scatter/state.go`**

Replace the file body with:

```go
package scatter

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

// State is the viz-specific pipeline state for the scatter visualization.
type State struct {
	IncludeBinaryFiles bool

	XAxis         AxisSpec
	YAxis         AxisSpec
	Size          metric.Name
	FillMetric    metric.Name
	FillPalette   palette.PaletteName
	BorderMetric  metric.Name
	BorderPalette palette.PaletteName

	Dataset      Dataset
	Inks         Inks
	Layout       ScatterLayout
	LegendConfig *canvas.LegendConfig
}
```

- [ ] **Step 6: Do not run `go build` yet — proceed to Task 3**

The viz packages and `cmd/codeviz` no longer compile because their stages still reference `s.Common()`, `s.Config`, etc. Task 3 fixes the stage signatures.

---

## Task 3: Rewrite viz `stages.go` for treemap

**Files:**
- Modify: `internal/treemap/stages.go`
- Create: `internal/treemap/labels_stage.go`
- Create: `internal/treemap/binary_filter.go`

- [ ] **Step 1: Rewrite `internal/treemap/stages.go`**

Replace the file body with:

```go
package treemap

import (
	"log/slog"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ResolveMetrics resolves size, fill, and border metrics + palettes and fills
// c.Requested.
func ResolveMetrics(c *stages.CommonState, t *State, cfg *config.Treemap) error {
	t.Size = metric.Name(stages.PtrString(cfg.Size))
	t.FillMetric = resolveFillMetric(cfg)
	t.FillPalette = stages.ResolveFillPalette(cfg.Fill, t.FillMetric)
	t.BorderMetric, t.BorderPalette = stages.ResolveBorderMetricAndPalette(cfg.Border)

	c.Requested = stages.CollectRequestedMetrics(t.Size, cfg.Fill, cfg.Border)

	return nil
}

func resolveFillMetric(cfg *config.Treemap) metric.Name {
	if fill := cfg.Fill.MetricName(); fill != "" {
		return fill
	}

	return metric.Name(stages.PtrString(cfg.Size))
}

// BuildInksStage builds the treemap inks. Also emits the "Rendering image"
// log line preserved from the legacy renderAndLog helper.
func BuildInksStage(c *stages.CommonState, t *State) error {
	slog.Info("Rendering image", "output", c.Output, "width", c.Width, "height", c.Height)

	t.Inks = BuildInks(c.Root, t.FillMetric, t.FillPalette, t.BorderMetric, t.BorderPalette)
	if !t.Flat {
		t.Inks.Fill = canvas.NewRadialGradientInk(t.Inks.Fill)
	}

	return nil
}

// BuildLegendStage builds the legend config from inks.
func BuildLegendStage(c *stages.CommonState, t *State, cfg *config.Treemap) error {
	_ = c
	pos, orient := legend.ResolveOptions(
		stages.PtrString(cfg.Legend),
		stages.PtrString(cfg.LegendOrientation),
	)

	t.LegendConfig = legend.Build(
		pos, orient,
		t.Inks.Fill, t.FillMetric,
		t.Inks.Border, t.BorderMetric,
		t.Size,
	)
	if t.LegendConfig != nil {
		t.LegendConfig.LabelSample = labelSampleLines(labelMetricsFor(t, cfg))
	}

	return nil
}

// LayoutStage reserves legend space, lays out rectangles, and applies the
// resulting offset.
func LayoutStage(c *stages.CommonState, t *State) error {
	availH := c.Height - stages.EffectiveFooterHeight(c.RootConfig)
	layoutW, layoutH := legend.ReserveAndLayout(t.LegendConfig, c.Width, availH)

	rect := Layout(c.Root, layoutW, layoutH, t.Size)

	if layoutW < c.Width || layoutH < availH {
		if t.LegendConfig != nil {
			wReduce, hReduce := t.LegendConfig.ReserveSpace()
			dx, dy := legend.LayoutOffset(t.LegendConfig, wReduce, hReduce)
			OffsetRects(&rect, dx, dy)
		}
	}

	t.Root = rect

	return nil
}

// RenderStage renders the treemap to a canvas and attaches the legend.
func RenderStage(c *stages.CommonState, t *State) error {
	cv := RenderToCanvas(t.Root, c.Root, c.Width, c.Height, t.Inks, t.Size)
	if t.LegendConfig != nil {
		cv.SetLegend(*t.LegendConfig)
	}

	slog.Debug("rendering", "width", c.Width, "height", c.Height, "output", c.Output)

	c.Canvas = cv

	return nil
}

// LabelStage builds the reusable block labels for treemap file rectangles.
func LabelStage(c *stages.CommonState, t *State, cfg *config.Treemap) error {
	t.BlockLabels = buildBlockLabels(t.Root, c.Root, t.Inks.Fill, labelMetricsFor(t, cfg))

	return nil
}

func labelMetricsFor(t *State, cfg *config.Treemap) LabelMetrics {
	return LabelMetrics{
		Size:   t.Size,
		Fill:   cfg.Fill.MetricName(),
		Border: cfg.Border.MetricName(),
	}
}

// LogResult logs the final summary.
func LogResult(c *stages.CommonState, t *State) error {
	files, dirs := stages.CountAll(c.Root)

	slog.Info(
		"Rendered treemap",
		"files", files,
		"directories", dirs,
		"output", c.Output,
		"width", c.Width,
		"height", c.Height,
		"size_metric", string(t.Size),
		"fill_metric", string(t.FillMetric),
		"fill_palette", string(t.FillPalette),
		"border_metric", string(t.BorderMetric),
		"border_palette", string(t.BorderPalette),
	)

	return nil
}
```

Note: the `var ( _ pipeline.Stage[*State] = ... )` block is gone, and the `pipeline` import drops out. The `_ = c` in `BuildLegendStage` exists because that stage genuinely doesn't read from `CommonState` but the orchestrator's `ApplyFuncXYZ` requires the parameter — keep the parameter so the signature pattern is uniform across viz stages.

- [ ] **Step 2: Create `internal/treemap/labels_stage.go`**

```go
package treemap

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ApplyCanvasBlockLabels fits and adds the treemap's block labels to c.Canvas.
// No-op when c.Canvas is nil.
func ApplyCanvasBlockLabels(c *stages.CommonState, t *State) error {
	if c.Canvas == nil {
		return nil
	}

	format, err := canvas.FormatFromPath(c.Output)
	if err != nil {
		return eris.Wrap(err, "resolve canvas label format")
	}

	for _, label := range t.BlockLabels {
		c.Canvas.AddBlockLabel(canvas.LayerOverlay, label, format)
	}

	return nil
}
```

- [ ] **Step 3: Create `internal/treemap/binary_filter.go`**

```go
package treemap

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// FilterBinaryFiles strips binary files from c.Root unless the treemap state
// requests they be kept.
func FilterBinaryFiles(c *stages.CommonState, t *State) error {
	return stages.FilterBinaryFiles(c, t.IncludeBinaryFiles)
}
```

(`stages.FilterBinaryFiles` becomes a concrete two-arg function in Task 4.)

- [ ] **Step 4: Do not compile-check yet — proceed to Task 4**

`stages.FilterBinaryFiles` and several other shared stages still have their old signatures; the workspace as a whole won't build until Task 4 completes.

---

## Task 4: Rewrite the shared stages

**Files:**
- Modify: `internal/stages/common.go`
- Modify: `internal/stages/paths.go`
- Modify: `internal/stages/filter.go`
- Modify: `internal/stages/scan.go`
- Modify: `internal/stages/git.go`
- Modify: `internal/stages/providers.go`
- Modify: `internal/stages/dimensions.go`
- Modify: `internal/stages/export.go`
- Modify: `internal/stages/canvas.go`
- Modify: `internal/stages/git_history.go`
- Modify: `internal/stages/binary.go`
- Delete: `internal/stages/labels.go`

- [ ] **Step 1: Delete `VizState` from `internal/stages/common.go`**

Open `internal/stages/common.go` and delete the entire `VizState` interface declaration (the trailing block):

```go
// VizState is satisfied by any state type that embeds CommonState and
// exposes it via a Common() method. Shared stages are generic over this
// interface; in practice the type argument is always a pointer type
// (e.g. *treemap.State).
type VizState interface {
	Common() *CommonState
}
```

- [ ] **Step 2: Rewrite `internal/stages/paths.go`**

Replace the body of `ValidatePaths` and remove the `pipeline` import / assertion:

```go
package stages

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
)

// ValidatePathsHelper validates the target directory and output file paths.
// Returns *TargetPathError or *OutputPathError on failure.
func ValidatePathsHelper(targetPath, output string) error {
	if _, err := canvas.FormatFromPath(output); err != nil {
		return &OutputPathError{Msg: err.Error()}
	}

	info, err := os.Stat(targetPath)
	if err != nil {
		if os.IsNotExist(err) {
			return &TargetPathError{Msg: "target path does not exist: " + targetPath}
		}

		return &TargetPathError{Msg: fmt.Sprintf("cannot access target path: %s", err)}
	}

	if !info.IsDir() {
		return &TargetPathError{Msg: "target path is not a directory: " + targetPath}
	}

	outDir := filepath.Dir(output)
	if outDir == "." {
		return nil
	}

	info, err = os.Stat(outDir)
	if err != nil {
		return &OutputPathError{Msg: "output directory does not exist: " + outDir}
	}

	if !info.IsDir() {
		return &OutputPathError{Msg: "output parent is not a directory: " + outDir}
	}

	return nil
}

// ValidatePaths validates c.TargetPath and c.Output.
func ValidatePaths(c *CommonState) error {
	if err := ValidatePathsHelper(c.TargetPath, c.Output); err != nil {
		return eris.Wrap(err, "invalid paths")
	}

	return nil
}
```

- [ ] **Step 3: Rewrite `internal/stages/filter.go`**

```go
package stages

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
)

// BuildFilterRulesHelper merges config-file filter rules with CLI filter
// flags. CLI filters must already have been syntax-validated by the
// command's Validate() method.
func BuildFilterRulesHelper(cfg *config.Config, cliFilters []filter.Rule) []filter.Rule {
	rules := make([]filter.Rule, 0, len(cfg.FileFilter)+len(cliFilters))
	rules = append(rules, cfg.FileFilter...)
	rules = append(rules, cliFilters...)

	return rules
}

// BuildFilterRules populates c.FilterRules from c.RootConfig.FileFilter plus
// c.CLIFilters.
func BuildFilterRules(c *CommonState) error {
	c.FilterRules = BuildFilterRulesHelper(c.RootConfig, c.CLIFilters)

	return nil
}
```

- [ ] **Step 4: Rewrite `internal/stages/scan.go`**

```go
package stages

import (
	"log/slog"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/scan"
)

// ScanFilesystem walks c.TargetPath, populates c.Root, and wires progress
// reporting based on Flags verbosity.
func ScanFilesystem(c *CommonState) error {
	slog.Info("Scanning filesystem", "path", c.TargetPath)

	scanProg, stopScanTicker := BuildScanProgress(c.Flags)

	root, err := scan.Scan(c.TargetPath, c.FilterRules, scanProg)

	stopScanTicker()

	if err != nil {
		return eris.Wrap(err, "scan failed")
	}

	c.Root = root

	return nil
}
```

- [ ] **Step 5: Rewrite `internal/stages/git.go`**

Update `CheckGitRequirement` and drop the assertion:

```go
// CheckGitRequirement wraps CheckGitRequirementHelper.
func CheckGitRequirement(c *CommonState) error {
	return CheckGitRequirementHelper(c.TargetPath, c.Requested)
}
```

Remove the `pipeline` import and the `var _ pipeline.Stage[...]` line.

- [ ] **Step 6: Rewrite `internal/stages/providers.go`**

```go
package stages

import (
	"log/slog"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// RunProviders calculates c.Requested metrics against c.Root.
func RunProviders(c *CommonState) error {
	slog.Info("Calculating metrics")

	metricProg, stopMetricTicker := BuildMetricProgress(c.Flags, model.CountFiles(c.Root))

	if err := provider.Run(c.Root, c.Requested, metricProg); err != nil {
		stopMetricTicker()

		return eris.Wrap(err, "failed to load metrics")
	}

	stopMetricTicker()

	return nil
}
```

- [ ] **Step 7: Rewrite `internal/stages/dimensions.go`**

Replace the file body:

```go
package stages

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
)

// PtrInt safely dereferences *int, returning fallback if nil.
func PtrInt(p *int, fallback int) int {
	if p == nil {
		return fallback
	}

	return *p
}

// PtrString safely dereferences *string, returning "" if nil.
func PtrString(p *string) string {
	if p == nil {
		return ""
	}

	return *p
}

// ResolveDimensions populates c.Width and c.Height from RootConfig, applying
// the documented defaults (1920x1080).
func ResolveDimensions(c *CommonState) error {
	var imageSize *config.ImageSize
	if c.RootConfig != nil {
		imageSize = c.RootConfig.ImageSize
	}

	var width, height *int
	if imageSize != nil {
		width = imageSize.Width
		height = imageSize.Height
	}

	c.Width = PtrInt(width, 1920)
	c.Height = PtrInt(height, 1080)

	return nil
}
```

- [ ] **Step 8: Rewrite `internal/stages/export.go`**

```go
package stages

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/export"
)

// ExportConfig writes the merged effective config to disk when
// Flags.ExportConfig is non-empty.
func ExportConfig(c *CommonState) error {
	if c.Flags.ExportConfig == "" {
		return nil
	}

	exportCfg := c.RootConfig.ForExport(c.VizName)
	if err := exportCfg.Save(c.Flags.ExportConfig); err != nil {
		return eris.Wrap(err, "failed to save config")
	}

	return nil
}

// ExportData writes computed metric data to disk when Flags.ExportData is
// non-empty.
func ExportData(c *CommonState) error {
	if err := export.Export(c.Root, c.Requested, c.Flags.ExportData); err != nil {
		return eris.Wrap(err, "failed to export data")
	}

	return nil
}
```

- [ ] **Step 9: Rewrite `internal/stages/canvas.go`**

Replace the `WriteCanvas` and `ApplyFooter` definitions (keep `EffectiveFooterHeight` and other helpers intact):

```go
package stages

import (
	"strings"
	"time"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
)

// WriteCanvas writes c.Canvas to c.Output.
func WriteCanvas(c *CommonState) error {
	if err := c.Canvas.Render(c.Output); err != nil {
		return eris.Wrap(err, "render failed")
	}

	return nil
}

// ApplyFooter sets the footer on c.Canvas from RootConfig.Footer.
// If the Footer is hidden, the canvas footer is left unset.
// If the Footer is nil or has no explicit text, the built-in default text is used.
func ApplyFooter(c *CommonState) error {
	if c.Canvas == nil || c.RootConfig == nil {
		return nil
	}

	footer := c.RootConfig.Footer
	if !footer.ShowFooter() {
		return nil
	}

	now := time.Now()
	rep := strings.NewReplacer(
		"$date", now.Format(time.DateOnly),
		"$time", now.Format(time.TimeOnly),
	)

	text := rep.Replace(*footer.Text)
	c.Canvas.SetFooter(text)

	return nil
}

// EffectiveFooterHeight returns the number of pixels that the footer occupies
// when rendered. Returns 0 when cfg is nil or the footer is not shown.
func EffectiveFooterHeight(cfg *config.Config) int {
	if cfg == nil {
		return 0
	}

	if !cfg.Footer.ShowFooter() {
		return 0
	}

	return int(canvas.FooterReservedHeight)
}
```

- [ ] **Step 10: Rewrite `internal/stages/git_history.go` signatures**

For each of `LoadGitHistory`, `GroupGitHistoryByFile`, `ExtractFileHistory`: change the signature from `func Xxx[S VizState](s S) error` to `func Xxx(c *CommonState) error` and remove the `c := s.Common()` line at the top of each function. Drop the `pipeline` import if it becomes unused.

For example, `LoadGitHistory` becomes:

```go
func LoadGitHistory(c *CommonState) error {
	repoRoot, err := git.RepoRootFor(c.Root.Path)
	// ... unchanged body ...
}
```

Do the same edits for `GroupGitHistoryByFile` and `ExtractFileHistory`. Remove any trailing `var _ pipeline.Stage[VizState] = ...` lines if present.

- [ ] **Step 11: Rewrite `internal/stages/binary.go`**

Replace the file body with:

```go
package stages

import (
	"log/slog"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/scan"
)

// CountAll returns the cumulative file and directory counts under root.
func CountAll(node *model.Directory) (files int, dirs int) {
	files = len(node.Files)
	for _, d := range node.Dirs {
		dirs++
		f, d2 := CountAll(d)
		files += f
		dirs += d2
	}

	return files, dirs
}

// FilterBinaryFilesHelper removes binary files from the tree in place.
// Returns *NoFilesAfterFilterError if nothing remains.
func FilterBinaryFilesHelper(root *model.Directory) error {
	beforeCount, _ := CountAll(root)
	filtered := scan.FilterBinaryFiles(root)
	afterCount, _ := CountAll(filtered)
	excluded := beforeCount - afterCount
	slog.Debug("binary file filter", "excluded", excluded, "remaining", afterCount)

	if afterCount == 0 {
		return &NoFilesAfterFilterError{Msg: NoFilesAfterFilterMsg}
	}

	root.Files = filtered.Files
	root.Dirs = filtered.Dirs

	return nil
}

// FilterBinaryFiles removes binary files from c.Root in place unless include
// is true. Per-viz adapter functions call this with t.IncludeBinaryFiles.
func FilterBinaryFiles(c *CommonState, include bool) error {
	if include {
		return nil
	}

	return FilterBinaryFilesHelper(c.Root)
}
```

- [ ] **Step 12: Delete `internal/stages/labels.go`**

Run: `rm /home/bevan/github/code-visualizer/internal/stages/labels.go`

The function moves into `internal/treemap/labels_stage.go` (created in Task 3 Step 2).

- [ ] **Step 13: Compile check**

Run: `cd /home/bevan/github/code-visualizer && go build ./internal/stages/... ./internal/treemap/...`
Expected: SUCCESS for both packages. The other viz packages will not yet build (their stages still reference `s.Common()`); that is the next task.

---

## Task 5: Rewrite viz `stages.go` for bubbletree

**Files:**
- Modify: `internal/bubbletree/stages.go`
- Create: `internal/bubbletree/binary_filter.go`

- [ ] **Step 1: Rewrite `internal/bubbletree/stages.go`**

Replace the file body with:

```go
package bubbletree

import (
	"log/slog"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ResolveMetrics resolves size/fill/border metrics + palettes plus label mode
// and populates c.Requested.
func ResolveMetrics(c *stages.CommonState, b *State, cfg *config.Bubbletree) error {
	b.Size = metric.Name(stages.PtrString(cfg.Size))
	b.FillMetric = resolveFillMetric(cfg, b.Size)
	b.FillPalette = stages.ResolveFillPalette(cfg.Fill, b.FillMetric)
	b.BorderMetric, b.BorderPalette = stages.ResolveBorderMetricAndPalette(cfg.Border)
	b.Labels = resolveLabels(cfg)

	c.Requested = stages.CollectRequestedMetrics(b.Size, cfg.Fill, cfg.Border)

	return nil
}

func resolveFillMetric(cfg *config.Bubbletree, size metric.Name) metric.Name {
	if fill := cfg.Fill.MetricName(); fill != "" {
		return fill
	}

	return size
}

func resolveLabels(cfg *config.Bubbletree) LabelMode {
	if lbl := stages.PtrString(cfg.Labels); lbl != "" {
		return LabelMode(lbl)
	}

	return LabelFoldersOnly
}

// BuildInksStage builds the bubble inks and emits the "Rendering image" log line.
func BuildInksStage(c *stages.CommonState, b *State) error {
	slog.Info("Rendering image", "output", c.Output, "width", c.Width, "height", c.Height)

	b.Inks = BuildInks(c.Root, b.FillMetric, b.FillPalette, b.BorderMetric, b.BorderPalette)
	if !b.Flat {
		b.Inks.Fill = canvas.NewRadialGradientInk(b.Inks.Fill)
	}

	return nil
}

// BuildLegendStage builds the legend config from inks.
func BuildLegendStage(c *stages.CommonState, b *State, cfg *config.Bubbletree) error {
	_ = c
	pos, orient := legend.ResolveOptions(
		stages.PtrString(cfg.Legend),
		stages.PtrString(cfg.LegendOrientation),
	)
	b.LegendConfig = legend.Build(
		pos, orient,
		b.Inks.Fill, b.FillMetric,
		b.Inks.Border, b.BorderMetric,
		b.Size,
	)

	return nil
}

// LayoutStage reserves legend space, runs the bubble layout algorithm, and
// offsets the result into the remaining canvas area.
func LayoutStage(c *stages.CommonState, b *State) error {
	availH := c.Height - stages.EffectiveFooterHeight(c.RootConfig)
	layoutW, layoutH := legend.ReserveAndLayout(b.LegendConfig, c.Width, availH)

	b.Nodes = Layout(c.Root, layoutW, layoutH, b.Size, b.Labels)

	if layoutW < c.Width || layoutH < availH {
		if b.LegendConfig != nil {
			wReduce, hReduce := b.LegendConfig.ReserveSpace()
			dx, dy := legend.LayoutOffset(b.LegendConfig, wReduce, hReduce)
			OffsetNodes(&b.Nodes, dx, dy)
		}
	}

	return nil
}

// RenderStage renders the bubble tree to a canvas and attaches the legend.
func RenderStage(c *stages.CommonState, b *State) error {
	cv := RenderToCanvas(&b.Nodes, c.Root, c.Width, c.Height, b.Inks)
	if b.LegendConfig != nil {
		cv.SetLegend(*b.LegendConfig)
	}

	c.Canvas = cv

	return nil
}

// LogResult logs the final summary.
func LogResult(c *stages.CommonState, b *State) error {
	files, dirs := stages.CountAll(c.Root)

	slog.Info(
		"Rendered bubble tree",
		"files", files,
		"directories", dirs,
		"output", c.Output,
		"width", c.Width,
		"height", c.Height,
		"size_metric", string(b.Size),
		"fill_metric", string(b.FillMetric),
		"fill_palette", string(b.FillPalette),
		"border_metric", string(b.BorderMetric),
		"border_palette", string(b.BorderPalette),
	)

	return nil
}
```

- [ ] **Step 2: Create `internal/bubbletree/binary_filter.go`**

```go
package bubbletree

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// FilterBinaryFiles strips binary files from c.Root unless the bubbletree state
// requests they be kept.
func FilterBinaryFiles(c *stages.CommonState, b *State) error {
	return stages.FilterBinaryFiles(c, b.IncludeBinaryFiles)
}
```

- [ ] **Step 3: Compile check**

Run: `cd /home/bevan/github/code-visualizer && go build ./internal/bubbletree/...`
Expected: SUCCESS.

---

## Task 6: Rewrite viz `stages.go` for radialtree

**Files:**
- Modify: `internal/radialtree/stages.go`
- Create: `internal/radialtree/binary_filter.go`

- [ ] **Step 1: Rewrite `internal/radialtree/stages.go`**

Replace the file body with:

```go
package radialtree

import (
	"log/slog"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ResolveMetrics resolves disc-size, fill, and border metrics + palettes and
// fills c.Requested.
func ResolveMetrics(c *stages.CommonState, r *State, cfg *config.Radial) error {
	r.DiscSize = metric.Name(stages.PtrString(cfg.DiscSize))
	r.FillMetric = resolveFillMetric(cfg, r.DiscSize)
	r.FillPalette = stages.ResolveFillPalette(cfg.Fill, r.FillMetric)
	r.BorderMetric, r.BorderPalette = stages.ResolveBorderMetricAndPalette(cfg.Border)
	r.Labels = resolveLabels(cfg)

	c.Requested = stages.CollectRequestedMetrics(r.DiscSize, cfg.Fill, cfg.Border)

	return nil
}

func resolveFillMetric(cfg *config.Radial, discSize metric.Name) metric.Name {
	if fill := cfg.Fill.MetricName(); fill != "" {
		return fill
	}

	return discSize
}

func resolveLabels(cfg *config.Radial) LabelMode {
	if lbl := stages.PtrString(cfg.Labels); lbl != "" {
		return LabelMode(lbl)
	}

	return LabelAll
}

// BuildInksStage builds the radial inks and emits the Rendering image log line.
func BuildInksStage(c *stages.CommonState, r *State) error {
	canvasSize := min(c.Width, c.Height)

	slog.Info("Rendering image", "output", c.Output, "canvas_size", canvasSize)

	r.Inks = BuildInks(c.Root, r.FillMetric, r.FillPalette, r.BorderMetric, r.BorderPalette)

	return nil
}

// BuildLegendStage builds the legend config from inks.
func BuildLegendStage(c *stages.CommonState, r *State, cfg *config.Radial) error {
	_ = c
	pos, orient := legend.ResolveOptions(
		stages.PtrString(cfg.Legend),
		stages.PtrString(cfg.LegendOrientation),
	)
	r.LegendConfig = legend.Build(
		pos, orient,
		r.Inks.Fill, r.FillMetric,
		r.Inks.Border, r.BorderMetric,
		r.DiscSize,
	)

	return nil
}

// LayoutStage runs the radial tree layout algorithm.
// Radial uses a square canvas: canvasSize = min(Width, Height).
func LayoutStage(c *stages.CommonState, r *State) error {
	availH := c.Height - stages.EffectiveFooterHeight(c.RootConfig)
	canvasSize := min(c.Width, availH)

	r.Nodes = Layout(c.Root, canvasSize, r.DiscSize, r.Labels)

	return nil
}

// RenderStage renders the radial tree to a canvas and attaches the legend.
func RenderStage(c *stages.CommonState, r *State) error {
	canvasSize := min(c.Width, c.Height)

	cv := RenderToCanvas(&r.Nodes, c.Root, canvasSize, r.Inks)
	if r.LegendConfig != nil {
		cv.SetLegend(*r.LegendConfig)
	}

	c.Canvas = cv

	return nil
}

// LogResult logs the final summary.
func LogResult(c *stages.CommonState, r *State) error {
	files, dirs := stages.CountAll(c.Root)
	canvasSize := min(c.Width, c.Height)

	slog.Info(
		"Rendered radial tree",
		"files", files,
		"directories", dirs,
		"output", c.Output,
		"canvas_size", canvasSize,
		"disc_metric", string(r.DiscSize),
		"fill_metric", string(r.FillMetric),
		"fill_palette", string(r.FillPalette),
		"border_metric", string(r.BorderMetric),
		"border_palette", string(r.BorderPalette),
	)

	return nil
}
```

- [ ] **Step 2: Create `internal/radialtree/binary_filter.go`**

```go
package radialtree

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// FilterBinaryFiles strips binary files from c.Root unless the radialtree
// state requests they be kept.
func FilterBinaryFiles(c *stages.CommonState, r *State) error {
	return stages.FilterBinaryFiles(c, r.IncludeBinaryFiles)
}
```

- [ ] **Step 3: Compile check**

Run: `cd /home/bevan/github/code-visualizer && go build ./internal/radialtree/...`
Expected: SUCCESS.

---

## Task 7: Rewrite viz `stages.go` for spiral

**Files:**
- Modify: `internal/spiral/stages.go`
- Create: `internal/spiral/binary_filter.go`

- [ ] **Step 1: Rewrite `internal/spiral/stages.go`**

Replace the file body with:

```go
package spiral

import (
	"log/slog"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ResolveMetrics resolves size, fill, border, resolution, and label settings
// from the spiral config and populates c.Requested.
func ResolveMetrics(c *stages.CommonState, p *State, cfg *config.Spiral) error {
	p.Size = metric.Name(stages.PtrString(cfg.Size))
	p.FillMetric = cfg.Fill.MetricName()
	p.FillPalette = stages.ResolveFillPalette(cfg.Fill, p.FillMetric)
	p.BorderMetric, p.BorderPalette = stages.ResolveBorderMetricAndPalette(cfg.Border)
	p.Resolution = resolveResolution(cfg)
	p.Labels = resolveLabels(cfg)

	c.Requested = collectRequestedMetrics(p.Size, cfg.Fill, cfg.Border)

	return nil
}

func resolveResolution(cfg *config.Spiral) Resolution {
	if r := stages.PtrString(cfg.Resolution); r == "hourly" {
		return Hourly
	}

	return Daily
}

func resolveLabels(cfg *config.Spiral) LabelMode {
	if lbl := stages.PtrString(cfg.Labels); lbl != "" {
		return LabelMode(lbl)
	}

	return LabelLaps
}

// collectRequestedMetrics merges size + fill + border into a deduplicated
// metric set. When size is empty (spiral defaults to commit count), only fill
// and border contribute.
func collectRequestedMetrics(size metric.Name, fill, border *config.MetricSpec) []metric.Name {
	if size != "" {
		return stages.CollectRequestedMetrics(size, fill, border)
	}

	seen := map[metric.Name]bool{}

	var names []metric.Name

	for _, spec := range []*config.MetricSpec{fill, border} {
		if spec != nil && spec.Metric != "" && !seen[spec.Metric] {
			seen[spec.Metric] = true
			names = append(names, spec.Metric)
		}
	}

	return names
}

// BuildTimeBucketsStage builds time buckets from c.FileTimeRange and
// distributes files into them from c.FileHistory.
func BuildTimeBucketsStage(c *stages.CommonState, p *State) error {
	tr := stages.CommitTimeRange(c.FileTimeRange)
	if tr.Earliest.IsZero() {
		return eris.New("no commit timestamps available to build time buckets")
	}

	buckets := BuildTimeBuckets(p.Resolution, tr.Earliest, tr.Latest)
	if len(buckets) == 0 {
		return eris.New("no time buckets created from commit time range")
	}

	AssignFilesToBuckets(buckets, c.FileHistory)

	p.Buckets = buckets

	return nil
}

// AggregateBucketMetricsStage fills in per-bucket aggregated metric values.
func AggregateBucketMetricsStage(c *stages.CommonState, p *State) error {
	_ = c
	AggregateBucketMetrics(p.Buckets, p.Size, p.FillMetric, p.BorderMetric)

	return nil
}

// BuildInksStage builds spiral inks and emits the Rendering image log line.
func BuildInksStage(c *stages.CommonState, p *State) error {
	p.Inks = BuildInks(p.Buckets, p.FillMetric, p.FillPalette, p.BorderMetric, p.BorderPalette)

	slog.Info("Rendering image", "output", c.Output, "width", c.Width, "height", c.Height)

	return nil
}

// BuildLegendStage builds the legend config from the inks.
func BuildLegendStage(c *stages.CommonState, p *State, cfg *config.Spiral) error {
	_ = c
	pos, orient := legend.ResolveOptions(
		stages.PtrString(cfg.Legend),
		stages.PtrString(cfg.LegendOrientation),
	)

	p.LegendConfig = legend.Build(
		pos, orient,
		p.Inks.Fill, p.FillMetric,
		p.Inks.Border, p.BorderMetric,
		p.Size,
	)

	return nil
}

// LayoutStage runs the spiral layout algorithm and applies disc sizing.
func LayoutStage(c *stages.CommonState, p *State) error {
	availH := c.Height - stages.EffectiveFooterHeight(c.RootConfig)

	layout := Layout(p.Buckets, c.Width, availH, p.Resolution, p.Labels)
	maxDisc := MaxDiscRadius(len(p.Buckets), c.Width, availH, p.Resolution)

	ApplyDiscSizes(layout.Nodes, p.Buckets, maxDisc)

	p.Layout = layout

	return nil
}

// RenderStage renders the spiral to a canvas and attaches the legend.
func RenderStage(c *stages.CommonState, p *State) error {
	cv := RenderToCanvas(p.Layout, p.Buckets, c.Width, c.Height, p.Inks)

	if p.LegendConfig != nil {
		cv.SetLegend(*p.LegendConfig)
	}

	c.Canvas = cv

	return nil
}

// LogResult logs the final summary line.
func LogResult(c *stages.CommonState, p *State) error {
	files, dirs := stages.CountAll(c.Root)

	slog.Info(
		"Rendered spiral",
		"files", files,
		"directories", dirs,
		"width", c.Width,
		"height", c.Height,
		"size_metric", string(p.Size),
		"fill_metric", string(p.FillMetric),
		"fill_palette", string(p.FillPalette),
		"border_metric", string(p.BorderMetric),
		"border_palette", string(p.BorderPalette),
	)

	return nil
}
```

- [ ] **Step 2: Create `internal/spiral/binary_filter.go`**

```go
package spiral

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// FilterBinaryFiles strips binary files from c.Root unless the spiral state
// requests they be kept.
func FilterBinaryFiles(c *stages.CommonState, p *State) error {
	return stages.FilterBinaryFiles(c, p.IncludeBinaryFiles)
}
```

- [ ] **Step 3: Compile check**

Run: `cd /home/bevan/github/code-visualizer && go build ./internal/spiral/...`
Expected: SUCCESS.

---

## Task 8: Rewrite viz `stages.go` for scatter

**Files:**
- Modify: `internal/scatter/stages.go`
- Create: `internal/scatter/binary_filter.go`

- [ ] **Step 1: Rewrite `internal/scatter/stages.go`**

Replace the file body with:

```go
package scatter

import (
	"log/slog"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ResolveMetrics resolves scatter axes, size, fill, and border settings.
func ResolveMetrics(c *stages.CommonState, x *State, cfg *config.Scatter) error {
	if stages.PtrString(cfg.XAxis) == "" {
		return eris.New("x-axis metric is required")
	}

	xAxis, err := resolveAxisSpec(cfg.XAxis)
	if err != nil {
		return eris.Wrap(err, "invalid x-axis metric")
	}

	if stages.PtrString(cfg.YAxis) == "" {
		return eris.New("y-axis metric is required")
	}

	yAxis, err := resolveAxisSpec(cfg.YAxis)
	if err != nil {
		return eris.Wrap(err, "invalid y-axis metric")
	}

	size := metric.Name(stages.PtrString(cfg.Size))
	if size == "" {
		return eris.New("size metric is required")
	}

	x.XAxis = xAxis
	x.YAxis = yAxis
	x.Size = size
	x.FillMetric = resolveFillMetric(cfg, size)
	x.FillPalette = stages.ResolveFillPalette(cfg.Fill, x.FillMetric)
	x.BorderMetric, x.BorderPalette = stages.ResolveBorderMetricAndPalette(cfg.Border)
	c.Requested = collectRequestedMetrics(xAxis.Metric, yAxis.Metric, size, cfg.Fill, cfg.Border)

	return nil
}

func resolveAxisSpec(name *string) (AxisSpec, error) {
	metricName := metric.Name(stages.PtrString(name))
	descriptor, ok := provider.GetDescriptor(metricName)

	if !ok {
		return AxisSpec{}, eris.Errorf("unknown axis metric %q", metricName)
	}

	return AxisSpec{Metric: metricName, Kind: descriptor.Kind}, nil
}

func resolveFillMetric(cfg *config.Scatter, size metric.Name) metric.Name {
	if fill := cfg.Fill.MetricName(); fill != "" {
		return fill
	}

	return size
}

func collectRequestedMetrics(xAxis, yAxis, size metric.Name, fill, border *config.MetricSpec) []metric.Name {
	seen := map[metric.Name]bool{}
	names := make([]metric.Name, 0, 5)

	for _, name := range []metric.Name{xAxis, yAxis, size, fill.MetricName(), border.MetricName()} {
		if name == "" || seen[name] {
			continue
		}

		seen[name] = true
		names = append(names, name)
	}

	return names
}

// BuildInksStage collects plottable files and creates point inks.
func BuildInksStage(c *stages.CommonState, x *State) error {
	x.Dataset = CollectDataset(c.Root, x.XAxis, x.YAxis, x.Size)
	x.Inks = BuildInks(x.Dataset, x.FillMetric, x.FillPalette, x.BorderMetric, x.BorderPalette)

	slog.Info("Rendering image", "output", c.Output, "width", c.Width, "height", c.Height)

	return nil
}

// BuildLegendStage builds the legend config from the resolved inks.
func BuildLegendStage(c *stages.CommonState, x *State, cfg *config.Scatter) error {
	_ = c
	pos, orient := legend.ResolveOptions(
		stages.PtrString(cfg.Legend),
		stages.PtrString(cfg.LegendOrientation),
	)

	x.LegendConfig = legend.Build(
		pos,
		orient,
		x.Inks.Fill,
		x.FillMetric,
		x.Inks.Border,
		x.BorderMetric,
		x.Size,
	)

	return nil
}

// LayoutStage positions scatter points within the drawable plot area.
func LayoutStage(c *stages.CommonState, x *State) error {
	availH := c.Height - stages.EffectiveFooterHeight(c.RootConfig)
	layoutW, layoutH := legend.ReserveAndLayout(x.LegendConfig, c.Width, availH)

	layout := Layout(x.Dataset, layoutW, layoutH, x.XAxis, x.YAxis)
	if layoutW < c.Width || layoutH < availH {
		if x.LegendConfig != nil {
			wReduce, hReduce := x.LegendConfig.ReserveSpace()
			dx, dy := legend.LayoutOffset(x.LegendConfig, wReduce, hReduce)
			OffsetLayout(&layout, dx, dy)
		}
	}

	x.Layout = layout

	return nil
}

// RenderStage renders the scatter plot to a canvas.
func RenderStage(c *stages.CommonState, x *State) error {
	cv := RenderToCanvas(x.Layout, c.Width, c.Height, x.Inks)
	if x.LegendConfig != nil {
		cv.SetLegend(*x.LegendConfig)
	}

	c.Canvas = cv

	return nil
}

// LogResult logs the final scatter summary.
func LogResult(c *stages.CommonState, x *State) error {
	skipped := x.Dataset.Skipped.MissingX + x.Dataset.Skipped.MissingY + x.Dataset.Skipped.MissingSize

	slog.Info(
		"Rendered scatter plot",
		"files", len(x.Dataset.Points),
		"skipped_missing_x", x.Dataset.Skipped.MissingX,
		"skipped_missing_y", x.Dataset.Skipped.MissingY,
		"skipped_missing_size", x.Dataset.Skipped.MissingSize,
		"skipped_total", skipped,
		"output", c.Output,
		"width", c.Width,
		"height", c.Height,
		"x_axis", string(x.XAxis.Metric),
		"y_axis", string(x.YAxis.Metric),
		"size_metric", string(x.Size),
		"fill_metric", string(x.FillMetric),
		"fill_palette", string(x.FillPalette),
		"border_metric", string(x.BorderMetric),
		"border_palette", string(x.BorderPalette),
	)

	return nil
}
```

- [ ] **Step 2: Create `internal/scatter/binary_filter.go`**

```go
package scatter

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// FilterBinaryFiles strips binary files from c.Root unless the scatter state
// requests they be kept.
func FilterBinaryFiles(c *stages.CommonState, x *State) error {
	return stages.FilterBinaryFiles(c, x.IncludeBinaryFiles)
}
```

- [ ] **Step 3: Compile check**

Run: `cd /home/bevan/github/code-visualizer && go build ./internal/scatter/...`
Expected: SUCCESS.

- [ ] **Step 4: Whole-package compile check**

Run: `cd /home/bevan/github/code-visualizer && go build ./internal/...`
Expected: SUCCESS. (The `cmd/codeviz` package still won't build until Task 9.)

---

## Task 9: Rewrite the orchestrators in `cmd/codeviz/`

**Files:**
- Modify: `cmd/codeviz/treemap_cmd.go`
- Modify: `cmd/codeviz/bubbletree_cmd.go`
- Modify: `cmd/codeviz/radialtree_cmd.go`
- Modify: `cmd/codeviz/spiral_cmd.go`
- Modify: `cmd/codeviz/scatter_cmd.go`

- [ ] **Step 1: Rewrite `TreemapCmd.Run` in `cmd/codeviz/treemap_cmd.go`**

Replace just the `Run` method body with:

```go
func (c *TreemapCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	common := &stages.CommonState{
		TargetPath: c.TargetPath,
		Output:     c.Output,
		Flags:      toStagesFlags(flags),
		RootConfig: flags.Config,
		VizName:    "treemap",
		CLIFilters: c.Filters(),
	}
	cfg := flags.Config.Treemap
	viz := &treemap.State{
		IncludeBinaryFiles: c.IncludeBinaryFiles,
		Flat:               c.Flat,
	}

	s := pipeline.NewState(common, cfg, viz)

	pipeline.ApplyFuncX(s, stages.ValidatePaths)
	pipeline.ApplyFuncX(s, stages.ExportConfig)
	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncXYZ(s, treemap.ResolveMetrics)
	pipeline.ApplyFuncX(s, stages.ScanFilesystem)
	pipeline.ApplyFuncX(s, stages.CheckGitRequirement)
	pipeline.ApplyFuncX(s, stages.RunProviders)
	pipeline.ApplyFuncXY(s, treemap.FilterBinaryFiles)
	pipeline.ApplyFuncX(s, stages.ExportData)
	pipeline.ApplyFuncX(s, stages.ResolveDimensions)
	pipeline.ApplyFuncXY(s, treemap.BuildInksStage)
	pipeline.ApplyFuncXYZ(s, treemap.BuildLegendStage)
	pipeline.ApplyFuncXY(s, treemap.LayoutStage)
	pipeline.ApplyFuncXY(s, treemap.RenderStage)
	pipeline.ApplyFuncXYZ(s, treemap.LabelStage)
	pipeline.ApplyFuncXY(s, treemap.ApplyCanvasBlockLabels)
	pipeline.ApplyFuncX(s, stages.ApplyFooter)
	pipeline.ApplyFuncX(s, stages.WriteCanvas)
	pipeline.ApplyFuncXY(s, treemap.LogResult)

	return eris.Wrap(s.Err(), "treemap pipeline failed")
}
```

The imports already include `pipeline`, `stages`, `treemap`, `eris` — nothing to change.

- [ ] **Step 2: Rewrite `BubbletreeCmd.Run` in `cmd/codeviz/bubbletree_cmd.go`**

```go
func (c *BubbletreeCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	common := &stages.CommonState{
		TargetPath: c.TargetPath,
		Output:     c.Output,
		Flags:      toStagesFlags(flags),
		RootConfig: flags.Config,
		VizName:    "bubbletree",
		CLIFilters: c.Filters(),
	}
	cfg := flags.Config.Bubbletree
	viz := &bubbletree.State{
		IncludeBinaryFiles: c.IncludeBinaryFiles,
		Flat:               c.Flat,
	}

	s := pipeline.NewState(common, cfg, viz)

	pipeline.ApplyFuncX(s, stages.ValidatePaths)
	pipeline.ApplyFuncX(s, stages.ExportConfig)
	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncXYZ(s, bubbletree.ResolveMetrics)
	pipeline.ApplyFuncX(s, stages.ScanFilesystem)
	pipeline.ApplyFuncX(s, stages.CheckGitRequirement)
	pipeline.ApplyFuncX(s, stages.RunProviders)
	pipeline.ApplyFuncXY(s, bubbletree.FilterBinaryFiles)
	pipeline.ApplyFuncX(s, stages.ExportData)
	pipeline.ApplyFuncX(s, stages.ResolveDimensions)
	pipeline.ApplyFuncXY(s, bubbletree.BuildInksStage)
	pipeline.ApplyFuncXYZ(s, bubbletree.BuildLegendStage)
	pipeline.ApplyFuncXY(s, bubbletree.LayoutStage)
	pipeline.ApplyFuncXY(s, bubbletree.RenderStage)
	pipeline.ApplyFuncX(s, stages.ApplyFooter)
	pipeline.ApplyFuncX(s, stages.WriteCanvas)
	pipeline.ApplyFuncXY(s, bubbletree.LogResult)

	return eris.Wrap(s.Err(), "bubbletree pipeline failed")
}
```

- [ ] **Step 3: Rewrite `RadialCmd.Run` in `cmd/codeviz/radialtree_cmd.go`**

```go
func (c *RadialCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	common := &stages.CommonState{
		TargetPath: c.TargetPath,
		Output:     c.Output,
		Flags:      toStagesFlags(flags),
		RootConfig: flags.Config,
		VizName:    "radial",
		CLIFilters: c.Filters(),
	}
	cfg := flags.Config.Radial
	viz := &radialtree.State{
		IncludeBinaryFiles: c.IncludeBinaryFiles,
	}

	s := pipeline.NewState(common, cfg, viz)

	pipeline.ApplyFuncX(s, stages.ValidatePaths)
	pipeline.ApplyFuncX(s, stages.ExportConfig)
	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncXYZ(s, radialtree.ResolveMetrics)
	pipeline.ApplyFuncX(s, stages.ScanFilesystem)
	pipeline.ApplyFuncX(s, stages.CheckGitRequirement)
	pipeline.ApplyFuncX(s, stages.RunProviders)
	pipeline.ApplyFuncXY(s, radialtree.FilterBinaryFiles)
	pipeline.ApplyFuncX(s, stages.ExportData)
	pipeline.ApplyFuncX(s, stages.ResolveDimensions)
	pipeline.ApplyFuncXY(s, radialtree.BuildInksStage)
	pipeline.ApplyFuncXYZ(s, radialtree.BuildLegendStage)
	pipeline.ApplyFuncXY(s, radialtree.LayoutStage)
	pipeline.ApplyFuncXY(s, radialtree.RenderStage)
	pipeline.ApplyFuncX(s, stages.ApplyFooter)
	pipeline.ApplyFuncX(s, stages.WriteCanvas)
	pipeline.ApplyFuncXY(s, radialtree.LogResult)

	return eris.Wrap(s.Err(), "radialtree pipeline failed")
}
```

The `//nolint:dupl` directive on this function can stay — the calls remain structurally similar.

- [ ] **Step 4: Rewrite `SpiralCmd.Run` in `cmd/codeviz/spiral_cmd.go`**

```go
func (c *SpiralCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	common := &stages.CommonState{
		TargetPath: c.TargetPath,
		Output:     c.Output,
		Flags:      toStagesFlags(flags),
		RootConfig: flags.Config,
		VizName:    "spiral",
		CLIFilters: c.Filters(),
	}
	cfg := flags.Config.Spiral
	viz := &spiral.State{
		IncludeBinaryFiles: c.IncludeBinaryFiles,
	}

	s := pipeline.NewState(common, cfg, viz)

	pipeline.ApplyFuncX(s, stages.ValidatePaths)
	pipeline.ApplyFuncX(s, stages.ExportConfig)
	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncXYZ(s, spiral.ResolveMetrics)
	pipeline.ApplyFuncX(s, stages.ScanFilesystem)
	pipeline.ApplyFuncX(s, stages.CheckGitRequirement)
	pipeline.ApplyFuncX(s, stages.RunProviders)
	pipeline.ApplyFuncXY(s, spiral.FilterBinaryFiles)
	pipeline.ApplyFuncX(s, stages.ExportData)
	pipeline.ApplyFuncX(s, stages.LoadGitHistory)
	pipeline.ApplyFuncX(s, stages.GroupGitHistoryByFile)
	pipeline.ApplyFuncX(s, stages.ExtractFileHistory)
	pipeline.ApplyFuncX(s, stages.ResolveDimensions)
	pipeline.ApplyFuncXY(s, spiral.BuildTimeBucketsStage)
	pipeline.ApplyFuncXY(s, spiral.AggregateBucketMetricsStage)
	pipeline.ApplyFuncXY(s, spiral.BuildInksStage)
	pipeline.ApplyFuncXYZ(s, spiral.BuildLegendStage)
	pipeline.ApplyFuncXY(s, spiral.LayoutStage)
	pipeline.ApplyFuncXY(s, spiral.RenderStage)
	pipeline.ApplyFuncX(s, stages.ApplyFooter)
	pipeline.ApplyFuncX(s, stages.WriteCanvas)
	pipeline.ApplyFuncXY(s, spiral.LogResult)

	return eris.Wrap(s.Err(), "spiral pipeline failed")
}
```

- [ ] **Step 5: Rewrite `ScatterCmd.Run` in `cmd/codeviz/scatter_cmd.go`**

Note: this file imports the `scatter` package under the alias `scatterviz`. Keep the alias.

```go
func (c *ScatterCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	common := &stages.CommonState{
		TargetPath: c.TargetPath,
		Output:     c.Output,
		Flags:      toStagesFlags(flags),
		RootConfig: flags.Config,
		VizName:    "scatter",
		CLIFilters: c.Filters(),
	}
	cfg := flags.Config.Scatter
	viz := &scatterviz.State{
		IncludeBinaryFiles: c.IncludeBinaryFiles,
	}

	s := pipeline.NewState(common, cfg, viz)

	pipeline.ApplyFuncX(s, stages.ValidatePaths)
	pipeline.ApplyFuncX(s, stages.ExportConfig)
	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncXYZ(s, scatterviz.ResolveMetrics)
	pipeline.ApplyFuncX(s, stages.ScanFilesystem)
	pipeline.ApplyFuncX(s, stages.CheckGitRequirement)
	pipeline.ApplyFuncX(s, stages.RunProviders)
	pipeline.ApplyFuncXY(s, scatterviz.FilterBinaryFiles)
	pipeline.ApplyFuncX(s, stages.ExportData)
	pipeline.ApplyFuncX(s, stages.ResolveDimensions)
	pipeline.ApplyFuncXY(s, scatterviz.BuildInksStage)
	pipeline.ApplyFuncXYZ(s, scatterviz.BuildLegendStage)
	pipeline.ApplyFuncXY(s, scatterviz.LayoutStage)
	pipeline.ApplyFuncXY(s, scatterviz.RenderStage)
	pipeline.ApplyFuncX(s, stages.ApplyFooter)
	pipeline.ApplyFuncX(s, stages.WriteCanvas)
	pipeline.ApplyFuncXY(s, scatterviz.LogResult)

	return eris.Wrap(s.Err(), "scatter pipeline failed")
}
```

- [ ] **Step 6: Whole-workspace compile check**

Run: `cd /home/bevan/github/code-visualizer && go build ./...`
Expected: SUCCESS.

- [ ] **Step 7: Commit Tasks 2–9 together**

The package APIs have changed; tests for those packages are still broken. Commit the production code changes as a single coherent unit, then fix tests in the next tasks.

```bash
cd /home/bevan/github/code-visualizer
git add internal/treemap/ internal/bubbletree/ internal/radialtree/ \
        internal/spiral/ internal/scatter/ internal/stages/ \
        cmd/codeviz/
git commit -m "refactor(pipeline): migrate viz commands to typed-state pipeline"
```

---

## Task 10: Fix shared-stages tests

**Files:**
- Modify: `internal/stages/paths_test.go`
- Modify: `internal/stages/binary_test.go`
- Modify: `internal/stages/canvas_test.go`
- Delete: `internal/stages/labels_test.go`

- [ ] **Step 1: Delete `fakeState` from `paths_test.go`**

In `internal/stages/paths_test.go`, delete the `fakeState` type and its `Common()` method. The remaining tests (`TestValidatePathsHelper_*`) already call the helper directly; no further edits needed.

Result: the file's contents become exactly the three `TestValidatePathsHelper_*` tests plus their imports.

- [ ] **Step 2: Rewrite `internal/stages/binary_test.go`**

Replace the file body with:

```go
package stages_test

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestFilterBinaryFiles_IncludeFlagSet_NoOp(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Files: []*model.File{{Name: "a.bin"}, {Name: "b.go"}},
	}

	c := &stages.CommonState{Root: root}

	g.Expect(stages.FilterBinaryFiles(c, true)).To(Succeed())
	g.Expect(root.Files).To(HaveLen(2))
}

func TestFilterBinaryFiles_AllBinary_ReturnsNoFilesError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Files: []*model.File{
			{Name: "a.bin", IsBinary: true},
			{Name: "b.bin", IsBinary: true},
		},
	}

	c := &stages.CommonState{Root: root}
	err := stages.FilterBinaryFiles(c, false)

	var nfe *stages.NoFilesAfterFilterError
	g.Expect(errors.As(err, &nfe)).To(BeTrue())
}

func TestCountAll_NestedDirs(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Files: []*model.File{{Name: "a"}, {Name: "b"}},
		Dirs: []*model.Directory{
			{Files: []*model.File{{Name: "c"}}},
		},
	}

	files, dirs := stages.CountAll(root)
	g.Expect(files).To(Equal(3))
	g.Expect(dirs).To(Equal(1))
}
```

- [ ] **Step 3: Rewrite `internal/stages/canvas_test.go`**

In every test, replace `&fakeState{common: stages.CommonState{...}}` with `&stages.CommonState{...}` and replace `stages.ApplyFooter[*fakeState](s)` with `stages.ApplyFooter(c)` (and similarly `WriteCanvas`).

Concretely, the replacement file body is:

```go
package stages_test

import (
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestApplyFooter_NilCanvas_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := &stages.CommonState{Canvas: nil, RootConfig: config.New()}
	g.Expect(stages.ApplyFooter(c)).To(Succeed())
}

func TestApplyFooter_NilRootConfig_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := &stages.CommonState{Canvas: canvas.NewCanvas(100, 100), RootConfig: nil}
	g.Expect(stages.ApplyFooter(c)).To(Succeed())
}

func TestApplyFooter_FooterHidden_NoFooterOnCanvas(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.OverrideHideFooter(true)

	cv := canvas.NewCanvas(800, 600)
	c := &stages.CommonState{Canvas: cv, RootConfig: cfg}

	g.Expect(stages.ApplyFooter(c)).To(Succeed())
	g.Expect(cv.FooterText()).To(BeEmpty())
}

func TestApplyFooter_FooterWithText_SetsCanvasFooter(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.OverrideFooterText("Generated by codeviz")

	cv := canvas.NewCanvas(800, 600)
	c := &stages.CommonState{Canvas: cv, RootConfig: cfg}

	g.Expect(stages.ApplyFooter(c)).To(Succeed())
	g.Expect(cv.FooterText()).NotTo(BeEmpty())
	g.Expect(cv.FooterText()).To(ContainSubstring("Generated by codeviz"))
}

func TestApplyFooter_TemplateSubstitution_DateIsReplaced(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.OverrideFooterText("Generated at $date and $time")

	cv := canvas.NewCanvas(800, 600)
	c := &stages.CommonState{Canvas: cv, RootConfig: cfg}

	g.Expect(stages.ApplyFooter(c)).To(Succeed())

	text := cv.FooterText()
	g.Expect(text).NotTo(ContainSubstring("$date"))
	g.Expect(text).NotTo(ContainSubstring("$time"))
	g.Expect(strings.HasPrefix(text, "Generated at ")).To(BeTrue())
}

func TestApplyFooter_DefaultConfig_SetsFooter(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cv := canvas.NewCanvas(800, 600)
	c := &stages.CommonState{Canvas: cv, RootConfig: config.New()}

	g.Expect(stages.ApplyFooter(c)).To(Succeed())
	g.Expect(cv.FooterText()).NotTo(BeEmpty())
}

func TestApplyFooter_FooterRendersWithoutError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.OverrideFooterText("Test footer at $date")

	cv := canvas.NewCanvas(400, 300)
	c := &stages.CommonState{
		Canvas:     cv,
		RootConfig: cfg,
		Output:     filepath.Join(t.TempDir(), "out.png"),
	}

	g.Expect(stages.ApplyFooter(c)).To(Succeed())
	g.Expect(stages.WriteCanvas(c)).To(Succeed())
}

func TestEffectiveFooterHeight_NilConfig_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(stages.EffectiveFooterHeight(nil)).To(Equal(0))
}

func TestEffectiveFooterHeight_FooterHidden_ReturnsZero(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()
	cfg.OverrideHideFooter(true)

	g.Expect(stages.EffectiveFooterHeight(cfg)).To(Equal(0))
}

func TestEffectiveFooterHeight_FooterShown_ReturnsPositive(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cfg := config.New()

	height := stages.EffectiveFooterHeight(cfg)
	g.Expect(height).To(BeNumerically(">", 0))
	g.Expect(height).To(Equal(int(canvas.FooterReservedHeight)))
}
```

- [ ] **Step 4: Delete `internal/stages/labels_test.go`**

Run: `rm /home/bevan/github/code-visualizer/internal/stages/labels_test.go`

(The corresponding test moves to `internal/treemap/labels_stage_test.go` in Task 11.)

- [ ] **Step 5: Run the `stages` tests**

Run: `cd /home/bevan/github/code-visualizer && go test ./internal/stages/...`
Expected: PASS.

---

## Task 11: Fix the per-viz tests

**Files:**
- Modify: `internal/treemap/stages_test.go`
- Create: `internal/treemap/labels_stage_test.go`
- Create: `internal/treemap/binary_filter_test.go`
- Modify: `internal/bubbletree/stages_test.go`
- Create: `internal/bubbletree/binary_filter_test.go`
- Modify: `internal/radialtree/stages_test.go`
- Create: `internal/radialtree/binary_filter_test.go`
- Modify: `internal/spiral/stages_test.go`
- Create: `internal/spiral/binary_filter_test.go`
- Modify: `internal/scatter/stages_test.go` (if its tests refer to removed fields)
- Create: `internal/scatter/binary_filter_test.go`

### Strategy

For every existing test that constructs a viz state via a composite literal:

1. Replace `&treemap.State{CommonState: stages.CommonState{X: …}, Config: cfg, …}` with separate values: `common := &stages.CommonState{X: …}`, `viz := &treemap.State{…}` (no `CommonState`, no `Config`).
2. Replace any call `treemap.ResolveMetrics(s)` with `treemap.ResolveMetrics(common, viz, cfg)` — and likewise for other stages whose signatures changed. Stages taking one viz arg get called as `treemap.BuildInksStage(common, viz)`.
3. Replace any test assertion using `s.Common().Foo` with `common.Foo`.
4. Delete any test asserting satisfaction of `stages.BinaryFilterToggler`, the `TestState_CommonReturnsEmbeddedPointer` test, and the `TestState_IncludeBinary` test (their fields and methods no longer exist).

### Tasks per viz

- [ ] **Step 1: Edit `internal/treemap/stages_test.go`**

Apply the strategy above. Concrete deletions and rewrites for the parts shown in the spec:

- Delete the `TestState_CommonReturnsEmbeddedPointer` test entirely.
- Delete the `TestState_IncludeBinary` test entirely.
- Delete the `var _ stages.BinaryFilterToggler = on` line wherever it appears.
- In `TestResolveMetrics_SizeOnly` and `TestResolveMetrics_FillOverridesSizeAsFillMetric`: construct `common := &stages.CommonState{}`, `viz := &treemap.State{}`, `cfg := &config.Treemap{…}`, then call `treemap.ResolveMetrics(common, viz, cfg)`. Assertions on `s.Common().Requested` become `common.Requested`. Assertions on `s.Size` etc. become `viz.Size`.
- In `TestBuildInksStage_WrapsFillInkUnlessFlat`: build `common := &stages.CommonState{Root: root, Output: "out.png", Width: 100, Height: 100}` and `viz := &treemap.State{FillMetric: …, FillPalette: …, Flat: tc.flat}`. Call `treemap.BuildInksStage(common, viz)`. Assert on `viz.Inks`.
- In `TestBuildLegendStage_AddsLabelSampleLines`: build `common := &stages.CommonState{}`, `viz := &treemap.State{FillMetric: …, BorderMetric: …, Size: …, Inks: …}`, and `cfg := &config.Treemap{Fill: …, Border: …}`. Call `treemap.BuildLegendStage(common, viz, cfg)`. Assert on `viz.LegendConfig`.
- In `TestLayoutStage_FooterEnabled_ReducesAvailableHeight` and `TestLayoutStage_FooterDisabled_UsesFullHeight`: build `common := &stages.CommonState{Root: root, Width: width, Height: height, RootConfig: cfg}` and `viz := &treemap.State{Size: …, FillMetric: …, FillPalette: …}`. Call `treemap.LayoutStage(common, viz)`. The assertion `s.Root.Y + s.Root.H` becomes `viz.Root.Y + viz.Root.H`.

Verify your edits compile: `go test ./internal/treemap/... -run xxx -count=1` (any non-matching pattern; this just exercises compilation).

- [ ] **Step 2: Create `internal/treemap/labels_stage_test.go`**

This is the relocated content of the old `internal/stages/labels_test.go`, ported to call the new function.

```go
package treemap_test

import (
	"image/color"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
	"github.com/theunrepentantgeek/code-visualizer/internal/treemap"
)

func TestApplyCanvasBlockLabels_AddsLabelsToCanvas(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	out := filepath.Join(t.TempDir(), "labels.svg")
	common := &stages.CommonState{
		Output: out,
		Canvas: canvas.NewCanvas(120, 80),
	}
	viz := &treemap.State{
		BlockLabels: []canvas.BlockLabel{{
			X:     10,
			Y:     10,
			W:     100,
			H:     40,
			Lines: []string{"hello", "42"},
			Ink:   color.RGBA{A: 255},
		}},
	}

	g.Expect(treemap.ApplyCanvasBlockLabels(common, viz)).NotTo(HaveOccurred())
	g.Expect(common.Canvas.Render(out)).To(Succeed())

	data, err := os.ReadFile(out)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(string(data)).To(ContainSubstring("hello"))
	g.Expect(string(data)).To(ContainSubstring("42"))
}

func TestApplyCanvasBlockLabels_NilCanvas_NoOp(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	common := &stages.CommonState{Output: "out.png", Canvas: nil}
	viz := &treemap.State{}

	g.Expect(treemap.ApplyCanvasBlockLabels(common, viz)).To(Succeed())
}
```

- [ ] **Step 3: Create `internal/treemap/binary_filter_test.go`**

```go
package treemap_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
	"github.com/theunrepentantgeek/code-visualizer/internal/treemap"
)

func TestFilterBinaryFiles_RespectsIncludeFlag(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Files: []*model.File{{Name: "a.bin", IsBinary: true}, {Name: "b.go"}},
	}
	common := &stages.CommonState{Root: root}
	viz := &treemap.State{IncludeBinaryFiles: true}

	g.Expect(treemap.FilterBinaryFiles(common, viz)).To(Succeed())
	g.Expect(root.Files).To(HaveLen(2)) // binary preserved
}

func TestFilterBinaryFiles_DefaultStripsBinary(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Files: []*model.File{{Name: "a.bin", IsBinary: true}, {Name: "b.go"}},
	}
	common := &stages.CommonState{Root: root}
	viz := &treemap.State{IncludeBinaryFiles: false}

	g.Expect(treemap.FilterBinaryFiles(common, viz)).To(Succeed())
	g.Expect(root.Files).To(HaveLen(1))
	g.Expect(root.Files[0].Name).To(Equal("b.go"))
}
```

- [ ] **Step 4: Run treemap tests**

Run: `cd /home/bevan/github/code-visualizer && go test ./internal/treemap/...`
Expected: PASS.

- [ ] **Step 5: Edit `internal/bubbletree/stages_test.go` and create `internal/bubbletree/binary_filter_test.go`**

Apply the same strategy to bubbletree's existing tests. Specifically:

- Delete the line `var _ stages.BinaryFilterToggler = on`.
- For every test constructing `&bubbletree.State{CommonState: …, Config: …, …}`, split into `common`, `viz`, `cfg`.
- For each stage call:
  - `bubbletree.ResolveMetrics(s)` → `bubbletree.ResolveMetrics(common, viz, cfg)`
  - `bubbletree.BuildInksStage(s)` → `bubbletree.BuildInksStage(common, viz)`
  - `bubbletree.BuildLegendStage(s)` → `bubbletree.BuildLegendStage(common, viz, cfg)`
  - `bubbletree.LayoutStage(s)` → `bubbletree.LayoutStage(common, viz)`
  - `bubbletree.RenderStage(s)` → `bubbletree.RenderStage(common, viz)`
  - `bubbletree.LogResult(s)` → `bubbletree.LogResult(common, viz)`
- Replace `s.Common().X` with `common.X`, `s.Config` with `cfg`, `s.<viz-field>` with `viz.<viz-field>`.

Create `internal/bubbletree/binary_filter_test.go` modeled on the treemap one above, with `bubbletree.FilterBinaryFiles` and `bubbletree.State`.

Run: `go test ./internal/bubbletree/...` and confirm PASS.

- [ ] **Step 6: Edit `internal/radialtree/stages_test.go` and create `internal/radialtree/binary_filter_test.go`**

Same procedure as Step 5, substituting `radialtree`.

Run: `go test ./internal/radialtree/...` and confirm PASS.

- [ ] **Step 7: Edit `internal/spiral/stages_test.go` and create `internal/spiral/binary_filter_test.go`**

Same procedure as Step 5, substituting `spiral`. Note that spiral has additional stages: `BuildTimeBucketsStage`, `AggregateBucketMetricsStage` — both now take `(common, viz)`.

Run: `go test ./internal/spiral/...` and confirm PASS.

- [ ] **Step 8: Edit `internal/scatter/stages_test.go` (if applicable) and create `internal/scatter/binary_filter_test.go`**

If `internal/scatter/stages_test.go` exists and references `s.Common()`/`s.Config`/`CommonState`-style literals, apply the same procedure. (If the file doesn't exist or has minimal coverage, skip the edit but still create the binary-filter test below.)

Create `internal/scatter/binary_filter_test.go` modeled on the treemap one, using `scatter.FilterBinaryFiles` and `scatter.State`.

Run: `go test ./internal/scatter/...` and confirm PASS.

- [ ] **Step 9: Full-workspace test run**

Run: `cd /home/bevan/github/code-visualizer && go test ./...`
Expected: PASS for every package.

- [ ] **Step 10: Commit**

```bash
cd /home/bevan/github/code-visualizer
git add internal/treemap/ internal/bubbletree/ internal/radialtree/ \
        internal/spiral/ internal/scatter/ internal/stages/
git commit -m "test: align viz and stage tests with typed-state pipeline"
```

---

## Task 12: Delete the legacy pipeline API

**Files:**
- Delete: `internal/pipeline/pipeline.go`
- Delete: `internal/pipeline/pipeline_test.go`

- [ ] **Step 1: Verify no remaining references**

Run:
```bash
cd /home/bevan/github/code-visualizer
grep -rn "pipeline\.Stage\|pipeline\.Run" --include='*.go' .
```
Expected: no matches inside `cmd/` or `internal/` (other than `internal/pipeline/pipeline.go` and `internal/pipeline/pipeline_test.go` themselves). Docs may still mention them — ignore those.

If there are any matches, fix them first (something in Tasks 9–11 was missed).

- [ ] **Step 2: Delete the files**

Run:
```bash
cd /home/bevan/github/code-visualizer
rm internal/pipeline/pipeline.go internal/pipeline/pipeline_test.go
```

- [ ] **Step 3: Run pipeline tests**

Run: `cd /home/bevan/github/code-visualizer && go test ./internal/pipeline/...`
Expected: PASS (only `pipeline2_test.go` and `state_test.go` remain).

- [ ] **Step 4: Full-workspace build + test**

Run: `cd /home/bevan/github/code-visualizer && go build ./... && go test ./...`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd /home/bevan/github/code-visualizer
git add internal/pipeline/
git commit -m "refactor(pipeline): remove legacy Run/Stage API"
```

---

## Task 13: Final CI gate

- [ ] **Step 1: Run `task ci` via the Explore subagent**

Per repository memory: `task lint` / `task ci` are noisy, so dispatch them through the `Explore` subagent and ask for exit status + only the failing-lint / failing-test summary. Prompt template:

> "Run `task ci` in /home/bevan/github/code-visualizer. Return only: exit status, count and identity of failing linters / failing tests, the offending file:line and message for each issue, or a one-line note if no issues."

Expected: exit status 0, no failures.

- [ ] **Step 2: Fix anything CI surfaces**

If the linter complains (typical: unused imports, unused parameters with `_ = c` not satisfying the linter, wrapcheck flagged on unwrapped passthroughs), make minimal targeted fixes. Common patterns:

- Unused parameter `c` in a viz stage: the `_ = c` discard in the spec is intentional. If the linter still complains, rename to `_ *stages.CommonState`.
- Unused import: remove it.

Re-run the Explore subagent until CI is clean.

- [ ] **Step 3: Final commit if any fixes were needed**

```bash
cd /home/bevan/github/code-visualizer
git add -A
git commit -m "chore: satisfy linter after pipeline migration"
```

---

## Self-review checklist

After executing all tasks, verify:

1. **No `pipeline.Run` or `pipeline.Stage` references** anywhere in `cmd/` or `internal/`.
2. **No `stages.VizState`, `stages.BinaryFilterToggler`, `stages.CanvasLabelledState`** references anywhere.
3. **No `s.Common()` or `s.Config` calls** in any viz `stages.go` or `stages_test.go`.
4. **Every viz package has a `FilterBinaryFiles`** function (one-line wrapper).
5. **Only `treemap` has `ApplyCanvasBlockLabels`** (no other viz needs it).
6. **`task ci` passes.**
