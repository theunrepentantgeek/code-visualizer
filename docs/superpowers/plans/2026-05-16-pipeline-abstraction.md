# Pipeline Abstraction Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Introduce a reusable pipeline scaffold so visualization commands declare their lifecycle as composable stages, and prove it by refactoring the treemap command end-to-end.

**Architecture:** Generic `pipeline.Stage[S]` / `pipeline.Run` already exists; this plan adds tests for it, extracts shared lifecycle stages into a new `internal/stages` package keyed off a `VizState` interface, extracts legend code into `internal/legend`, then rewrites `cmd/codeviz/treemap_cmd.go` `Run()` as a flat composition of stages.

**Tech Stack:** Go 1.26.1, Kong, eris, Gomega, Goldie v2, fogleman/gg, go-git. Toolchain via Taskfile (`task build`, `task test`, `task lint`, `task ci`).

**Reference spec:** [docs/superpowers/specs/2026-05-16-pipeline-abstraction-design.md](../specs/2026-05-16-pipeline-abstraction-design.md)

---

## File Structure

| Path                                 | Responsibility                                                                                                                                                                              |
| ------------------------------------ | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `internal/pipeline/pipeline.go`      | `Stage[S any] = func(S) error` + `Run[S any]`. Domain-agnostic.                                                                                                                             |
| `internal/pipeline/pipeline_test.go` | Tests for the seven documented semantics.                                                                                                                                                   |
| `internal/legend/legend.go`          | `Build`, `ResolveOptions`. Constructs `canvas.LegendConfig` from inks + metric names.                                                                                                       |
| `internal/legend/reserve.go`         | `ReserveAndLayout`, `LayoutOffset`, `cornerLegendOffset`, `MinReservableSize`.                                                                                                              |
| `internal/legend/*_test.go`          | Tests ported from `cmd/codeviz/legend_builder_test.go`.                                                                                                                                     |
| `internal/stages/common.go`          | `CommonState` struct, `VizState` interface.                                                                                                                                                 |
| `internal/stages/errors.go`          | Sentinel error types (`gitRequiredError`, `targetPathError`, `outputPathError`, `noFilesAfterFilterError`) and their constructors / matchers.                                               |
| `internal/stages/paths.go`           | `ValidatePathsHelper` (the pure func) and `ValidatePaths[S]` (the stage).                                                                                                                   |
| `internal/stages/filter.go`          | `BuildFilterRulesHelper`, `BuildFilterRules[S]` stage.                                                                                                                                      |
| `internal/stages/git.go`             | `CheckGitRepoHelper`, `CheckGitRequirement[S]` stage.                                                                                                                                       |
| `internal/stages/scan.go`            | `ScanFilesystem[S]` stage + progress helpers (`buildScanProgress` and friends moved from `cmd/codeviz/progress.go`).                                                                        |
| `internal/stages/metrics.go`         | `RunProviders[S]` stage + `metricProgressTracker` (moved from progress.go) + `resolveFillPalette`, `resolveBorderMetricAndPalette`, `specMetric`, `specPalette`, `collectRequestedMetrics`. |
| `internal/stages/binary.go`          | `FilterBinaryFiles[S]` stage + `filterBinaryFilesHelper`, `countAll`.                                                                                                                       |
| `internal/stages/export.go`          | `ExportConfig[S]`, `ExportData[S]` stages.                                                                                                                                                  |
| `internal/stages/dimensions.go`      | `ResolveDimensions[S]` stage + `ptrInt`.                                                                                                                                                    |
| `internal/stages/canvas.go`          | `WriteCanvas[S]` stage.                                                                                                                                                                     |
| `internal/stages/*_test.go`          | Per-stage tests using a tiny fake `VizState`.                                                                                                                                               |
| `internal/treemap/state.go`          | `State` struct, `Common()` method.                                                                                                                                                          |
| `internal/treemap/stages.go`         | `ResolveMetrics`, `BuildInksStage`, `BuildLegendStage`, `LayoutStage`, `RenderStage`, `LogResult` stages.                                                                                   |
| `internal/treemap/inks.go`           | `BuildInks` + `Inks` type, **moved** from `cmd/codeviz/treemap_canvas.go`.                                                                                                                  |
| `internal/treemap/render.go`         | `RenderToCanvas`, moved from `cmd/codeviz/treemap_canvas.go`.                                                                                                                               |
| `cmd/codeviz/treemap_cmd.go`         | Kong struct, config merging, `Run()` pipeline composition. Helpers gone.                                                                                                                    |
| `cmd/codeviz/main.go`                | `classifyError` updated to use sentinels from `internal/stages`.                                                                                                                            |
| `cmd/codeviz/viz_cmd_helpers.go`     | Shrinks. Only retains helpers the not-yet-refactored bubbletree/radialtree/spiral commands still need (most are now re-exports from `internal/stages` via thin forwarders).                 |
| `cmd/codeviz/progress.go`            | Most contents moved to `internal/stages/scan.go` and `internal/stages/metrics.go`. `buildHistoryProgress` stays here (spiral-only, not refactored in this plan).                            |

---

## Task 1: Drop unused `C` type parameter and relax `Stage` to take `S`

**Files:**
- Modify: `internal/pipeline/pipeline.go`

- [ ] **Step 1: Replace pipeline.go contents**

Replace the entire file with:

```go
package pipeline

// Stage is a single step in a pipeline. It receives the state and returns an
// error if execution should halt. When the type argument S is a pointer type,
// mutations made by the stage are visible to subsequent stages and to the
// caller of Run.
type Stage[S any] func(S) error

// Run executes stages in order against initialState. If any stage returns an
// error, execution halts immediately and the (possibly partially mutated)
// state plus the unwrapped error are returned. Run does not wrap stage
// errors; callers and stages own wrapping conventions.
func Run[S any](initialState S, stages ...Stage[S]) (S, error) {
	state := initialState
	for _, stage := range stages {
		if err := stage(state); err != nil {
			return state, err
		}
	}

	return state, nil
}
```

- [ ] **Step 2: Build to confirm compilation**

Run: `go build ./internal/pipeline/...`
Expected: no errors. (No callers exist yet — package is unused.)

- [ ] **Step 3: Commit**

```bash
git add internal/pipeline/pipeline.go
git commit -m "refactor(pipeline): drop unused C type param; Stage[S] takes S"
```

---

## Task 2: Pipeline unit tests

**Files:**
- Create: `internal/pipeline/pipeline_test.go`

- [ ] **Step 1: Write the failing tests**

