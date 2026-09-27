# Pipeline Progress Boundaries Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace PR #763's descriptor-driven command progress executor with explicit, failure-safe progress boundaries around the existing visualization pipelines.

**Architecture:** `cmd/codeviz` will own a small boundary object that begins an already-known operation, starts one named stage, injects its typed active sink, and explicitly resolves that stage outside `pipeline.ApplyFunc*`. The six ordinary visualization commands and Alluvial will call those operations beside their current pipeline sequences; `runApplication` will become the sole reporter-close owner.

**Tech Stack:** Go 1.26.1, Kong, `log/slog`, Gomega, existing `internal/pipeline` and `internal/progress` packages

**Spec:** `docs/superpowers/specs/2026-09-27-pipeline-progress-boundaries-design.md`

## Global Constraints

- Run every repository command through `./dev.sh -c '<command>'`.
- Change only command orchestration and directly related tests; do not change visualization algorithms, renderers, progress flags, output formats, or generic pipeline execution semantics.
- Retain `context.Context` and the active `progress.Sink` as separate typed values in `pipeline.State`; do not add either to `stages.CommonState`.
- Do not make `pipeline.ApplyFunc*` progress-aware or use an apply helper to call `Complete`, `Fail`, or `Cancel`.
- Preserve `internal/progress` lifecycle validation, plain/TTY behavior, diagnostic-writer coordination, and existing cancellation/error exit classification.
- Keep processing errors discoverable through `errors.Is`/`errors.As` when terminal reporter and close errors are joined.
- `runApplication` is the only reporter-close owner after successful reporter construction; close exactly once and retain a close error.
- Ordinary commands expose exactly four curated stages: `Preparing`, `Acquiring data`, `Rendering`, and `Writing output`.
- Alluvial exposes exactly `len(cfg.References) + 3` stages: `Preparing`, ordered `Loading <reference>` stages (empty reference displays as `HEAD`), `Rendering`, and `Writing output`.
- Replace a finished stage in `pipeline.State` with an inactive typed `progress.Sink` that returns lifecycle errors; do not silently ignore stale updates.

## Review Focus

- A pipeline error set by an `ApplyFunc*` must still terminally fail the already-started stage; Task 1 adds `TestProgressBoundaries_EndsFailedStageAfterPipelineError`.
- Cancellation detected after a processing group but before terminal resolution must cancel, not complete, that stage; Task 1 adds `TestProgressBoundaries_CancelsWhenContextEndsAtBoundary`.
- A configuration failure before the first stage must still close the constructed reporter exactly once; Task 2 adds `TestRunApplication_ClosesReporterAfterPreBoundaryCommandFailure`.
- A finished stage and the pipeline's replacement sink must reject late updates without producing a later-stage event; Task 1 adds `TestProgressBoundaries_ReplacesFinishedSinkWithInactiveSink`.
- An Alluvial command with an empty final reference must use the exact dynamic count and ordered `Loading HEAD` name; Task 4 replaces the phase-slice test with `TestAlluvialCmd_ReportsOrderedReferenceBoundaries`.

---

## File Map

| File | Responsibility |
|---|---|
| `cmd/codeviz/progress_boundaries.go` | Defines the command-only boundary lifecycle, inactive typed sink, four shared phase labels, and reporter fallback for direct command tests. |
| `cmd/codeviz/progress_boundaries_test.go` | Tests lifecycle ordering, terminal outcomes outside pipeline short-circuiting, joined errors, cancellation, and stale-sink rejection using a reporter double. |
| `cmd/codeviz/workflow.go` and `cmd/codeviz/workflow_test.go` | Remain temporarily during ordinary-command migration, then are deleted once Alluvial no longer references the descriptor executor. |
| `cmd/codeviz/main.go` | Closes every constructed reporter once after command dispatch, joins its error, and exposes a narrow reporter-construction seam for application tests. |
| `cmd/codeviz/main_test.go` | Verifies reporter closure on success and pre-boundary command failure through `runApplication`. |
| `cmd/codeviz/{treemap,radialtree,donuttree,bubbletree,spiral,scatter}_cmd.go` | Replaces phase-array callbacks with visible begin/start/end boundary calls around the unchanged four processing groups. |
| `cmd/codeviz/alluvial_cmd.go` | Replaces `buildAlluvialPhases` with direct preparation, one ordered reference boundary per configured reference, rendering, and writing. |
| `cmd/codeviz/alluvial_cmd_test.go` | Verifies Alluvial's dynamic count and stage names through the boundary lifecycle rather than a detached descriptor slice. |
| `cmd/codeviz/progress_integration_test.go` | Retains plain-mode coverage and adds a preset assertion that one delegated visualization creates one four-stage operation. |

