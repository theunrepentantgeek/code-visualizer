# CLI Progress Output Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace routine CLI progress logs with deterministic plain output, PTerm live output, curated workflow phases, per-reference Alluvial progress, and prompt Ctrl-C cancellation.

**Architecture:** `internal/progress` owns mode resolution, lifecycle validation, rendering, and diagnostic coordination. Commands coordinate curated phase groups and place `context.Context` plus the current `progress.Sink` directly in `pipeline.State`; long-running stages declare those dependencies explicitly through `ApplyFuncXY`/`ApplyFuncXYZ`.

**Tech Stack:** Go 1.26.1, Kong, `log/slog`, PTerm v0.12.83, `golang.org/x/term` v0.46.0, Gomega, Goldie v2

**Spec:** `docs/superpowers/specs/2026-09-26-cli-progress-output-design.md`

## Global Constraints

- Run repository commands through `./dev.sh -c '<command>'`.
- Progress and diagnostics write only to stderr; existing stdout behavior stays unchanged.
- `--progress` accepts exactly `auto`, `tty`, and `plain`, with `auto` as the default.
- `--quiet` suppresses rendering, `--verbose` adds active-stage status, and `--debug` enables diagnostics.
- `--no-color`, non-empty `NO_COLOR`, `FORCE_COLOR=0`, and `TERM=dumb` disable color without otherwise disabling live output.
- Ordinary visualization workflows have several durable phases and exactly one live acquisition stage.
- Alluvial has one live stage per configured reference, preserving reference order.
- Context and the active progress sink are separate typed values in `pipeline.State`, never fields on `stages.CommonState`.
- Processing code publishes absolute counts; only renderers calculate and format percentages.
- Existing non-cancellation exit codes remain unchanged; cancellation exits 130.
- Retain processing errors as primary when joining reporter or cleanup errors.

## Review Focus

- `CI=false` and `CI=0` must not force plain mode, while a truthy `CI` value must.
- A diagnostic writer receiving partial or multiple lines must emit complete lines without corrupting an active TTY component.
- A cancellation arriving between curated phases must cancel the next phase rather than render it as successful.
- A live stage whose selected progress source has zero work must remain indeterminate and still complete cleanly.
- An Alluvial failure for one reference must name that reference, stop later references, and leave prior reference summaries durable.

---

### Task 1: Progress lifecycle and pipeline injection

**Files:**
- Create: `internal/progress/progress.go`
- Create: `internal/progress/reporter.go`
- Create: `internal/progress/reporter_test.go`
- Modify: `internal/pipeline/state.go`
- Modify: `internal/pipeline/apply_test.go`

**Interfaces:**
- Produces: `progress.Mode`, `progress.StageKind`, `progress.WorkKind`, `progress.Config`, `progress.Reporter`, `progress.Stage`, and `progress.Sink`.
- Produces: `pipeline.Set[T any](state *pipeline.State, value T)`.

- [ ] **Step 1: Write failing typed pipeline replacement tests**

Add tests proving `pipeline.Set[progress.Sink]` stores an interface under its static type, replaces the prior sink, and does not clear an existing `State.Err()`.

- [ ] **Step 2: Run the focused pipeline tests**

Run: `./dev.sh -c 'go test ./internal/pipeline -run "TestSet" -count=1'`

Expected: FAIL because `pipeline.Set` does not exist.

- [ ] **Step 3: Implement typed replacement**

Add:

```go
func Set[T any](state *State, value T)
```

It stores under `keyOf[T]()` even when `T` is an interface and does not alter the state's error.

- [ ] **Step 4: Write failing reporter lifecycle tests**

Define and test:

```go
type Mode string
const (ModeAuto Mode = "auto"; ModeTTY Mode = "tty"; ModePlain Mode = "plain")

type StageKind uint8
const (StageSummary StageKind = iota; StageLive)

type WorkKind uint8
const (WorkNone WorkKind = iota; WorkObservations; WorkCommits)

type Config struct {
    Mode       Mode
    Writer     io.Writer
    IsTerminal func(io.Writer) bool
    SupportsUnicode func() bool
    LookupEnv  func(string) (string, bool)
    NoColor    bool
    Quiet      bool
    Verbose    bool
    Now        func() time.Time
}

type Sink interface {
    WorkKind() WorkKind
    SetTotal(total int64) error
    SetProgress(current int64) error
    SetStatus(message string) error
}

type Stage interface {
    Sink
    Complete() error
    Fail(error) error
    Cancel(error) error
}

type Reporter interface {
    Begin(title string, stageCount int) error
    StartStage(name string, kind StageKind, work WorkKind) (Stage, error)
    Finish() error
    DiagnosticWriter() io.Writer
    Close() error
}

func New(config Config) (Reporter, error)
```