```go
package pipeline_test

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
)

type counter struct {
	n   int
	log []string
}

func incN(amount int) pipeline.Stage[*counter] {
	return func(c *counter) error {
		c.n += amount
		c.log = append(c.log, "inc")
		return nil
	}
}

func fail(msg string) pipeline.Stage[*counter] {
	return func(c *counter) error {
		c.log = append(c.log, "fail")
		return errors.New(msg)
	}
}

func TestRun_EmptyPipeline_ReturnsStateUnchanged(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := &counter{n: 7}
	got, err := pipeline.Run(c)

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(got).To(BeIdenticalTo(c))
	g.Expect(c.n).To(Equal(7))
	g.Expect(c.log).To(BeEmpty())
}

func TestRun_SingleStage_RunsOnce(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := &counter{}
	_, err := pipeline.Run(c, incN(3))

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(c.n).To(Equal(3))
	g.Expect(c.log).To(Equal([]string{"inc"}))
}

func TestRun_MultipleStages_RunInDeclarationOrder(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := &counter{}
	_, err := pipeline.Run(c, incN(1), incN(2), incN(4))

	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(c.n).To(Equal(7))
	g.Expect(c.log).To(Equal([]string{"inc", "inc", "inc"}))
}

func TestRun_ErrorHaltsExecution(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := &counter{}
	_, err := pipeline.Run(c, incN(1), fail("boom"), incN(100))

	g.Expect(err).To(MatchError("boom"))
	g.Expect(c.n).To(Equal(1))
	g.Expect(c.log).To(Equal([]string{"inc", "fail"}))
}

func TestRun_PartialStateReturnedOnError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	c := &counter{}
	got, err := pipeline.Run(c, incN(2), incN(3), fail("stop"))

	g.Expect(err).To(HaveOccurred())
	g.Expect(got).To(BeIdenticalTo(c))
	g.Expect(c.n).To(Equal(5))
}

func TestRun_ErrorReturnedUnwrapped(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sentinel := errors.New("sentinel")
	c := &counter{}
	_, err := pipeline.Run(c, func(*counter) error { return sentinel })

	g.Expect(err).To(BeIdenticalTo(sentinel))
}

func TestRun_NilStage_Panics(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	defer func() {
		r := recover()
		g.Expect(r).NotTo(BeNil())
	}()

	c := &counter{}
	_, _ = pipeline.Run(c, nil)
}
```

- [ ] **Step 2: Run tests to verify they pass**

Run: `go test ./internal/pipeline/...`
Expected: PASS (all 7 tests).

- [ ] **Step 3: Commit**

```bash
git add internal/pipeline/pipeline_test.go
git commit -m "test(pipeline): cover stage semantics"
```

---

## Task 3: Create `internal/legend` — port `Build` and `ResolveOptions`

**Files:**
- Create: `internal/legend/legend.go`
- Create: `internal/legend/legend_test.go`

- [ ] **Step 1: Create legend.go with Build and ResolveOptions**

```go
// Package legend constructs canvas.LegendConfig values from resolved
// visualization options and reserves canvas space for legend rendering.
// It is reusable across all visualization types.
package legend

import (
	"image/color"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

// white is the colour used for FixedInk in size-only entries.
var white = color.RGBA{R: 255, G: 255, B: 255, A: 255} //nolint:gochecknoglobals // shared colour constant

// ResolveOptions resolves legend position and orientation from raw strings.
// Empty position defaults to "bottom-right"; empty orientation is derived
// from the resolved position.
func ResolveOptions(posStr, orientStr string) (canvas.LegendPosition, canvas.LegendOrientation) {
	pos := canvas.LegendPosition(posStr)
	if pos == "" {
		pos = canvas.LegendPositionBottomRight
	}

	orient := canvas.LegendOrientation(orientStr)
	if orient == "" {
		orient = canvas.DefaultOrientation(pos)
	}

	return pos, orient
}

// Build constructs a LegendConfig from resolved options and the pre-built
// Ink objects used for rendering. Returns nil if the legend is disabled
// ("none") or no entries would be produced.
func Build(
	position canvas.LegendPosition,
	orientation canvas.LegendOrientation,
	fillInk canvas.Ink,
	fillMetric metric.Name,
	borderInk canvas.Ink,
	borderMetric metric.Name,
	sizeMetric metric.Name,
) *canvas.LegendConfig {
	if position == canvas.LegendPositionNone {
		return nil
	}

	if orientation == "" {
		orientation = canvas.DefaultOrientation(position)
	}

	var entries []canvas.LegendEntry

	if fillMetric != "" {
		entries = append(entries, canvas.LegendEntry{
			Role:       canvas.LegendRoleFill,
			MetricName: string(fillMetric),
			Ink:        fillInk,
		})
	}

	if borderMetric != "" {
		entries = append(entries, canvas.LegendEntry{
			Role:       canvas.LegendRoleBorder,
			MetricName: string(borderMetric),
			Ink:        borderInk,
		})
	}

	if sizeMetric != "" && sizeMetric != fillMetric {
		entries = append(entries, canvas.LegendEntry{
			Role:       canvas.LegendRoleSize,
			MetricName: string(sizeMetric),
			Ink:        canvas.FixedInk(white),
		})
	}

	if len(entries) == 0 {
		return nil
	}

	return &canvas.LegendConfig{
		Position:    position,
		Orientation: orientation,
		Entries:     entries,
	}
}
```

- [ ] **Step 2: Port tests verbatim with rename**

Copy `cmd/codeviz/legend_builder_test.go` into `internal/legend/legend_test.go`, changing:
- `package main` → `package legend_test`
- Add import `"github.com/theunrepentantgeek/code-visualizer/internal/legend"`
- Replace bare `resolveLegendOptions(...)` with `legend.ResolveOptions(...)` and `buildLegendConfig(...)` with `legend.Build(...)`.
- Keep `makeLegendTestRoot`, `collectNumericValues`, `collectDistinctTypes` as unexported helpers in the test file (you may need to copy those funcs from `cmd/codeviz` — check `treemap_canvas_test.go` for the `collectNumericValues` / `collectDistinctTypes` definitions and copy them too).

Verify by running:

Run: `go test ./internal/legend/...`
Expected: PASS

- [ ] **Step 3: Commit**

```bash
git add internal/legend/
git commit -m "feat(legend): extract Build and ResolveOptions to internal/legend"
```

---

## Task 4: Move legend reservation/offset math to `internal/legend`

**Files:**
- Create: `internal/legend/reserve.go`

- [ ] **Step 1: Create reserve.go**

```go
package legend

import "github.com/theunrepentantgeek/code-visualizer/internal/canvas"

// MinReservableSize is the smallest canvas dimension (px) that still
// produces a usable visualization. If reserving legend space would shrink
// either dimension below this, ReserveAndLayout falls back to the full
// canvas (overlay behaviour).
const MinReservableSize = 100

// ReserveAndLayout returns the layout dimensions after reserving space
// for the legend. Falls back to (width, height) when reservation would
// shrink either dimension below MinReservableSize.
func ReserveAndLayout(cfg *canvas.LegendConfig, width, height int) (layoutW, layoutH int) {
	if cfg == nil {
		return width, height
	}

	wReduce, hReduce := cfg.ReserveSpace()

	w := width - int(wReduce)
	h := height - int(hReduce)

	if w < MinReservableSize || h < MinReservableSize {
		return width, height
	}

	return w, h
}

// LayoutOffset returns the (dx, dy) offset to apply to layout output
// when space has been reserved for the legend.
func LayoutOffset(cfg *canvas.LegendConfig, wReduce, hReduce float64) (dx, dy float64) {
	if cfg == nil {
		return 0, 0
	}

	switch cfg.Position {
	case canvas.LegendPositionTopCenter:
		return 0, hReduce
	case canvas.LegendPositionCenterLeft:
		return wReduce, 0
	default:
		return cornerOffset(cfg, wReduce, hReduce)
	}
}

func cornerOffset(cfg *canvas.LegendConfig, wReduce, hReduce float64) (dx, dy float64) {
	isTop := cfg.Position == canvas.LegendPositionTopLeft || cfg.Position == canvas.LegendPositionTopRight
	isLeft := cfg.Position == canvas.LegendPositionTopLeft || cfg.Position == canvas.LegendPositionBottomLeft

	if cfg.Orientation == canvas.LegendOrientationVertical {
		if isLeft {
			return wReduce, 0
		}

		return 0, 0
	}

	if isTop {
		return 0, hReduce
	}

	return 0, 0
}
```