## Interfaces

Task 1 produces these command-private interfaces in
`cmd/codeviz/progress_boundaries.go`:

```go
const (
    phasePreparing = "Preparing"
    phaseAcquiring = "Acquiring data"
    phaseRendering = "Rendering"
    phaseWriting   = "Writing output"
)

type progressBoundaries struct {
    ctx      context.Context
    reporter progress.Reporter
    state    *pipeline.State
    active   progress.Stage
}

func newProgressBoundaries(
    flags *Flags,
    state *pipeline.State,
    title string,
    stageCount int,
) (*progressBoundaries, error)

func (b *progressBoundaries) Start(
    name string,
    kind progress.StageKind,
    work progress.WorkKind,
) error

func (b *progressBoundaries) End() error
func (b *progressBoundaries) Finish() error
```

`Start` stores its `progress.Stage` as `pipeline.Set[progress.Sink]`.
`End` reads `state.Err()` and `ctx.Err()`, calls exactly one terminal `Stage`
method without an apply helper, and always stores an inactive implementation of
`progress.Sink` before returning. `Finish` delegates only to
`Reporter.Finish`.

Task 2 adds this optional application test seam:

```go
type application struct {
    // existing fields ...
    newReporter func(progress.Config) (progress.Reporter, error)
}

func (app application) makeReporter(config progress.Config) (progress.Reporter, error)
```

`makeReporter` calls `progress.New` when `newReporter` is nil. Production
callers leave the field nil.

## Tasks

### Task 1: Add explicit progress-boundary lifecycle support

**Files:**
- Create: `cmd/codeviz/progress_boundaries.go`
- Create: `cmd/codeviz/progress_boundaries_test.go`

**Interfaces:**
- Consumes: `Flags.Context`, `Flags.Reporter`, `pipeline.Set[T]`,
  `pipeline.State.Err()`, `progress.Reporter`, `progress.Stage`, and
  `progress.Sink`.
- Produces: `progressBoundaries`, `newProgressBoundaries`, `Start`, `End`,
  and `Finish` exactly as declared in the Interfaces section.

- [ ] **Step 1: Write failing boundary lifecycle tests**

Create a reporter/stage double that records begin, stage start, terminal
outcomes, finish, and injected work kinds. Add:

```go
func TestProgressBoundaries_StartsAndCompletesStages(t *testing.T)
func TestProgressBoundaries_EndsFailedStageAfterPipelineError(t *testing.T)
func TestProgressBoundaries_CancelsWhenContextEndsAtBoundary(t *testing.T)
func TestProgressBoundaries_ReplacesFinishedSinkWithInactiveSink(t *testing.T)
func TestProgressBoundaries_JoinsProcessingAndTerminalErrors(t *testing.T)
```

The failure test must set `State.Err()` through `pipeline.ApplyFuncX` before
calling `End`, proving the failure terminal event is not skipped. The stale
sink test must assert both a retained stage and a sink read from a subsequent
`ApplyFuncX` return explicit errors after `End`.

- [ ] **Step 2: Run the focused boundary tests to verify they fail**

