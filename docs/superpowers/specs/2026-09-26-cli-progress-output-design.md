# CLI Progress Output Design

## Status

Approved for implementation planning.

## Intent

Replace routine progress logging with a dedicated reporting subsystem that
makes long-running CodeViz commands understandable in terminals, redirected
output, and CI.

The feature is for people running visualization commands interactively or in
automation. It succeeds when users can identify the current operation, see
durable phase outcomes, follow the one meaningful unit of live progress, and
understand failures without terminal output corruption.

## Goals

- Show a concise, stable workflow rather than internal pipeline implementation
  details.
- Use polished live progress on capable terminals.
- Emit stable, append-only text when stderr is redirected or running in CI.
- Preserve stdout for functional and machine-consumable output.
- Keep terminal rendering out of processing code.
- Make progress, diagnostics, failures, and cancellation coexist safely.
- Support deterministic tests through injected writers, clocks, and terminal
  detection.
- Cancel long-running work promptly on Ctrl-C.

## Non-goals

- An interactive or full-screen terminal UI.
- User input, menus, or keyboard navigation.
- Concurrent or nested progress bars.
- Mirroring every internal pipeline stage in user-facing output.
- A general logging framework replacement.
- Persisting progress or color preferences in CodeViz configuration files.
- JSON progress events in the initial implementation.

## User-facing behavior

### Output modes

Add global flags:

```text
--progress=auto|tty|plain
--no-color
```

`auto` is the default.

| Mode | Behavior |
|---|---|
| `auto` | Use rich output when stderr is a capable interactive terminal; otherwise use plain output. |
| `tty` | Force rich output and return a usage/startup error when stderr cannot support it. |
| `plain` | Emit append-only text without animation, cursor control, or ANSI styling. |

An invalid mode is a usage error before processing starts. Progress and
diagnostics go to stderr; stdout behavior remains unchanged.

`auto` selects plain output when stderr is not a terminal, `TERM=dumb`, a
recognized CI environment is present, or terminal capability initialization
fails. An automatic capability failure may emit one concise diagnostic before
continuing in plain mode. Explicit `tty` does not fall back silently.

Color is independent of progress mode. `--no-color`, `NO_COLOR`,
`FORCE_COLOR=0`, and `TERM=dumb` disable color. A color-disabled capable
terminal may still use live cursor updates.

### Verbosity

- `--quiet` suppresses progress by installing a no-op reporter. Warnings and
  errors remain visible.
- Default mode shows stable phase outcomes and the meaningful live progress
  stage.
- `--verbose` adds status details within the active stage without changing the
  stage plan.
- `--debug` enables diagnostic logging.

`--verbose` and `--debug` remain mutually exclusive through the existing Kong
flag group.

### Workflow presentation

Visualization commands expose curated phases such as:

1. Preparing inputs
2. Acquiring data
3. Rendering visualization
4. Writing output

Names may be adjusted to fit the operation, but they must remain user-facing
and stable. Internal pipeline functions do not each become a visible phase.

For ordinary visualizations, only the data-acquisition phase uses live
progress. It reports the most meaningful determinate unit available:

- file or metric observations for file-oriented work;
- commits for history-oriented work;
- an indeterminate spinner until a meaningful total is discovered, or when no
  meaningful total exists.

The other phases still produce durable summaries. In plain mode they emit
start and outcome events. In TTY mode they do not animate; they become visible
as durable completion or failure lines.

Alluvial commands replace the single acquisition phase with one live
`Loading <reference>` stage per configured reference, in reference order. Each
reference has its own progress and outcome. `HEAD` is used as the display name
for an empty reference, matching existing normalization.

TTY output includes a heading, durable completed-phase lines, at most one
active spinner or progress bar, elapsed completion times, clear outcome
indicators, and a final operation summary. Output must remain understandable
without color and should use ASCII-safe symbols when Unicode is unavailable.