- [ ] **Step 2: Build and run existing tests**

Run: `go build ./internal/legend/... && go test ./internal/legend/...`
Expected: PASS (no new tests yet; verifies the file compiles).

- [ ] **Step 3: Commit**

```bash
git add internal/legend/reserve.go
git commit -m "feat(legend): move space reservation and offset math"
```

---

## Task 5: Redirect treemap callers and delete `cmd/codeviz/legend_builder.go`

**Files:**
- Modify: `cmd/codeviz/treemap_cmd.go`
- Delete: `cmd/codeviz/legend_builder.go`
- Delete: `cmd/codeviz/legend_builder_test.go`

- [ ] **Step 1: Update treemap_cmd.go imports and call sites**

Add import: `"github.com/theunrepentantgeek/code-visualizer/internal/legend"`

Replace these calls (search for them in `treemap_cmd.go`):

- `resolveLegendOptions(...)` → `legend.ResolveOptions(...)`
- `buildLegendConfig(...)` → `legend.Build(...)`
- `reserveAndLayout(...)` → `legend.ReserveAndLayout(...)`
- `legendLayoutOffset(...)` → `legend.LayoutOffset(...)`
- `minReservableSize` → `legend.MinReservableSize`

Remove `reserveAndLayout`, `legendLayoutOffset`, `cornerLegendOffset`, and `minReservableSize` definitions from `treemap_cmd.go` (lines around 220–270 in the current file).

- [ ] **Step 2: Repeat for any other current legend callers**

Run: `grep -rn "resolveLegendOptions\|buildLegendConfig\|reserveAndLayout\|legendLayoutOffset\|minReservableSize" cmd/codeviz/`

For each match, replace with the `legend.*` form. The other viz `*_cmd.go` files likely call `resolveLegendOptions` and `buildLegendConfig` — update them too. (We must keep CI green for all four commands even though only treemap is being refactored.)

- [ ] **Step 3: Delete old files**

```bash
rm cmd/codeviz/legend_builder.go cmd/codeviz/legend_builder_test.go
```

- [ ] **Step 4: Build, test, lint**

Run: `task ci`
Expected: PASS — all golden tests, all unit tests, lint clean.

- [ ] **Step 5: Commit**

```bash
git add -A cmd/codeviz/ internal/legend/
git commit -m "refactor: route viz commands through internal/legend"
```

---

## Task 6: Create `internal/stages` skeleton — `CommonState` and `VizState`

**Files:**
- Create: `internal/stages/common.go`

- [ ] **Step 1: Create common.go**

```go
// Package stages provides shared visualization-pipeline stages and the
// CommonState type that they operate on. Visualization-specific state
// types embed CommonState and satisfy the VizState interface.
package stages

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
)

// Flags is the cross-cutting flag bundle passed to every viz command's Run.
// It mirrors cmd/codeviz.Flags but lives here so this package does not
// depend on package main. The orchestrator constructs one and assigns it
// into CommonState.Flags before running the pipeline.
type Flags struct {
	Quiet        bool
	Verbose      bool
	Debug        bool
	ExportConfig string
	ExportData   string
	Config       *config.Config
}

// CommonState contains fields used by shared stages. Every viz state struct
// embeds this and exposes it via a pointer-receiver Common() method.
type CommonState struct {
	// Inputs: set by the orchestrator before pipeline.Run.
	TargetPath  string
	Output      string
	Flags       *Flags
	RootConfig  *config.Config
	CLIFilters  []string

	// Populated by shared stages during the pipeline:
	FilterRules []filter.Rule    // BuildFilterRules
	Requested   []metric.Name    // viz-specific ResolveMetrics
	Root        *model.Directory // ScanFilesystem
	Width       int              // ResolveDimensions
	Height      int              // ResolveDimensions
	Canvas      *canvas.Canvas   // viz-specific Render
}

// VizState is satisfied by any state type that embeds CommonState and
// exposes it via a Common() method. Shared stages are generic over this
// interface; in practice the type argument is always a pointer type
// (e.g. *treemap.State).
type VizState interface {
	Common() *CommonState
}
```

- [ ] **Step 2: Build**

Run: `go build ./internal/stages/...`
Expected: PASS.

- [ ] **Step 3: Commit**

```bash
git add internal/stages/common.go
git commit -m "feat(stages): add CommonState and VizState skeleton"
```

---

## Task 7: Move sentinel error types to `internal/stages`

**Files:**
- Create: `internal/stages/errors.go`
- Modify: `cmd/codeviz/main.go`
- Modify: `cmd/codeviz/viz_cmd_helpers.go`

- [ ] **Step 1: Create errors.go with the moved types**

```go
package stages

import (
	"fmt"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
)

// GitRequiredError reports that a requested metric needs a git repository
// but the target path is not inside one.
type GitRequiredError struct {
	Metric metric.Name
	Target string
}

func (e *GitRequiredError) Error() string {
	return fmt.Sprintf("metric %q requires a git repository, but %q is not a git repository", e.Metric, e.Target)
}

// TargetPathError reports a problem with the target directory argument.
type TargetPathError struct {
	Msg string
}

func (e *TargetPathError) Error() string { return e.Msg }

// OutputPathError reports a problem with the output file path.
type OutputPathError struct {
	Msg string
}

func (e *OutputPathError) Error() string { return e.Msg }

// NoFilesAfterFilterMsg is the message used when binary filtering empties the tree.
const NoFilesAfterFilterMsg = "no files available for visualization after excluding binary files"

// NoFilesAfterFilterError reports that no files remain after filtering.
type NoFilesAfterFilterError struct {
	Msg string
}

func (e *NoFilesAfterFilterError) Error() string { return e.Msg }
```

- [ ] **Step 2: Remove the old type definitions from `cmd/codeviz/main.go`**

Delete the `gitRequiredError`, `targetPathError`, `outputPathError`, `noFilesAfterFilterError` type declarations and the `noFilesAfterFilterMsg` constant (currently at the bottom of `main.go`).

- [ ] **Step 3: Update `classifyError` to use the new types**

Replace the `classifyError` body with:

```go
func classifyError(err error) int {
	var (
		gitErr     *stages.GitRequiredError
		targetErr  *stages.TargetPathError
		outputErr  *stages.OutputPathError
		noFilesErr *stages.NoFilesAfterFilterError
	)

	switch {
	case errors.As(err, &targetErr):
		return 2
	case errors.As(err, &gitErr):
		return 3
	case errors.As(err, &outputErr):
		return 4
	case errors.As(err, &noFilesErr):
		return 6
	default:
		return 5
	}
}
```

Add import: `"github.com/theunrepentantgeek/code-visualizer/internal/stages"`. Drop the now-unused `metric` import from `main.go` if it is no longer referenced.

- [ ] **Step 4: Update producers in `cmd/codeviz/viz_cmd_helpers.go`**

Replace references in `validatePaths`, `verifyGitRepo`, `filterBinaryFiles`:
- `&outputPathError{msg: ...}` → `&stages.OutputPathError{Msg: ...}`
- `&targetPathError{msg: ...}` → `&stages.TargetPathError{Msg: ...}`
- `&gitRequiredError{metric: name, target: targetPath}` → `&stages.GitRequiredError{Metric: name, Target: targetPath}`
- `&noFilesAfterFilterError{msg: noFilesAfterFilterMsg}` → `&stages.NoFilesAfterFilterError{Msg: stages.NoFilesAfterFilterMsg}`

Add `stages` import.

- [ ] **Step 5: Build, test**