Run: `./dev.sh -c 'go test ./cmd/codeviz -run "^TestProgressBoundaries_" -count=1'`

Expected: FAIL because the boundary implementation and tests do not yet
exist.

- [ ] **Step 3: Implement the boundary API**

Implement the exact signatures in `progress_boundaries.go`. Preserve the
current quiet, discard-writer reporter fallback when `Flags.Reporter` is nil,
and use `context.Background()` when `Flags.Context` is nil. `Start` must
return an error when a stage is already active or reporter startup fails.

Implement `End() error` so it:

1. selects `state.Err()` first, then `ctx.Err()` when no pipeline error
   exists;
2. calls `Cancel` for `context.Canceled` or `context.DeadlineExceeded`,
   `Fail` for other errors, and `Complete` otherwise;
3. joins a processing error with a terminal reporter error using
   `errors.Join`;
4. stores the inactive `progress.Sink` even if the terminal reporter call
   fails.

The inactive sink's `WorkKind` is `progress.WorkNone`; all three mutation
methods return the same explicit inactive-stage error. Do not add any
progress behavior to `internal/pipeline`.

- [ ] **Step 4: Run the focused boundary tests**

Run: `./dev.sh -c 'go test ./cmd/codeviz -run "^TestProgressBoundaries_" -count=1'`

Expected: PASS.

- [ ] **Step 5: Commit the boundary lifecycle**

```bash
git add cmd/codeviz/progress_boundaries.go cmd/codeviz/progress_boundaries_test.go
git commit -m "refactor: add progress boundaries" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 2: Move reporter closure to application ownership

**Files:**
- Modify: `cmd/codeviz/main.go: application and runApplication`
- Modify: `cmd/codeviz/main_test.go: runApplication coverage`

**Interfaces:**
- Consumes: `application.makeReporter(config progress.Config) (progress.Reporter, error)`
  and a test reporter implementing the existing `progress.Reporter` interface.
- Produces: one `reporter.Close()` call after successful construction, with its
  error joined to command dispatch errors before `classifyError`.

- [ ] **Step 1: Write failing application-close tests**

Add a configurable reporter double and these tests:

```go
func TestRunApplication_ClosesReporterAfterSuccessfulCommand(t *testing.T)
func TestRunApplication_ClosesReporterAfterPreBoundaryCommandFailure(t *testing.T)
func TestRunApplication_ReturnsCloseFailure(t *testing.T)
```

Inject it through `application.newReporter`. The pre-boundary case uses an
invalid visualization configuration that fails during command validation, then
asserts one close. The close-failure case uses `help metrics`, asserts exit
code `5`, and checks the close error is emitted rather than reporting success.

- [ ] **Step 2: Run the focused application-close tests to verify they fail**

Run: `./dev.sh -c 'go test ./cmd/codeviz -run "^TestRunApplication_(ClosesReporter|ReturnsCloseFailure)" -count=1'`

Expected: FAIL because `application` cannot inject reporter construction and
does not close pre-boundary reporters.

- [ ] **Step 3: Implement application-owned closure**

Add `newReporter` and `makeReporter` with the exact signatures above. Replace
the direct `progress.New` call with `app.makeReporter`. After reporter
construction succeeds, join `reporter.Close()` with the result of configuration
loading or `ctx.Run(flags)` before logging and classifying it. Preserve parse
and reporter-construction behavior, because neither has a reporter to close.

- [ ] **Step 4: Run the focused application-close tests**

Run: `./dev.sh -c 'go test ./cmd/codeviz -run "^TestRunApplication_(ClosesReporter|ReturnsCloseFailure)" -count=1'`

Expected: PASS.

- [ ] **Step 5: Commit application close ownership**

```bash
git add cmd/codeviz/main.go cmd/codeviz/main_test.go
git commit -m "refactor: close progress reporter in application" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 3: Make ordinary visualization progress boundaries explicit

