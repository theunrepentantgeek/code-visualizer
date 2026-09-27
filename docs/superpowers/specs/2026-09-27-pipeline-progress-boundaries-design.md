# Pipeline Progress Boundaries Design

## Status

Ready for user review before implementation planning.

## Intent

This focused follow-up to PR #763 removes the `workflowPhase` data model and
`runWorkflow` command-level executor. Those abstractions duplicate the
imperative visualization pipelines they wrap: each command already defines the
ordering and purpose of its pipeline calls, then repeats that ordering as
callbacks in a separate phase slice.

The replacement makes progress lifecycle boundaries explicit at those existing
composition points. It preserves the established `internal/progress` reporter
and renderers, the separate context and typed active `progress.Sink` pipeline
dependencies, user-facing phase presentation, diagnostics coordination, and
reporter lifecycle validation. It does not change visualization output,
progress modes, or cancellation semantics.

Success means a reader can follow a command's processing sequence and see
exactly where each visible progress stage starts and ends, while every started
stage receives a terminal outcome even after pipeline processing records an
error.

## Scope

The change is limited to command orchestration and the small boundary support
needed to make that orchestration safe:

- remove `workflowPhase`, `runWorkflow`, and the phase-slice construction in
  visualization commands;
- inject explicit begin/start/end progress boundaries around existing pipeline
  groups for ordinary visualizations and Alluvial;
- make reporter closure application-owned so it is guaranteed after successful
  commands, command failures before the first phase, and progress failures;
- add focused tests for lifecycle ordering, errors, stage counts, dynamic
  Alluvial stages, and inactive/stale sinks.

It excludes new output formats, renderer changes, progress flags, pipeline
execution semantics, visualization algorithms, and a general pipeline event
system.

## Existing constraints

`pipeline.ApplyFunc*` deliberately does nothing once `State.Err()` is set.
That is correct for processing work but makes it unsuitable for progress
finalization: a terminal `Complete`, `Fail`, or `Cancel` operation expressed
as another apply call could be skipped precisely when it is required.

The reporter owns lifecycle validation and rendering. A `Stage` is also the
typed `progress.Sink` used by processing code for totals, absolute progress,
and verbose status. `context.Context` remains a distinct typed value in
`pipeline.State`; it continues to own cancellation checks in scanning,
providers, and Git traversal. Neither dependency moves into
`stages.CommonState`.

The reporter's diagnostic writer remains the only route for application
diagnostics after reporter construction. TTY pausing/redrawing and plain
append-only behavior are not moved into command orchestration.

## Options considered

### 1. Keep phase descriptors and simplify `runWorkflow`

This retains a `workflowPhase` slice but removes small pieces of branching
from the executor. It preserves a second representation of the same pipeline
and leaves the command's real sequence hidden in callbacks. It also makes
phase-specific dependencies, especially Alluvial's reference count and work
kind, indirect. Reject this option.

### 2. Make `pipeline.ApplyFunc*` progress-aware

The apply helpers could start and finish a progress stage around every
function, or gain progress-specific variants. This exposes internal
implementation steps as user-facing behavior, couples generic state
application to an unrelated UI concern, and cannot reliably finalize a stage
because `ApplyFunc*` short-circuits after `State.Err()`. Reject this option.

### 3. Explicit command-boundary operations around existing sequences

Commands continue to contain their pipeline calls in execution order. A small
boundary helper starts one curated reporter stage, injects its active sink,
and has an explicit end operation that examines the pipeline result without
using `ApplyFunc*`. The command calls these operations beside the processing
groups it already owns. This is the selected option: it preserves the pipeline
as the processing model and progress as a composition-boundary concern.

## Design

### Boundary ownership

`cmd/codeviz` gains a narrowly scoped progress-boundary helper in place of
the phase-plan executor. It manages only one currently active reporter stage;
it does not accept processing callbacks, build a slice of stages, or iterate
over a workflow.

The command composition follows this shape:

1. Begin the reporter once after configuration has merged and validated.
2. Start the named curated stage immediately before its existing pipeline
   calls, then store that returned stage as `progress.Sink` in the state.
3. Execute the existing calls in their current order.
4. Explicitly end the active stage immediately after the group, inspecting
   `state.Err()` and `context.Context` directly rather than via an apply
   helper.
5. Stop after the first processing failure; on success, finish the reporter
   after the final stage.

The helper can centralize repeated reporter error wrapping and terminal-outcome
selection, but callers must make begin/start/end visible at each pipeline
boundary. It must not recreate `workflowPhase`, accept `func(*pipeline.State)`,
or infer the command's stage plan.

The terminal end operation is unconditional for every successfully started
stage:

| Processing result | Terminal reporter call |
|---|---|
| no pipeline error and context is active | `Complete()` |
| error matching `context.Canceled` or `context.DeadlineExceeded` | `Cancel(err)` |
| any other pipeline error | `Fail(err)` |

If the pipeline itself did not record an error but the context is cancelled
between processing and finalization, cancellation wins and the stage is
cancelled. A reporter failure returned by start or end halts further pipeline
work. No terminal operation is expressed as `pipeline.ApplyFunc*`.

### Error and close ownership

The processing error is primary. A boundary joins, rather than replaces, an
error from `Fail`, `Cancel`, or `Complete` with that processing error using
`errors.Join`. This lets error classification still find cancellation and the
existing domain errors while retaining renderer/write failures.

On the all-success path, an error from `reporter.Finish()` is returned and
prevents successful command completion. `Finish()` is not called after a
failed or cancelled stage because its lifecycle contract requires all stages
to have succeeded.