Run: `task ci`
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "refactor(stages): move sentinel error types to internal/stages"
```

---

## Task 8: Move path validation to `internal/stages`

**Files:**
- Create: `internal/stages/paths.go`
- Create: `internal/stages/paths_test.go`
- Modify: `cmd/codeviz/viz_cmd_helpers.go`

- [ ] **Step 1: Create paths.go**

```go
package stages

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
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

// ValidatePaths is a pipeline.Stage that validates Common().TargetPath and
// Common().Output.
func ValidatePaths[S VizState](s S) error {
	c := s.Common()
	if err := ValidatePathsHelper(c.TargetPath, c.Output); err != nil {
		return eris.Wrap(err, "invalid paths")
	}

	return nil
}

// Compile-time assurance the signature is a valid Stage.
var _ pipeline.Stage[VizState] = ValidatePaths[VizState]
```

- [ ] **Step 2: Write paths_test.go covering helper + stage**

```go
package stages_test

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// fakeState is the minimal VizState used by stage tests in this package.
type fakeState struct {
	common stages.CommonState
}

func (f *fakeState) Common() *stages.CommonState { return &f.common }

func TestValidatePathsHelper_MissingTarget(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	err := stages.ValidatePathsHelper("/no/such/path", "out.png")

	var tpe *stages.TargetPathError
	g.Expect(errors.As(err, &tpe)).To(BeTrue())
}

func TestValidatePathsHelper_BadOutputFormat(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	err := stages.ValidatePathsHelper(dir, "out.unknown")

	var ope *stages.OutputPathError
	g.Expect(errors.As(err, &ope)).To(BeTrue())
}

func TestValidatePathsHelper_OK(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	out := filepath.Join(dir, "out.png")

	g.Expect(stages.ValidatePathsHelper(dir, out)).To(Succeed())
}

func TestValidatePaths_Stage_WrapsError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &fakeState{common: stages.CommonState{TargetPath: "/nope", Output: "out.png"}}
	err := stages.ValidatePaths[*fakeState](s)

	g.Expect(err).To(HaveOccurred())
	var tpe *stages.TargetPathError
	g.Expect(errors.As(err, &tpe)).To(BeTrue())
}

func TestValidatePaths_Stage_OK(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	out := filepath.Join(dir, "out.png")
	// also ensure parent dir exists
	g.Expect(os.MkdirAll(filepath.Dir(out), 0o755)).To(Succeed())

	s := &fakeState{common: stages.CommonState{TargetPath: dir, Output: out}}
	g.Expect(stages.ValidatePaths[*fakeState](s)).To(Succeed())
}
```

- [ ] **Step 3: Delete `validatePaths` from `viz_cmd_helpers.go` and redirect callers**

In `cmd/codeviz/viz_cmd_helpers.go`, delete the `validatePaths` function entirely. In each `*_cmd.go` that calls `validatePaths(c.TargetPath, c.Output)`, replace with `stages.ValidatePathsHelper(c.TargetPath, c.Output)`. Use grep to find them.

- [ ] **Step 4: Build, test, lint**

Run: `task ci`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor(stages): move path validation to internal/stages"
```

---

## Task 9: Move filter rule building to `internal/stages`

**Files:**
- Create: `internal/stages/filter.go`
- Create: `internal/stages/filter_test.go`
- Modify: `cmd/codeviz/viz_cmd_helpers.go` and callers

- [ ] **Step 1: Create filter.go**

```go
package stages

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
)

// BuildFilterRulesHelper merges config-file filter rules with CLI --filter
// flags. CLI filters must already have been syntax-validated by the
// command's Validate() method.
func BuildFilterRulesHelper(cfg *config.Config, cliFilters []string) []filter.Rule {
	rules := make([]filter.Rule, 0, len(cfg.FileFilter)+len(cliFilters))
	rules = append(rules, cfg.FileFilter...)

	for _, f := range cliFilters {
		rule, _ := filter.ParseFilterFlag(f) // already validated
		rules = append(rules, rule)
	}

	return rules
}

// BuildFilterRules is a pipeline.Stage that populates Common().FilterRules
// from Common().RootConfig.FileFilter plus Common().CLIFilters.
func BuildFilterRules[S VizState](s S) error {
	c := s.Common()
	c.FilterRules = BuildFilterRulesHelper(c.RootConfig, c.CLIFilters)
	return nil
}

var _ pipeline.Stage[VizState] = BuildFilterRules[VizState]
```

- [ ] **Step 2: Write filter_test.go**

```go
package stages_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestBuildFilterRulesHelper_MergesConfigAndCLI(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rule, err := filter.ParseFilterFlag("*.go")
	g.Expect(err).NotTo(HaveOccurred())

	cfg := &config.Config{FileFilter: []filter.Rule{rule}}

	got := stages.BuildFilterRulesHelper(cfg, []string{"!*_test.go"})

	g.Expect(got).To(HaveLen(2))
}

func TestBuildFilterRules_Stage_PopulatesCommon(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &fakeState{common: stages.CommonState{
		RootConfig: &config.Config{},
		CLIFilters: []string{"*.go"},
	}}

	g.Expect(stages.BuildFilterRules[*fakeState](s)).To(Succeed())
	g.Expect(s.Common().FilterRules).To(HaveLen(1))
}
```

- [ ] **Step 3: Delete `buildFilterRules` from viz_cmd_helpers.go and redirect**

Replace every callsite `buildFilterRules(flags.Config, c.Filter)` with `stages.BuildFilterRulesHelper(flags.Config, c.Filter)`.

- [ ] **Step 4: Run CI**

Run: `task ci`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor(stages): move filter-rule helper to internal/stages"
```

---

## Task 10: Move git requirement check to `internal/stages`

**Files:**
- Create: `internal/stages/git.go`
- Create: `internal/stages/git_test.go`
- Modify: `cmd/codeviz/viz_cmd_helpers.go` and callers

- [ ] **Step 1: Create git.go**

```go
package stages

import (
	"path/filepath"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/scan"
)

// CheckGitRequirementHelper verifies the target is inside a git repository
// when any requested metric needs git. No-op otherwise.
func CheckGitRequirementHelper(targetPath string, requested []metric.Name) error {
	name, needsGit := findGitMetric(requested)
	if !needsGit {
		return nil
	}

	return verifyGitRepo(targetPath, name)
}

// CheckGitRepoHelper verifies the target path is inside a git repository.
// Used by visualizations (such as spiral) that always require git.
func CheckGitRepoHelper(targetPath string) error {
	return verifyGitRepo(targetPath, "spiral")
}

func verifyGitRepo(targetPath string, metricLabel metric.Name) error {
	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return eris.Wrap(err, "failed to resolve absolute path")
	}

	isGit, err := scan.IsGitRepo(absPath)
	if err != nil {
		return eris.Wrap(err, "git check failed")
	}

	if !isGit {
		return &GitRequiredError{Metric: metricLabel, Target: targetPath}
	}

	return nil
}

func findGitMetric(requested []metric.Name) (metric.Name, bool) {
	for _, name := range requested {
		if git.IsGitMetric(name) {
			return name, true
		}
	}

	return "", false
}

// CheckGitRequirement is a pipeline.Stage wrapping CheckGitRequirementHelper.
func CheckGitRequirement[S VizState](s S) error {
	c := s.Common()
	return CheckGitRequirementHelper(c.TargetPath, c.Requested)
}