Tests assert the normal summary/live lifecycle; one-time positive total;
monotonic bounded progress; failure and cancellation; rejection of overlapping
stages, duplicate outcomes, finish with pending/active work, updates after an
outcome, and idempotent `Close`. Also assert a quiet reporter validates the same
lifecycle while writing no bytes, `Complete` renders a final total when needed,
and `Finish` emits the final operation summary exactly once.

- [ ] **Step 5: Run reporter lifecycle tests**

Run: `./dev.sh -c 'go test ./internal/progress -run "TestReporter|TestStage" -count=1'`

Expected: FAIL because the package and lifecycle are not implemented.

- [ ] **Step 6: Implement the renderer-independent state machine**

Keep state transitions and validation in `reporter.go`. Define an unexported
renderer interface receiving semantic begin/start/progress/status/outcome/
finish events; use a discard renderer for quiet mode. Return explicit errors
for every invalid transition and make `Close` idempotent without synthesizing
success.

- [ ] **Step 7: Run focused tests and commit**

Run: `./dev.sh -c 'go test ./internal/pipeline ./internal/progress -count=1'`

Expected: PASS.

Commit:

```bash
git add internal/pipeline internal/progress
git commit -m "feat: add progress lifecycle" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 2: Mode resolution and plain rendering

**Files:**
- Create: `internal/progress/mode.go`
- Create: `internal/progress/mode_test.go`
- Create: `internal/progress/plain.go`
- Create: `internal/progress/plain_test.go`
- Create: `internal/progress/testdata/plain-success.golden`
- Create: `internal/progress/testdata/plain-failure.golden`

**Interfaces:**
- Consumes: `progress.Config` and reporter events from Task 1.
- Produces: `ParseMode(value string) (Mode, error)` and plain renderer selection from `New`.

- [ ] **Step 1: Write failing mode tests**

Cover all valid modes, invalid values, terminal/non-terminal auto mode,
`TERM=dumb`, truthy `CI`, `CI=false`, `CI=0`, explicit plain on a terminal,
forced TTY on an unsupported writer, non-empty `NO_COLOR`, `FORCE_COLOR=0`, and
explicit `NoColor`.

- [ ] **Step 2: Run mode tests**

Run: `./dev.sh -c 'go test ./internal/progress -run "TestParseMode|TestResolveMode|TestResolveColor" -count=1'`

Expected: FAIL with missing mode resolution.

- [ ] **Step 3: Implement parsing and capability resolution**

Use `Config.IsTerminal` and `Config.LookupEnv` exclusively inside core logic.
Treat `CI` as true when set to a non-empty value other than case-insensitive
`false` or `0`. `auto` capability failure resolves to plain; forced TTY returns
an error.

- [ ] **Step 4: Write failing plain-renderer golden and throttling tests**

Use a fake clock. Assert stable indexed start/done/fail/cancel lines, elapsed
durations, no ANSI bytes, five-percentage-point updates, ten-second updates,
duplicate suppression, a final 100-percent update, line-safe error text, and a
zero-total live stage that emits no percentage. Assert status lines appear only
when `Config.Verbose` is true and the final operation summary is stable.

- [ ] **Step 5: Run plain-renderer tests**

Run: `./dev.sh -c 'go test ./internal/progress -run "TestPlain" -count=1'`

Expected: FAIL because plain rendering is missing.

- [ ] **Step 6: Implement the plain renderer**

Calculate percentages in the renderer, serialize writes, normalize embedded
newlines in status/error text, and apply the exact throttle rules from the
spec. Use Goldie only for complete output examples; keep threshold behavior in
direct assertions.

- [ ] **Step 7: Run focused tests and commit**

Run: `./dev.sh -c 'go test ./internal/progress -count=1'`

Expected: PASS.

Commit:

```bash
git add internal/progress
git commit -m "feat: add plain progress output" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 3: PTerm rendering and coordinated diagnostics