Plain output includes stable stage indexes and totals:

```text
[1/4] Preparing inputs: started
[1/4] Preparing inputs: done (0.2s)
[2/4] Acquiring data: started
[2/4] Acquiring data: 100/500 (20%)
[2/4] Acquiring data: done (8.7s)
```

Plain progress updates are emitted when the displayed percentage advances by
at least five points, ten seconds have elapsed since the previous update, or
the stage reaches 100 percent. Duplicate displayed values are suppressed.

## Architecture

### Progress package

Add `internal/progress`. It owns:

- mode parsing and resolution;
- terminal and CI capability detection;
- the reporter state machine and stage handles;
- PTerm-backed TTY rendering;
- plain-text rendering;
- the quiet no-op implementation;
- timing and plain-output throttling;
- coordinated diagnostic writes.

The package does not own visualization-specific stage definitions or business
operations.

Construction accepts an `io.Writer`, clock, and terminal detector. Mode is
resolved once. PTerm is used only by the TTY renderer and must be configured
against the injected writer rather than process-global printers.

### Pipeline dependencies

Progress and cancellation are explicit typed dependencies stored directly in
`pipeline.State`; they are not added to `stages.CommonState`.

Stages that require progress or cancellation change their signatures to accept
the narrow dependency and move from `pipeline.ApplyFuncX` to
`ApplyFuncXY`/`ApplyFuncXYZ` as needed. Stages that need neither remain
unchanged. This keeps dependencies visible in function signatures and
preserves the type-keyed pipeline architecture instead of allowing
`CommonState` to accumulate cross-cutting collaborators.

The stored values are distinct:

- `context.Context` for cancellation checks;
- a narrow active-stage progress sink for status, total, and absolute progress
  updates.

Add a generic typed `pipeline.Set` operation so the command-level coordinator
can replace the active stage sink between curated phase groups. `Set` is not
progress-aware and does not alter `ApplyFunc*` error short-circuiting. The
pipeline continues to store the concrete active sink directly rather than a
cross-cutting runtime container or a new `CommonState` field.

No generic `pipeline.ApplyFunc*` primitive gains progress-specific behavior.

### Workflow coordinator

Each visualization command builds a stage plan after configuration is merged
and validated. This allows plain output to know the final stage count before
execution, including Alluvial's configured references.

A small command/pipeline-boundary coordinator owns:

- beginning the operation;
- activating each planned phase;
- placing its sink in pipeline state;
- completing, failing, or cancelling the phase;
- finishing the operation;
- closing terminal resources.

The coordinator wraps curated groups of existing pipeline calls. It does not
replace the pipeline or move rendering/business logic into command types.
Preset rendering continues to delegate to the selected visualization command
and therefore creates only one progress operation.

### Reporter lifecycle

A planned stage is either `summary` or `live`. At most one stage is active.

```text
pending -> running -> succeeded
                   -> failed
                   -> cancelled
```

A live stage begins indeterminate. It may set a positive total once the
underlying operation discovers one, allowing scanning or repository setup to
use a spinner before transitioning to determinate file/commit progress. Once
set, the total cannot change.

The active sink accepts:

- a positive total, set at most once;
- an absolute completed-work value;
- a human-readable status message when verbose detail is enabled.

Progress must be non-negative, monotonic, and no greater than a known total.
Processing code never calculates percentages or formats progress strings.

The reporter explicitly returns errors for:

- beginning twice;
- starting a stage while another is active;
- updating without an active matching handle;
- setting an invalid or replacement total;
- decreasing or over-total progress;
- completing, failing, or cancelling more than once;
- finishing with pending or active work.

`Close` is idempotent, stops animation, and restores terminal state. Closing an
unfinished operation never renders success.

### Diagnostics

Routine `slog` progress messages are removed. Actionable warnings and debug
diagnostics remain.