var _ pipeline.Stage[VizState] = CheckGitRequirement[VizState]
```

- [ ] **Step 2: Write git_test.go**

```go
package stages_test

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestCheckGitRequirementHelper_NoGitMetric_OK(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(stages.CheckGitRequirementHelper("/nonexistent", []metric.Name{"file-size"})).To(Succeed())
}

func TestCheckGitRequirement_Stage_SkipsWhenNoGitMetricRequested(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &fakeState{common: stages.CommonState{
		TargetPath: "/no/such/dir",
		Requested:  []metric.Name{"file-size"},
	}}

	g.Expect(stages.CheckGitRequirement[*fakeState](s)).To(Succeed())
}

func TestCheckGitRequirement_Stage_FailsWhenGitMetricRequestedAndNoRepo(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Use a path guaranteed not to be a git repo.
	dir := t.TempDir()

	s := &fakeState{common: stages.CommonState{
		TargetPath: dir,
		Requested:  []metric.Name{"file-age"}, // file-age is a git metric (internal/provider/git)
	}}
	err := stages.CheckGitRequirement[*fakeState](s)

	var gre *stages.GitRequiredError
	g.Expect(errors.As(err, &gre)).To(BeTrue())
}
```

- [ ] **Step 3: Delete `checkGitRequirement`, `checkGitRepo`, `verifyGitRepo`, `findGitMetric` from viz_cmd_helpers.go**

Replace each call site:
- `checkGitRequirement(c.TargetPath, requested)` → `stages.CheckGitRequirementHelper(c.TargetPath, requested)`
- `checkGitRepo(c.TargetPath)` → `stages.CheckGitRepoHelper(c.TargetPath)`

- [ ] **Step 4: CI**

Run: `task ci`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor(stages): move git-requirement helpers to internal/stages"
```

---

## Task 11: Move binary file filtering to `internal/stages`

**Files:**
- Create: `internal/stages/binary.go`
- Create: `internal/stages/binary_test.go`
- Modify: `cmd/codeviz/viz_cmd_helpers.go`, `cmd/codeviz/main.go` (move `countAll`)

- [ ] **Step 1: Create binary.go**

```go
package stages

import (
	"log/slog"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
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

	// Update root in place — avoid struct copy which would copy the mutex.
	root.Files = filtered.Files
	root.Dirs = filtered.Dirs

	return nil
}

// BinaryFilterToggler is implemented by per-viz state types that expose an
// "include binary files" flag. FilterBinaryFiles uses this to decide
// whether to run.
type BinaryFilterToggler interface {
	VizState
	IncludeBinary() bool
}

// FilterBinaryFiles is a pipeline.Stage that removes binary files from
// Common().Root unless the state's IncludeBinary() returns true.
func FilterBinaryFiles[S BinaryFilterToggler](s S) error {
	if s.IncludeBinary() {
		return nil
	}

	return FilterBinaryFilesHelper(s.Common().Root)
}

var _ pipeline.Stage[BinaryFilterToggler] = FilterBinaryFiles[BinaryFilterToggler]
```

- [ ] **Step 2: Write binary_test.go**

```go
package stages_test

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// fakeBinaryState satisfies BinaryFilterToggler for these tests.
type fakeBinaryState struct {
	common    stages.CommonState
	includeBin bool
}

func (f *fakeBinaryState) Common() *stages.CommonState { return &f.common }
func (f *fakeBinaryState) IncludeBinary() bool         { return f.includeBin }

func TestFilterBinaryFiles_IncludeFlagSet_NoOp(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Files: []*model.File{{Name: "a.bin"}, {Name: "b.go"}},
	}

	s := &fakeBinaryState{
		common:     stages.CommonState{Root: root},
		includeBin: true,
	}

	g.Expect(stages.FilterBinaryFiles[*fakeBinaryState](s)).To(Succeed())
	g.Expect(root.Files).To(HaveLen(2))
}

func TestFilterBinaryFiles_AllBinary_ReturnsNoFilesError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	// Build a tree of only binary files.
	root := &model.Directory{
		Files: []*model.File{
			{Name: "a.bin", IsBinary: true},
			{Name: "b.bin", IsBinary: true},
		},
	}

	s := &fakeBinaryState{common: stages.CommonState{Root: root}}
	err := stages.FilterBinaryFiles[*fakeBinaryState](s)

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

Note: the field is `model.File.IsBinary` (a `bool`), verified against `internal/model/file.go`.

- [ ] **Step 3: Delete `filterBinaryFiles` and `countAll` from cmd/codeviz**

- Remove `filterBinaryFiles` from `viz_cmd_helpers.go`.
- Remove `countAll` from `main.go`.
- Redirect callers: `filterBinaryFiles(root)` → `stages.FilterBinaryFilesHelper(root)`, `countAll(node)` → `stages.CountAll(node)`.

- [ ] **Step 4: CI**

Run: `task ci`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor(stages): move binary filtering and CountAll to internal/stages"
```

---

## Task 12: Move palette resolution and metric collection helpers

**Files:**
- Create: `internal/stages/metrics.go`
- Modify: `cmd/codeviz/viz_cmd_helpers.go`

- [ ] **Step 1: Create metrics.go**

```go
package stages

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// SpecMetric returns the metric name from a *MetricSpec, or "" if nil.
func SpecMetric(s *config.MetricSpec) metric.Name {
	if s == nil {
		return ""
	}
	return s.Metric
}

// SpecPalette returns the palette name from a *MetricSpec, or "" if nil.
func SpecPalette(s *config.MetricSpec) palette.PaletteName {
	if s == nil {
		return ""
	}
	return s.Palette
}

// CollectRequestedMetrics returns the unique ordered list of metric names
// implied by size + optional fill + optional border specs.
func CollectRequestedMetrics(size metric.Name, fill, border *config.MetricSpec) []metric.Name {
	seen := map[metric.Name]bool{size: true}
	names := []metric.Name{size}

	for _, spec := range []*config.MetricSpec{fill, border} {
		if spec != nil && spec.Metric != "" {
			if !seen[spec.Metric] {
				seen[spec.Metric] = true
				names = append(names, spec.Metric)
			}
		}
	}

	return names
}

// ResolveFillPalette returns the fill palette to use, consulting (in order)
// the explicit fill spec, the provider's default palette, and palette.Neutral.
func ResolveFillPalette(fill *config.MetricSpec, fillMetric metric.Name) palette.PaletteName {
	if fp := SpecPalette(fill); fp != "" {
		return fp
	}

	if d, ok := provider.GetDescriptor(fillMetric); ok {
		return d.DefaultPalette
	}

	return palette.Neutral
}

// ResolveBorderMetricAndPalette returns the effective border metric and
// palette name, or ("", "") when no border is configured.
func ResolveBorderMetricAndPalette(border *config.MetricSpec) (metric.Name, palette.PaletteName) {
	borderMetric := SpecMetric(border)
	if borderMetric == "" {
		return "", ""
	}

	borderPaletteName := SpecPalette(border)
	if borderPaletteName == "" {
		if d, ok := provider.GetDescriptor(borderMetric); ok {
			borderPaletteName = d.DefaultPalette
		} else {
			borderPaletteName = palette.Neutral
		}
	}

	return borderMetric, borderPaletteName
}
```

- [ ] **Step 2: Delete the same functions from `cmd/codeviz/viz_cmd_helpers.go` and `cmd/codeviz/treemap_cmd.go`**

In `viz_cmd_helpers.go` delete `resolveFillPalette`, `resolveBorderMetricAndPalette`. In `treemap_cmd.go` delete `specMetric`, `specPalette`, `collectRequestedMetrics` (search for which file they live in — they may all be in `viz_cmd_helpers.go`).