**Files:**
- Modify: `go.mod`
- Modify: `go.sum`
- Create: `internal/progress/tty.go`
- Create: `internal/progress/tty_test.go`
- Create: `internal/progress/diagnostic_writer.go`
- Create: `internal/progress/diagnostic_writer_test.go`

**Interfaces:**
- Consumes: reporter events and resolved color/mode settings from Tasks 1-2.
- Produces: PTerm-backed live rendering and `Reporter.DiagnosticWriter()`.

- [ ] **Step 1: Add dependency declarations**

Run: `./dev.sh -c 'go get github.com/pterm/pterm@v0.12.83 golang.org/x/term@v0.46.0'`

Expected: `go.mod` and `go.sum` record compatible direct dependencies without
changing the module's Go version.

- [ ] **Step 2: Write failing semantic TTY tests**

Inject fake spinner/progress components behind an unexported factory. Assert
summary stages never animate, live stages start as spinners, `SetTotal`
transitions once to a progress bar, all writes use the configured writer,
no-color mode removes styling, and success/failure/cancellation/early close
stop the active component. Assert `SupportsUnicode=false` selects ASCII-safe
symbols. Make the fake factory fail initialization and assert `auto` emits one
diagnostic then uses plain rendering, while explicit `tty` returns the error.

- [ ] **Step 3: Run TTY tests**

Run: `./dev.sh -c 'go test ./internal/progress -run "TestTTY" -count=1'`

Expected: FAIL because the TTY renderer is absent.

- [ ] **Step 4: Implement the PTerm renderer**

Wrap configured PTerm printers rather than global defaults. Render durable
summaries after each stage, keep at most one live component, and avoid golden
assertions against PTerm escape-sequence details.

- [ ] **Step 5: Write failing diagnostic-writer tests**

Assert one complete line pauses/writes/redraws, partial writes buffer until a
newline, one write containing multiple lines emits each exactly once, plain
mode remains append-only, and `Close` flushes a final partial line safely.

- [ ] **Step 6: Implement the coordinated diagnostic writer**

Serialize renderer events and diagnostics with the reporter mutex. The writer
must coordinate pause/redraw through renderer methods rather than writing
around the reporter.

- [ ] **Step 7: Run focused tests and commit**

Run: `./dev.sh -c 'go test ./internal/progress -count=1'`

Expected: PASS.

Commit:

```bash
git add go.mod go.sum internal/progress
git commit -m "feat: add terminal progress rendering" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 4: Context-aware long-running operations

**Files:**
- Modify: `internal/scan/scanner.go`
- Modify: `internal/scan/fs_walker.go`
- Modify: `internal/scan/walker.go`
- Modify: `internal/scan/scanner_test.go`
- Modify: `internal/provider/loader.go`
- Modify: `internal/provider/run.go`
- Modify: `internal/provider/run_test.go`
- Modify: `internal/provider/filesystem/metrics.go`
- Modify: `internal/provider/filesystem/register.go`
- Modify: `internal/provider/golang/file_loader.go`
- Modify: `internal/provider/golang/register.go`
- Modify: `internal/provider/classification/provider.go`
- Modify: `internal/provider/git/loader.go`
- Modify: `internal/provider/git/register.go`
- Modify: `internal/provider/git/authorship_loader.go`
- Modify: `internal/provider/git/commit.go`
- Modify: `internal/provider/git/author_history.go`
- Modify: `internal/provider/git/history_range.go`
- Test: matching `*_test.go` files under `internal/provider/git`

**Interfaces:**
- Produces: `scan.ScanTree(ctx context.Context, ...)` and `scan.Scan(ctx context.Context, ...)`.
- Produces: `provider.LoadFunc func(context.Context, *model.Directory, []metric.Name) error`.
- Produces: `provider.RunLoaders(ctx context.Context, root *model.Directory, requested []metric.Name, progress MetricProgress) error`.
- Produces: context-first signatures for commit counting/walking, file Git metrics, and authorship history functions.

- [ ] **Step 1: Write failing scan cancellation tests**

Use an already-cancelled context and a filesystem with multiple entries.
Assert `ScanTree` and `Scan` return errors matching `context.Canceled` before
reporting successful completion.

- [ ] **Step 2: Make scanning context-aware**

Store context on both walker types and check `ctx.Err()` before directory reads,
entry processing, recursive descent, and expensive file probing. Preserve
existing permission and disappearing-file behavior.

- [ ] **Step 3: Write failing provider cancellation tests**

Assert an already-cancelled context starts no loaders; cancellation during a
parallel loader level cancels sibling work through `errgroup.WithContext`; and
filesystem, Go, classification, and Git loaders stop at file/work-unit
boundaries with an error matching `context.Canceled`.

- [ ] **Step 4: Thread context through loader APIs**

Change `LoadFunc` and all registered loaders to accept context. Use
`errgroup.WithContext`, check context before scheduling and processing each
file, and preserve original loader errors when they win the race with
cancellation.

- [ ] **Step 5: Write failing Git cancellation tests**

Cover cancellation during commit counting, tracked-history walking, prewarming,
and authorship aggregation. Assert repository locks/iterators are released and
returned errors match `context.Canceled`.

- [ ] **Step 6: Thread context through Git history APIs**

Add context as the first argument to the internal commit-total, bulk-history,
prewarm, file-metric, and authorship functions and update all callers. Check it
before each iterator step and expensive per-commit/per-file operation.

- [ ] **Step 7: Run focused tests and commit**

Run: `./dev.sh -c 'go test ./internal/scan ./internal/provider/... -count=1'`

Expected: PASS.

Commit:

```bash
git add internal/scan internal/provider
git commit -m "feat: cancel long-running data acquisition" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 5: Stage progress adapters