**Files:**
- Modify: `cmd/codeviz/treemap_cmd.go`
- Modify: `cmd/codeviz/radialtree_cmd.go`
- Modify: `cmd/codeviz/donuttree_cmd.go`
- Modify: `cmd/codeviz/bubbletree_cmd.go`
- Modify: `cmd/codeviz/spiral_cmd.go`
- Modify: `cmd/codeviz/scatter_cmd.go`
- Modify: `cmd/codeviz/progress_integration_test.go`

**Interfaces:**
- Consumes: `newProgressBoundaries(flags, state, title, 4)`,
  `(*progressBoundaries).Start`, `End`, and `Finish`.
- Produces: six commands whose visible execution order is four direct
  boundaries around their existing prepare, acquire, render, and write calls.

- [ ] **Step 1: Capture the behavior-preserving ordinary-command baseline**

Add `TestRunApplication_PresetUsesSingleProgressOperation`, running
`render structure-tree-map` in plain mode against a one-file target. Assert
that stderr contains one `Tree map` operation and exactly one each of
`[1/4] Preparing: started`, `[2/4] Acquiring data: started`,
`[3/4] Rendering: started`, and `[4/4] Writing output: started`. This pins
the no-nesting requirement while the task changes only command-side
composition; Task 1 supplies the new lifecycle behavior tests.

Run: `./dev.sh -c 'go test ./cmd/codeviz -run "TestRunApplication_(PlainAndRedirectedAutoUsePlainProgress|PreCancelledContextReturns130|ProcessingFailureLeavesDurableFailedStage|PresetUsesSingleProgressOperation)$|TestTreemapCmd_Run_WritesFileLabelsIntoSVG|TestRenderCmd_" -count=1'`

Expected: PASS before migration.

- [ ] **Step 2: Replace phase slices in the six ordinary commands**

For each listed command, preserve its existing pipeline calls and their order.
After state construction:

1. call `newProgressBoundaries` with stage count `4` and, respectively,
   `"Tree map"`, `"Radial tree"`, `"Donut tree"`, `"Bubble tree"`,
   `"Spiral"`, or `"Scatter plot"`;
2. explicitly start and end `Preparing` around validation/configuration/metric
   resolution;
3. only when preparation ended successfully, start `Acquiring data` with
   `stages.AcquisitionWork(common)`, then run the existing `AcquireData`;
4. explicitly start/end summary `Rendering` and `Writing output` around their
   unchanged functions;
5. call `Finish` only after all four `End` calls succeed.

Return the same command-specific `eris.Wrap` context currently used. Do not
extract a callback runner or descriptor slice; the visible calls in each
command are the intended composition boundary.

- [ ] **Step 3: Run ordinary command and integration tests**

Run: `./dev.sh -c 'go test ./cmd/codeviz -run "TestRunApplication_(PlainAndRedirectedAutoUsePlainProgress|PreCancelledContextReturns130|ProcessingFailureLeavesDurableFailedStage|PresetUsesSingleProgressOperation)$|TestTreemapCmd_Run_WritesFileLabelsIntoSVG|TestRenderCmd_" -count=1'`

Expected: PASS; the preset assertions confirm dispatch still reaches exactly
one selected visualization workflow.

- [ ] **Step 4: Commit ordinary command orchestration**