Redirect all callers:
- `resolveFillPalette(...)` → `stages.ResolveFillPalette(...)`
- `resolveBorderMetricAndPalette(...)` → `stages.ResolveBorderMetricAndPalette(...)`
- `specMetric(...)` → `stages.SpecMetric(...)`
- `specPalette(...)` → `stages.SpecPalette(...)`
- `collectRequestedMetrics(...)` → `stages.CollectRequestedMetrics(...)`

- [ ] **Step 3: CI**

Run: `task ci`
Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add -A
git commit -m "refactor(stages): move palette and metric helpers to internal/stages"
```

---

## Task 13: Move progress wiring to `internal/stages`

**Files:**
- Create: `internal/stages/progress.go`
- Modify: `cmd/codeviz/progress.go` — leave only `buildHistoryProgress` and its helpers (spiral-only).
- Modify: `cmd/codeviz/viz_cmd_helpers.go` and callers.

- [ ] **Step 1: Move scan + metric progress code**

Copy from `cmd/codeviz/progress.go` into `internal/stages/progress.go` the following, capitalising names where they cross the package boundary:

- `BuildScanProgress(*Flags) (scan.Progress, func())`
- `BuildMetricProgress(*Flags, int) (provider.MetricProgress, func())`
- private types `scanCounter`, `metricProgressTracker`
- private helpers `startScanTicker`, `startMetricTicker`, `startProgressTicker`, `logMetricProgress`, `removeMetric`

Note: `BuildScanProgress` and `BuildMetricProgress` now take `*stages.Flags`, not the cmd/codeviz `*Flags`. Update their signatures.

Use this exact public surface:

```go
func BuildScanProgress(flags *Flags) (scan.Progress, func())
func BuildMetricProgress(flags *Flags, totalFiles int) (provider.MetricProgress, func())
```

- [ ] **Step 2: Delete the moved code from `cmd/codeviz/progress.go`**

Keep `buildHistoryProgress`, `startHistoryTicker`, and any of its dedicated helpers (they're only used by spiral and stay until that viz is refactored in a follow-up spec).

- [ ] **Step 3: Adapt `*Flags` translation at call sites**

In each `*_cmd.go` that calls `buildScanProgress(flags)`, we need a `*stages.Flags`. The simplest route: change call sites once the orchestrator code exists in Task 17. For *this* task, add a small adapter in `cmd/codeviz/main.go`:

```go
// toStagesFlags converts the cmd-local Flags struct into the stages-package form.
func toStagesFlags(f *Flags) *stages.Flags {
	return &stages.Flags{
		Quiet:        f.Quiet,
		Verbose:      f.Verbose,
		Debug:        f.Debug,
		ExportConfig: f.ExportConfig,
		ExportData:   f.ExportData,
		Config:       f.Config,
	}
}
```

Then update each `buildScanProgress(flags)` → `stages.BuildScanProgress(toStagesFlags(flags))` and `buildMetricProgress(flags, n)` → `stages.BuildMetricProgress(toStagesFlags(flags), n)`.

- [ ] **Step 4: CI**

Run: `task ci`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor(stages): move scan/metric progress wiring to internal/stages"
```

---

## Task 14: Implement `ScanFilesystem`, `RunProviders`, `ExportConfig`, `ExportData`, `ResolveDimensions`, `WriteCanvas` stages

**Files:**
- Create: `internal/stages/scan.go`
- Create: `internal/stages/providers.go`
- Create: `internal/stages/export.go`
- Create: `internal/stages/dimensions.go`
- Create: `internal/stages/canvas.go`
- Create: `internal/stages/scan_test.go` (covers stages with branching behaviour)

- [ ] **Step 1: scan.go**

```go
package stages

import (
	"log/slog"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/scan"
)

// ScanFilesystem walks Common().TargetPath, populates Common().Root, and
// wires progress reporting based on Flags verbosity.
func ScanFilesystem[S VizState](s S) error {
	c := s.Common()

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

var _ pipeline.Stage[VizState] = ScanFilesystem[VizState]
```

- [ ] **Step 2: providers.go**

```go
package stages

import (
	"log/slog"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
)

// RunProviders calculates Common().Requested metrics against Common().Root,
// wiring progress reporting based on Flags verbosity.
func RunProviders[S VizState](s S) error {
	c := s.Common()

	slog.Info("Calculating metrics")

	metricProg, stopMetricTicker := BuildMetricProgress(c.Flags, model.CountFiles(c.Root))

	if err := provider.Run(c.Root, c.Requested, metricProg); err != nil {
		stopMetricTicker()
		return eris.Wrap(err, "failed to load metrics")
	}

	stopMetricTicker()
	return nil
}

var _ pipeline.Stage[VizState] = RunProviders[VizState]
```

- [ ] **Step 3: export.go**

```go
package stages

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/export"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
)

// ExportConfig writes the merged effective config to disk when
// Flags.ExportConfig is non-empty.
func ExportConfig[S VizState](s S) error {
	c := s.Common()
	if c.Flags.ExportConfig == "" {
		return nil
	}

	if err := c.RootConfig.Save(c.Flags.ExportConfig); err != nil {
		return eris.Wrap(err, "failed to save config")
	}

	return nil
}

// ExportData writes computed metric data to disk when Flags.ExportData is
// non-empty.
func ExportData[S VizState](s S) error {
	c := s.Common()
	if err := export.Export(c.Root, c.Requested, c.Flags.ExportData); err != nil {
		return eris.Wrap(err, "failed to export data")
	}
	return nil
}

var (
	_ pipeline.Stage[VizState] = ExportConfig[VizState]
	_ pipeline.Stage[VizState] = ExportData[VizState]
)
```

- [ ] **Step 4: dimensions.go**

```go
package stages

import "github.com/theunrepentantgeek/code-visualizer/internal/pipeline"

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

// ResolveDimensions populates Common().Width and Common().Height from
// RootConfig, applying the documented defaults (1920x1080).
func ResolveDimensions[S VizState](s S) error {
	c := s.Common()
	c.Width = PtrInt(c.RootConfig.Width, 1920)
	c.Height = PtrInt(c.RootConfig.Height, 1080)
	return nil
}

var _ pipeline.Stage[VizState] = ResolveDimensions[VizState]
```

- [ ] **Step 5: canvas.go**

```go
package stages

import (
	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
)

// WriteCanvas writes Common().Canvas to Common().Output.
func WriteCanvas[S VizState](s S) error {
	c := s.Common()
	if err := c.Canvas.Render(c.Output); err != nil {
		return eris.Wrap(err, "render failed")
	}
	return nil
}

var _ pipeline.Stage[VizState] = WriteCanvas[VizState]
```

- [ ] **Step 6: scan_test.go — cover branching stages**