The reporter provides or backs a coordinated `io.Writer` used by the
application's `slog` handler. In TTY mode, a complete diagnostic line pauses or
clears the active component, writes the diagnostic, and redraws live progress.
In plain mode it writes a complete append-only line. Direct writes to the
progress stream while animation is active are not allowed.

Bootstrap errors emitted before reporter construction continue to use the
early logger.

### Cancellation and errors

`main` creates a root context with `signal.NotifyContext` for Ctrl-C and passes
it as a distinct pipeline dependency. Long-running scan, provider, Git
history, and Alluvial reference loops check the context at practical work-unit
boundaries and return a wrapped `context.Canceled` promptly.

When processing returns:

- `context.Canceled`, the active stage renders cancellation and the process
  exits with status 130;
- another error, the active stage renders failure and existing error
  classification determines the exit status;
- nil, the stage completes normally.

Reporter lifecycle and write errors are returned. When processing and
reporting or cleanup both fail, preserve the processing error as primary and
retain the additional error with `errors.Join`. The overall operation is
reported successful only after all stages succeed and reporter finalization
succeeds.

## Testing

### Unit tests

Mode parsing and resolution:

- all valid and invalid values;
- terminal and non-terminal `auto`;
- `TERM=dumb` and recognized CI;
- forced plain and forced unsupported TTY;
- all no-color inputs.

Lifecycle:

- summary and live success paths;
- indeterminate progress promoted once to determinate progress;
- failure and cancellation;
- overlapping active stages;
- invalid, decreasing, duplicate, and over-total updates;
- duplicate terminal outcomes;
- finish with unfinished work;
- idempotent close.

Plain rendering, using a buffer and fake clock:

- stable golden output;
- stage numbering and elapsed durations;
- five-percent and ten-second throttling;
- duplicate suppression and final 100-percent update;
- concise, line-safe errors;
- no ANSI or cursor-control bytes.

TTY rendering, tested semantically rather than against PTerm internals:

- summary stages do not animate;
- indeterminate live stages select a spinner;
- setting a total transitions to a progress bar;
- color-disabled output has no color escapes;
- output is isolated to the configured writer;
- success, failure, cancellation, and early close clean up terminal state.

### Integration tests

Run representative commands with:

- explicit plain mode and captured streams;
- default auto mode with redirected stderr;
- `NO_COLOR=1` and `--no-color`;
- a forced processing failure;
- cancellation during file and Git history work;
- an Alluvial command with multiple references.

Verify stdout remains functional, stderr is append-only and ANSI-free when
plain, existing non-cancellation exit codes remain unchanged, Ctrl-C exits 130,
Alluvial reports each reference separately and in order, diagnostics do not
corrupt live output, and no animation goroutine remains after exit.

## Documentation

Update CLI help and usage documentation for:

- `--progress`;
- `--no-color`;
- the relationship among quiet, verbose, debug, and progress;
- stderr/stdout separation;
- automatic behavior in terminals, redirection, and CI.

## Acceptance criteria

- `--progress` accepts exactly `auto`, `tty`, and `plain`; default is `auto`.
- Rich output is used by default only on capable interactive stderr.
- Redirected and CI output is stable, plain, append-only, and ANSI-free.
- `--quiet`, `--verbose`, `--debug`, and `--no-color` behave as specified.
- Ordinary visualizations have several durable phase summaries and only one
  live acquisition stage.
- Alluvial has one ordered live stage per configured reference.
- Processing code publishes absolute progress and never formats percentages.
- Progress and context are explicit pipeline-state dependencies, not fields on
  `CommonState`.
- Reporter misuse and write failures are surfaced.
- Diagnostics cannot corrupt live output.
- Ctrl-C cancels long-running work, renders cancellation, cleans up terminal
  state, and exits 130.
- Progress and diagnostics use stderr; stdout and existing non-cancellation
  exit codes are unchanged.
- Deterministic unit and integration tests cover renderers, lifecycle,
  cancellation, Alluvial references, and stream separation.