`runApplication` owns `reporter.Close()` from the moment reporter construction
succeeds. It joins a close error with either the command error or a successful
result before classifying and logging the result. This covers validation and
configuration errors that occur before any command creates a progress
boundary. Command helpers do not close the reporter, so there is exactly one
close path. The reporter's existing idempotence remains a defensive property,
not a reason to discard a close error.

When a stage cannot start, it has no terminal outcome to render; the command
returns the start error and the application still closes the reporter. When
`Begin` fails, commands do not start processing and the application still
closes the reporter.

### Active and stale sinks

Immediately before a processing group, the boundary stores the returned
`progress.Stage` under the static `progress.Sink` key. Existing stages receive
that narrow typed dependency unchanged, and context remains separately stored.

Immediately after the terminal reporter operation, the boundary replaces the
state value with an inactive typed sink that returns an explicit lifecycle
error for updates. It must not leave the completed stage in state and must not
silently discard late updates. A component retaining a previous `Stage` also
continues to receive the reporter's existing "already finished" or "not
active" error. These two protections prevent a delayed callback from
rendering against a later phase or appearing to update a completed one.

The inactive sink is command-boundary support, not a new `CommonState` field
or a change to `pipeline.Set`/`pipeline.ApplyFunc*`. It exists only to keep
the typed dependency valid while making misuse observable.

### Ordinary visualization flow

Each ordinary visualization command explicitly presents four stages after
config validation:

| Index | Phase | Kind | Existing processing group |
|---|---|---|---|
| 1 | Preparing | summary | validate paths, export config, build filters, register selections, resolve metrics |
| 2 | Acquiring data | live | the visualization's existing `AcquireData` pipeline |
| 3 | Rendering | summary | the existing render pipeline |
| 4 | Writing output | summary | the existing output pipeline |

The command computes `stages.AcquisitionWork(common)` only after successful
metric resolution and uses it when it explicitly starts phase 2. It does not
mutate a future descriptor from phase 1. A failed preparation stage ends as
failed or cancelled and prevents phases 2–4 from starting. Preset rendering
still delegates to the selected visualization command, so it creates one
four-stage operation rather than a nested operation.

### Alluvial flow and dynamic references

Alluvial begins after effective configuration validation with exactly
`len(cfg.References) + 3` stages:

1. `Preparing` (summary): path/configuration/filter/metric setup and snapshot
   preparation that must precede reference acquisition;
2. one `Loading <reference>` live stage for every configured reference, in
   configuration order, with an empty reference displayed as `HEAD`;
3. `Rendering` (summary);
4. `Writing output` (summary).

The dynamic stage count is passed to `reporter.Begin` only after configuration
validation guarantees at least two unique references. Each reference boundary
starts with the work kind already established by preparation and injects its
own active sink before calling the existing reference-acquisition operation.
Each stage completes, fails, or cancels before the next reference begins.
Preparation remains a summary stage; its existing verbose status notifications
are valid summary-stage status updates and it does not publish totals or
absolute progress.

This retains the current user-facing reference ordering and does not expose
the internal snapshot and shared-history substeps as additional phases.

### Placement and diagnostics

Boundaries belong in the visualization commands, where configuration, state
construction, and user-facing command titles meet the existing pipelines:
the seven ordinary visualization commands and the Alluvial command. They do
not belong in `internal/pipeline`, `internal/stages`, renderers, providers, or
the preset dispatcher.

`runApplication` remains responsible for reporter creation, logger setup via
`DiagnosticWriter`, root cancellation context, reporter close, error
classification, and exit status. Plain and TTY reporters continue to render
the same lifecycle events, and diagnostic synchronization remains wholly in
`internal/progress`.

## Testing

Replace phase-slice executor tests with focused boundary tests using the
existing reporter doubles:

- normal ordinary lifecycle: begin, four explicit starts/terminal completions,
  finish, and application close in order;
- preparation, acquisition, rendering, and writing failures each terminally
  fail exactly their started stage and prevent later stages;
- context cancellation before a phase and during a phase terminally cancels
  the active stage and preserves `errors.Is(err, context.Canceled)`;
- terminal reporter errors and close errors are joined with the processing
  error, without losing the primary error for classification;
- begin/start/finish failures prevent inappropriate processing and still
  reach the sole close owner;
- the typed sink is replaced for every active stage and becomes inactive after
  end; updates through stale handles and the inactive state sink fail
  explicitly and emit no renderer events;
- ordinary commands pass four to `Begin`, while Alluvial passes exactly the
  configured reference count plus three and starts reference names in order,
  including `HEAD` normalization;
- a preset visualization still begins one operation and has no nested
  reporter lifecycle.

Retain the existing progress renderer, integration, cancellation, stream
separation, and lifecycle suites. Add no renderer snapshots unless an intended
user-visible event sequence changes; this refactor should preserve plain/TTY
output semantics.

## Acceptance criteria

- `workflowPhase`, `runWorkflow`, and phase-array construction are removed.
- Visualization commands show their existing processing order with explicit
  progress start/end boundaries at curated phase edges.
- No `pipeline.ApplyFunc*` API or behavior is coupled to progress, and no
  terminal stage operation depends on an apply call.
- Every started stage receives exactly one complete, fail, or cancel outcome,
  including when the pipeline has already recorded an error.
- Processing errors remain discoverable through `errors.Is`/`errors.As` after
  reporter and close failures are joined.
- Reporter closure is attempted exactly once by the application after
  successful construction, including pre-boundary command failures.
- Context cancellation, typed active sinks, diagnostics coordination, and
  plain/TTY renderer behavior remain intact.
- Ordinary commands have four curated stages; Alluvial has its reference
  count plus three, with ordered `Loading <reference>` stages.
- Completed stages cannot remain usable as the pipeline's active sink, and
  stale updates fail explicitly rather than affecting a later phase.