```go
package stages_test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestExportConfig_NoFlag_NoOp(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &fakeState{common: stages.CommonState{
		Flags:      &stages.Flags{ExportConfig: ""},
		RootConfig: config.New(),
	}}

	g.Expect(stages.ExportConfig[*fakeState](s)).To(Succeed())
}

func TestResolveDimensions_AppliesDefaults(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &fakeState{common: stages.CommonState{RootConfig: &config.Config{}}}

	g.Expect(stages.ResolveDimensions[*fakeState](s)).To(Succeed())
	g.Expect(s.Common().Width).To(Equal(1920))
	g.Expect(s.Common().Height).To(Equal(1080))
}

func TestResolveDimensions_UsesConfigValues(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	w, h := 800, 600
	s := &fakeState{common: stages.CommonState{
		RootConfig: &config.Config{Width: &w, Height: &h},
	}}

	g.Expect(stages.ResolveDimensions[*fakeState](s)).To(Succeed())
	g.Expect(s.Common().Width).To(Equal(800))
	g.Expect(s.Common().Height).To(Equal(600))
}

// Smoke test: ScanFilesystem against an empty tempdir.
func TestScanFilesystem_EmptyDir(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	// Touch a file so the scan isn't completely empty.
	g.Expect(writeFile(filepath.Join(dir, "x.txt"), "hi")).To(Succeed())

	s := &fakeState{common: stages.CommonState{
		TargetPath: dir,
		Flags:      &stages.Flags{},
	}}

	g.Expect(stages.ScanFilesystem[*fakeState](s)).To(Succeed())
	g.Expect(s.Common().Root).NotTo(BeNil())
}

func writeFile(path, content string) error {
	return os.WriteFile(path, []byte(content), 0o644)
}
```

Add the missing `import "os"`. Engineer note: if the `config.Config` struct's `Width`/`Height` field types differ from `*int`, adjust the test accordingly.

- [ ] **Step 7: CI**

Run: `task ci`
Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add -A internal/stages/
git commit -m "feat(stages): implement scan, providers, export, dimensions, canvas stages"
```

---

## Task 15: Move treemap canvas helpers into the `internal/treemap` package

**Files:**
- Create: `internal/treemap/inks.go`
- Create: `internal/treemap/render.go`
- Modify: `cmd/codeviz/treemap_canvas.go` — replaced with thin re-exports during transition, then deleted in Task 17.
- Modify: `cmd/codeviz/treemap_canvas_test.go` — move with the code.

- [ ] **Step 1: Move types and functions to the package**

Cut the contents of `cmd/codeviz/treemap_canvas.go` into `internal/treemap/inks.go` (the `treemapInks` type + `buildTreemapInks` function) and `internal/treemap/render.go` (the `renderTreemapToCanvas` function and its helpers). Export them as `Inks`, `BuildInks`, `RenderToCanvas`. Update package declaration to `package treemap`.

- [ ] **Step 2: Move tests**

Move `cmd/codeviz/treemap_canvas_test.go` to `internal/treemap/render_test.go` (package `treemap_test`). Update references: `buildTreemapInks` → `treemap.BuildInks`, `renderTreemapToCanvas` → `treemap.RenderToCanvas`, `treemapInks` → `treemap.Inks`.

- [ ] **Step 3: Update treemap_cmd.go to call the new locations**

In `cmd/codeviz/treemap_cmd.go`, replace:
- `buildTreemapInks(...)` → `treemap.BuildInks(...)`
- `renderTreemapToCanvas(...)` → `treemap.RenderToCanvas(...)`
- `treemapInks` → `treemap.Inks`

Add `"github.com/theunrepentantgeek/code-visualizer/internal/treemap"` import (likely already present for `treemap.Layout`).

Delete `cmd/codeviz/treemap_canvas.go`.

- [ ] **Step 4: CI + golden tests**

Run: `task ci`
Expected: PASS (golden treemap PNG/SVG tests must pass byte-for-byte).

- [ ] **Step 5: Commit**

```bash
git add -A
git commit -m "refactor(treemap): move ink and render helpers into internal/treemap"
```

---

## Task 16: Define treemap `State`, viz stages, and rewrite `TreemapCmd.Run`

**Files:**
- Create: `internal/treemap/state.go`
- Create: `internal/treemap/stages.go`
- Create: `internal/treemap/stages_test.go`
- Modify: `cmd/codeviz/treemap_cmd.go`

- [ ] **Step 1: Create state.go**

```go
package treemap

import (
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// State is the pipeline state for the treemap visualization.
type State struct {
	stages.CommonState

	Config             *config.Treemap
	IncludeBinaryFiles bool

	// Resolved during the pipeline:
	Size          metric.Name
	FillMetric    metric.Name
	FillPalette   palette.PaletteName
	BorderMetric  metric.Name
	BorderPalette palette.PaletteName
	Inks          Inks
	Root          TreemapRectangle // root of laid-out rectangles
	LegendConfig  *canvas.LegendConfig
}

func (s *State) Common() *stages.CommonState { return &s.CommonState }

// IncludeBinary lets State satisfy stages.BinaryFilterToggler.
func (s *State) IncludeBinary() bool { return s.IncludeBinaryFiles }
```

> Note: `Layout` returns a single `TreemapRectangle` (the laid-out root), not a slice. `OffsetRects` takes `*TreemapRectangle`. The field is named `Root` to distinguish it from `Common().Root` (the directory tree).

- [ ] **Step 2: Create stages.go**

```go
package treemap

import (
	"log/slog"

	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ResolveMetrics resolves size, fill, and border metrics + palettes and
// fills Common().Requested.
func ResolveMetrics(s *State) error {
	cfg := s.Config

	s.Size = metric.Name(stages.PtrString(cfg.Size))
	s.FillMetric = resolveFillMetric(cfg)
	s.FillPalette = stages.ResolveFillPalette(cfg.Fill, s.FillMetric)
	s.BorderMetric, s.BorderPalette = stages.ResolveBorderMetricAndPalette(cfg.Border)

	s.Common().Requested = stages.CollectRequestedMetrics(s.Size, cfg.Fill, cfg.Border)
	return nil
}

func resolveFillMetric(cfg *config.Treemap) metric.Name {
	if fill := stages.SpecMetric(cfg.Fill); fill != "" {
		return fill
	}
	return metric.Name(stages.PtrString(cfg.Size))
}

// BuildInksStage builds the treemap inks.
func BuildInksStage(s *State) error {
	s.Inks = BuildInks(s.Common().Root, s.FillMetric, s.FillPalette, s.BorderMetric, s.BorderPalette)
	return nil
}

// BuildLegendStage builds the legend config from inks.
func BuildLegendStage(s *State) error {
	pos, orient := legend.ResolveOptions(stages.PtrString(s.Config.Legend), stages.PtrString(s.Config.LegendOrientation))
	s.LegendConfig = legend.Build(
		pos, orient,
		s.Inks.Fill, s.FillMetric,
		s.Inks.Border, s.BorderMetric,
		s.Size,
	)
	return nil
}

// LayoutStage reserves legend space, lays out rectangles, and applies the
// resulting offset.
func LayoutStage(s *State) error {
	c := s.Common()
	layoutW, layoutH := legend.ReserveAndLayout(s.LegendConfig, c.Width, c.Height)

	rect := Layout(c.Root, layoutW, layoutH, s.Size)

	if layoutW < c.Width || layoutH < c.Height {
		if s.LegendConfig != nil {
			wReduce, hReduce := s.LegendConfig.ReserveSpace()
			dx, dy := legend.LayoutOffset(s.LegendConfig, wReduce, hReduce)
			OffsetRects(&rect, dx, dy)
		}
	}

	s.Root = rect
	return nil
}

// RenderStage renders the treemap to a canvas and attaches the legend.
func RenderStage(s *State) error {
	c := s.Common()

	slog.Info("Rendering image", "output", c.Output, "width", c.Width, "height", c.Height)

	cv := RenderToCanvas(s.Root, c.Root, c.Width, c.Height, s.Inks)
	if s.LegendConfig != nil {
		cv.SetLegend(*s.LegendConfig)
	}

	c.Canvas = cv
	return nil
}

// LogResult logs the final summary.
func LogResult(s *State) error {
	c := s.Common()
	files, dirs := stages.CountAll(c.Root)

	slog.Info(
		"Rendered treemap",
		"files", files,
		"directories", dirs,
		"output", c.Output,
		"width", c.Width,
		"height", c.Height,
		"size_metric", string(s.Size),
		"fill_metric", string(s.FillMetric),
		"fill_palette", string(s.FillPalette),
		"border_metric", string(s.BorderMetric),
		"border_palette", string(s.BorderPalette),
	)
	return nil
}

// Compile-time checks.
var (
	_ pipeline.Stage[*State] = ResolveMetrics
	_ pipeline.Stage[*State] = BuildInksStage
	_ pipeline.Stage[*State] = BuildLegendStage
	_ pipeline.Stage[*State] = LayoutStage
	_ pipeline.Stage[*State] = RenderStage
	_ pipeline.Stage[*State] = LogResult
)
```

Add a `config` import to support the `resolveFillMetric` parameter type.

- [ ] **Step 3: Create stages_test.go**

```go
package treemap_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
	"github.com/theunrepentantgeek/code-visualizer/internal/treemap"
)

func TestResolveMetrics_SizeOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	s := &treemap.State{
		Config: &config.Treemap{Size: &sizeStr},
	}

	g.Expect(treemap.ResolveMetrics(s)).To(Succeed())
	g.Expect(s.Size).To(Equal(metric.Name("file-size")))
	g.Expect(s.FillMetric).To(Equal(metric.Name("file-size"))) // falls back to size
	g.Expect(s.Common().Requested).To(ConsistOf(metric.Name("file-size")))
}