**Files:**
- Replace: `internal/stages/progress.go`
- Replace: `internal/stages/progress_test.go`
- Replace: `internal/stages/progress_internal_test.go`
- Modify: `internal/stages/scan.go`
- Modify: `internal/stages/scan_test.go`
- Modify: `internal/stages/providers.go`
- Modify: `internal/stages/providers_test.go`
- Modify: `internal/stages/git_history.go`
- Modify: `internal/stages/git_history_test.go`
- Modify: `internal/stages/author_history.go`
- Modify: `internal/stages/common.go`

**Interfaces:**
- Consumes: `context.Context` and `progress.Sink` from pipeline state.
- Produces: `ScanFilesystem(*CommonState, context.Context, progress.Sink) error`.
- Produces: `RunProviders(*CommonState, context.Context, progress.Sink) error`.
- Produces: context/sink-aware history and authorship stages.
- Produces: `AcquisitionWork(*CommonState) progress.WorkKind`.

- [ ] **Step 1: Write failing adapter tests**

Use a recording sink. Assert scan publishes verbose discovered-file status but
does not claim a total; observation-oriented provider work sets the exact
`FileProgressTotal` and reports absolute observations; commit-oriented history
sets commit total and reports absolute commits; non-selected sources do not
replace a total; and cancellation propagates unchanged.

- [ ] **Step 2: Run stage progress tests**

Run: `./dev.sh -c 'go test ./internal/stages -run "Test.*Progress|TestRunProviders|TestLoadGitHistory|TestScanFilesystem" -count=1'`

Expected: FAIL against the old ticker/slog implementation.

- [ ] **Step 3: Replace ticker logging with sink adapters**

Delete timer goroutines and percentage formatting. `AcquisitionWork` selects
`WorkCommits` for explicit commit-expression/authorship history and
`WorkObservations` otherwise. Stage functions compare `sink.WorkKind()` before
setting totals or progress; non-selected work may publish verbose status only.
Keep warnings as `slog` diagnostics and remove routine info/debug progress
lines.

- [ ] **Step 4: Update stage pipeline call signatures**

Remove progress fields from `stages.Flags`. Keep progress and context out of
`CommonState`; pipeline callers obtain them as separate typed inputs.

- [ ] **Step 5: Run focused tests and commit**

Run: `./dev.sh -c 'go test ./internal/stages ./internal/scan ./internal/provider/... -count=1'`

Expected: PASS with no ticker-based assertions or goroutines.

Commit:

```bash
git add internal/stages internal/scan internal/provider
git commit -m "feat: report stage acquisition progress" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 6: Command workflow coordinator and ordinary visualizations

**Files:**
- Create: `cmd/codeviz/workflow.go`
- Create: `cmd/codeviz/workflow_test.go`
- Modify: `cmd/codeviz/main.go`
- Modify: `cmd/codeviz/main_test.go`
- Modify: `cmd/codeviz/treemap_cmd.go`
- Modify: `cmd/codeviz/radialtree_cmd.go`
- Modify: `cmd/codeviz/bubbletree_cmd.go`
- Modify: `cmd/codeviz/donuttree_cmd.go`
- Modify: `cmd/codeviz/spiral_cmd.go`
- Modify: `cmd/codeviz/scatter_cmd.go`
- Modify: `internal/treemap/pipeline.go`
- Modify: `internal/radialtree/pipeline.go`
- Modify: `internal/bubbletree/pipeline.go`
- Modify: `internal/donuttree/pipeline.go`
- Modify: `internal/spiral/pipeline.go`
- Modify: `internal/scatter/pipeline.go`
- Modify: affected pipeline and command tests

**Interfaces:**
- Consumes: `progress.Reporter`, `progress.Sink`, context-aware stages, and `pipeline.Set`.
- Produces: `workflowPhase` and `runWorkflow(context.Context, progress.Reporter, *pipeline.State, string, []workflowPhase) error`.
- Produces: `Flags.Context context.Context` and `Flags.Reporter progress.Reporter`.

- [ ] **Step 1: Write failing workflow coordinator tests**

Assert ordered summary/live stages, exactly one ordinary live phase, typed sink
replacement before each phase, finish only after all phases, cancellation
between phases, failure versus cancellation outcomes, and `errors.Join` when a
processing error and reporter failure coexist.

- [ ] **Step 2: Implement the coordinator**

Define:

```go
type workflowPhase struct {
    Name string
    Kind progress.StageKind
    Work progress.WorkKind
    Run  func(*pipeline.State)
}

func runWorkflow(
    ctx context.Context,
    reporter progress.Reporter,
    state *pipeline.State,
    title string,
    phases []workflowPhase,
) error
```

Set `context.Context` once and replace `progress.Sink` for every phase through
`pipeline.Set`. Check `ctx.Err()` before running each phase. Preserve the
pipeline error as primary when outcome, finish, or close also fails.

- [ ] **Step 3: Write failing CLI flag and exit tests**

Assert Kong accepts `--progress=auto|tty|plain`, rejects another value as usage
error, parses `--no-color`, retains verbosity XOR behavior, and classifies
`context.Canceled` as exit 130 while existing codes remain 1-6.

- [ ] **Step 4: Wire application composition**

Create the signal context with `signal.NotifyContext` in `main`; construct the
reporter against stderr after argument parsing; configure `slog` with the
reporter's diagnostic writer; and pass context/reporter through `Flags`.
Default logging level is warning, while `--debug` enables debug. Keep the early
bootstrap logger for parser/setup errors.

- [ ] **Step 5: Split ordinary pipelines into curated groups**

For each ordinary visualization, expose command-callable groups for preparing,
acquiring, rendering, and writing while preserving existing golden-test entry
points. Build a phase plan with summary preparation/render/write phases and one
live acquisition phase whose work kind comes from `stages.AcquisitionWork`.
Update `ApplyFunc*` arity only where context or sink is a declared dependency.

- [ ] **Step 6: Run ordinary workflow tests**

Run: `./dev.sh -c 'go test ./cmd/codeviz ./internal/{treemap,radialtree,bubbletree,donuttree,spiral,scatter} ./internal/goldentest -count=1'`

Expected: PASS with unchanged image goldens.

- [ ] **Step 7: Commit**

```bash
git add cmd/codeviz internal/treemap internal/radialtree internal/bubbletree internal/donuttree internal/spiral internal/scatter internal/goldentest
git commit -m "feat: coordinate visualization progress" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 7: Alluvial per-reference progress

**Files:**
- Modify: `cmd/codeviz/alluvial_cmd.go`
- Modify: `cmd/codeviz/alluvial_cmd_test.go`
- Modify: `internal/alluvial/pipeline.go`
- Create: `internal/alluvial/progress_test.go`
- Modify: `internal/goldentest/viz_golden_test.go`

**Interfaces:**
- Consumes: the workflow coordinator and stage sinks from Task 6.
- Produces: Alluvial preparation, `AcquireReference`, final data assembly, rendering, and writing phase functions.

- [ ] **Step 1: Write failing per-reference tests**

For references `v1`, `v2`, and `HEAD`, assert one ordered live
`Loading <reference>` stage per reference; each stage receives independent
progress starting from zero; summary preparation happens before reference
loading; and rendering/writing happen afterward. Inject a failure for `v2` and
assert `v3` never starts, the error names `v2`, and `v1` remains completed.