```bash
git add cmd/codeviz/treemap_cmd.go cmd/codeviz/radialtree_cmd.go cmd/codeviz/donuttree_cmd.go cmd/codeviz/bubbletree_cmd.go cmd/codeviz/spiral_cmd.go cmd/codeviz/scatter_cmd.go cmd/codeviz/progress_integration_test.go
git commit -m "refactor: inline visualization progress boundaries" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 4: Inline Alluvial's ordered reference boundaries

**Files:**
- Modify: `cmd/codeviz/alluvial_cmd.go`
- Modify: `cmd/codeviz/alluvial_cmd_test.go`
- Delete: `cmd/codeviz/workflow.go`
- Delete: `cmd/codeviz/workflow_test.go`

**Interfaces:**
- Consumes: `newProgressBoundaries(flags, s, "Alluvial", len(cfg.References)+3)`
  and `alluvial.SnapshotReference(reference) string`.
- Produces: direct Alluvial boundaries with ordered live `Loading <reference>`
  stages and no `buildAlluvialPhases` function.

- [ ] **Step 1: Replace the phase-descriptor test with a failing boundary test**

Remove `TestBuildAlluvialPhasesOrdersOneLiveStagePerReference`. Add

```go
func TestAlluvialCmd_ReportsOrderedReferenceBoundaries(t *testing.T)
```

by constructing `Flags` with the Task 1 reporter double. For references
`[]string{"v1", "v2", ""}`, assert `Begin` receives `6`, stages start in this exact order:
`Preparing`, `Loading v1`, `Loading v2`, `Loading HEAD`, `Rendering`,
`Writing output`, and the three loading stages use the established acquisition
work kind.

- [ ] **Step 2: Run the focused Alluvial boundary test to verify it fails**

Run: `./dev.sh -c 'go test ./cmd/codeviz -run "^TestAlluvialCmd_ReportsOrderedReferenceBoundaries$" -count=1'`

Expected: FAIL because `buildAlluvialPhases` still owns the stage definition.

- [ ] **Step 3: Inline the Alluvial lifecycle**

Create boundaries with `len(cfg.References)+3` only after
`mergeConfigAndValidate`. Keep current preparation pipeline calls, including
`alluvial.PrepareReferences`, inside the summary `Preparing` boundary. Loop
through `cfg.References` in configuration order; for each, start
`Loading ` plus `alluvial.SnapshotReference(reference)`, run the unchanged
`alluvial.AcquireReference` apply call, and end the boundary before the next
reference. Then explicitly bracket the current finalize/render group and
write group. Delete `buildAlluvialPhases`; do not expose snapshot or shared
history substeps as new stages. Once no command references the old executor,
delete `workflow.go` and `workflow_test.go`.

- [ ] **Step 4: Run Alluvial and full regression validation**

Run: `./dev.sh -c 'go test ./cmd/codeviz -run "TestAlluvialCmd_(ReportsOrderedReferenceBoundaries|Run_)" -count=1'`

Expected: PASS.

Run: `./dev.sh -c 'task ci'`

Expected: PASS with no failing tests or linters.

- [ ] **Step 5: Commit Alluvial orchestration cleanup**

```bash
git add cmd/codeviz/alluvial_cmd.go cmd/codeviz/alluvial_cmd_test.go cmd/codeviz/workflow.go cmd/codeviz/workflow_test.go
git commit -m "refactor: inline alluvial progress boundaries" -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

## Plan Self-Review

| Approved-spec requirement | Plan coverage |
|---|---|
| Remove phase descriptors/executor and keep boundaries explicit | Task 3 removes ordinary phase arrays; Task 4 removes the Alluvial array and then deletes the executor files. |
| Finalize outside `ApplyFunc*` despite pipeline short-circuiting | Task 1's `End` contract and pipeline-error test. |
| Preserve typed context/sink dependencies and reject stale sinks | Task 1 interfaces and stale-sink test. |
| Preserve error ownership/joining and guarantee reporter closure | Tasks 1 and 2, including terminal and close-failure tests. |
| Preserve curated ordinary phases, preset delegation, and Alluvial dynamics | Tasks 3 and 4 with exact counts/names and existing preset integration coverage. |
| Avoid generic pipeline/progress coupling and preserve reporters/diagnostics | Global Constraints, Task 1's narrow command helper, and Task 2's unchanged reporter construction/diagnostic setup. |

The plan contains four independently reviewable, testable tasks. It introduces
no product behavior outside the approved orchestration cleanup and leaves
renderer, provider, and generic pipeline code untouched.