func TestResolveMetrics_FillOverridesSizeAsFillMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	s := &treemap.State{
		Config: &config.Treemap{
			Size: &sizeStr,
			Fill: &config.MetricSpec{Metric: "file-type"},
		},
	}

	g.Expect(treemap.ResolveMetrics(s)).To(Succeed())
	g.Expect(s.FillMetric).To(Equal(metric.Name("file-type")))
	g.Expect(s.Common().Requested).To(ContainElements(metric.Name("file-size"), metric.Name("file-type")))
}

// Sanity: State.Common returns a pointer to embedded state.
func TestState_CommonReturnsEmbeddedPointer(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &treemap.State{}
	c := s.Common()
	c.Width = 42
	g.Expect(s.CommonState.Width).To(Equal(42))
}

// IncludeBinary plumbs through correctly.
func TestState_IncludeBinary(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	on := &treemap.State{IncludeBinaryFiles: true}
	off := &treemap.State{IncludeBinaryFiles: false}
	g.Expect(on.IncludeBinary()).To(BeTrue())
	g.Expect(off.IncludeBinary()).To(BeFalse())

	// And satisfies the stages constraint:
	var _ stages.BinaryFilterToggler = on
}
```

- [ ] **Step 4: Rewrite `TreemapCmd.Run`**

Replace the entire `Run` method and its helper `renderAndLog` in `cmd/codeviz/treemap_cmd.go` with:

```go
func (c *TreemapCmd) Run(flags *Flags) error {
	if err := c.mergeConfigAndValidate(flags); err != nil {
		return err
	}

	state := &treemap.State{
		CommonState: stages.CommonState{
			TargetPath: c.TargetPath,
			Output:     c.Output,
			Flags:      toStagesFlags(flags),
			RootConfig: flags.Config,
			CLIFilters: c.Filter,
		},
		Config:             flags.Config.Treemap,
		IncludeBinaryFiles: c.IncludeBinaryFiles,
	}

	_, err := pipeline.Run(state,
		stages.ValidatePaths[*treemap.State],
		stages.ExportConfig[*treemap.State],
		stages.BuildFilterRules[*treemap.State],
		treemap.ResolveMetrics,
		stages.ScanFilesystem[*treemap.State],
		stages.CheckGitRequirement[*treemap.State],
		stages.RunProviders[*treemap.State],
		stages.FilterBinaryFiles[*treemap.State],
		stages.ExportData[*treemap.State],
		stages.ResolveDimensions[*treemap.State],
		treemap.BuildInksStage,
		treemap.BuildLegendStage,
		treemap.LayoutStage,
		treemap.RenderStage,
		stages.WriteCanvas[*treemap.State],
		treemap.LogResult,
	)
	return err
}
```

Add imports: `"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"`, `"github.com/theunrepentantgeek/code-visualizer/internal/stages"`, `"github.com/theunrepentantgeek/code-visualizer/internal/treemap"`.

Delete `renderAndLog`, `resolveFillMetric` (now in `internal/treemap`), and any other helpers that were only used by the old `Run`.

- [ ] **Step 5: CI + golden tests**

Run: `task ci`
Expected: PASS. The treemap golden PNG/SVG tests are the integration safety net — they MUST pass byte-for-byte.

If a golden test fails, do NOT regenerate the fixture. Diagnose the regression: it almost certainly indicates a behaviour change introduced by the refactor (most likely an ordering bug where `ResolveMetrics` ran after `RunProviders`, or `FilterBinaryFiles` ran before the binary flag was honoured).

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "refactor(treemap): rewrite TreemapCmd.Run as pipeline composition"
```

---

## Task 17: Final cleanup of `cmd/codeviz`

**Files:**
- Modify: `cmd/codeviz/viz_cmd_helpers.go` — should now be minimal.
- Modify: `cmd/codeviz/main.go` — `countAll` already gone; verify only orchestration remains.
- Modify: `cmd/codeviz/progress.go` — confirm only history progress remains.

- [ ] **Step 1: Audit `viz_cmd_helpers.go`**

Run: `cat cmd/codeviz/viz_cmd_helpers.go`

The file should now contain only helpers that haven't yet been refactored *and* are still used by bubbletree, radialtree, or spiral commands. Anything unused: delete. Anything still in use whose stages-package equivalent exists: replace the body with a direct call to the stages-package version.

- [ ] **Step 2: Audit unused imports**

Run: `task lint`
Expected: PASS. If any unused imports remain, remove them. The linter will flag them.

- [ ] **Step 3: Audit linter `nolint:dupl` annotations**

In `treemap_cmd.go` the `//nolint:dupl,revive,cyclop,funlen // Run methods share workflow structure across visualization commands` comment on `Run` is no longer accurate (the duplication is gone). Remove it. The function is short; lint rules should pass without suppression.

- [ ] **Step 4: Full CI**

Run: `task ci`
Expected: PASS. Lint clean.

- [ ] **Step 5: Verify treemap end-to-end smoke**

Run: `./bin/codeviz render treemap . -o /tmp/treemap_smoke.png`
Expected: file written, exit 0, log lines match historical output.

- [ ] **Step 6: Commit**

```bash
git add -A
git commit -m "refactor: clean up cmd/codeviz after pipeline migration"
```

---

## Verification

After Task 17, the following must all be true:

- [ ] `task ci` passes.
- [ ] `cmd/codeviz/treemap_cmd.go` `Run()` body is a single `pipeline.Run(...)` call (~20 lines including state construction).
- [ ] `cmd/codeviz/legend_builder.go` and `cmd/codeviz/legend_builder_test.go` do not exist.
- [ ] `cmd/codeviz/treemap_canvas.go` does not exist (moved to `internal/treemap`).
- [ ] `internal/pipeline/pipeline_test.go` exists and passes.
- [ ] All treemap golden tests pass with the original fixtures.
- [ ] `./bin/codeviz render treemap <repo>` produces identical output to before the refactor (modulo identical render).

If any of the above fail, do not proceed to declaring complete.