- [ ] **Step 2: Refactor snapshot acquisition around reference phases**

Retain the current two-pass optimization: prepare/scan all snapshot trees and
share history paths in a summary phase, then expose one finish/acquire function
per reference. Each live reference stage receives the active sink directly
from pipeline state and chooses commit or observation progress without
changing totals mid-stage.

- [ ] **Step 3: Build the dynamic Alluvial phase plan**

After config validation, create stage count `3 + len(references)` for prepare,
ordered references, render, and write. Normalize empty display references to
`HEAD`; preserve configured order and avoid starting a second reporter when
called through `render` presets.

- [ ] **Step 4: Run Alluvial and golden tests**

Run: `./dev.sh -c 'go test ./cmd/codeviz ./internal/alluvial ./internal/goldentest -run "Alluvial|alluvial" -count=1'`

Expected: PASS with unchanged visualization goldens.

- [ ] **Step 5: Commit**

```bash
git add cmd/codeviz/alluvial_cmd.go cmd/codeviz/alluvial_cmd_test.go internal/alluvial internal/goldentest/viz_golden_test.go
git commit -m "feat: report alluvial reference progress" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```

### Task 8: End-to-end behavior and documentation

**Files:**
- Create: `cmd/codeviz/progress_integration_test.go`
- Create: `cmd/codeviz/progress_signal_unix_test.go`
- Modify: `cmd/codeviz/main.go`
- Modify: `docs/content/docs/usage.md`
- Modify: `internal/config/config.go`
- Modify: `internal/scan/scanner.go`
- Modify: `internal/treemap/stages.go`
- Modify: `internal/radialtree/stages.go`
- Modify: `internal/bubbletree/stages.go`
- Modify: `internal/donuttree/stages.go`
- Modify: `internal/spiral/stages.go`
- Modify: `internal/scatter/stages.go`

**Interfaces:**
- Consumes: completed CLI progress and cancellation behavior.
- Produces: documented, end-to-end verified user experience.

- [ ] **Step 1: Write failing stream and failure integration tests**

Run a representative command through the testable application entry point.
Assert explicit plain mode and redirected auto mode write indexed ANSI-free
progress only to stderr; functional output/help remains on stdout; quiet writes
no progress; no-color output has no styling; forced unsupported TTY and invalid
mode are usage failures; and a processing failure keeps its existing exit code
and produces a durable failed stage.

- [ ] **Step 2: Write failing cancellation integration tests**

Use a cancellable context for deterministic tests and a Unix subprocess test
for SIGINT. Assert an in-flight data stage renders `cancelled`, later phases do
not start, cleanup returns, and the exit status is 130.

- [ ] **Step 3: Close integration gaps**

Refactor only the entry-point seams needed for deterministic injection of args,
stderr, terminal detection, environment lookup, and context. Remove remaining
routine progress `slog.Info`/`slog.Debug` calls while retaining actionable
warnings and debug diagnostics.

- [ ] **Step 4: Update usage documentation**

Document `--progress`, `--no-color`, automatic terminal/CI behavior,
quiet/verbose/debug interaction, stderr/stdout separation, Alluvial
per-reference progress, and exit code 130.

- [ ] **Step 5: Run focused and full tests**

Run: `./dev.sh -c 'go test ./cmd/codeviz ./internal/progress ./internal/stages ./internal/alluvial -count=1'`

Expected: PASS.

Run: `./dev.sh -c 'task test:race'`

Expected: PASS with no reporter, loader, or diagnostic-writer races.

- [ ] **Step 6: Verify supported release targets compile**

Run: `./dev.sh -c 'task build:release'`

Expected: PASS for Linux, macOS, and Windows amd64/arm64 targets.

- [ ] **Step 7: Run repository CI through an Explore-equivalent task runner**

Delegate: `./dev.sh -c 'task ci'`

Expected concise result: exit status 0, zero failing tests/linters, and a
one-line no-issues note. Do not suppress the configured verbose lint output in
the delegated command.

- [ ] **Step 8: Commit**

```bash
git add cmd/codeviz internal docs/content/docs/usage.md go.mod go.sum
git commit -m "feat: finish CLI progress integration" \
  -m "Co-authored-by: Copilot <223556219+Copilot@users.noreply.github.com>"
```
