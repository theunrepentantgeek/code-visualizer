# Abstractions — `internal/progress`

## Reporter

**Purpose.**

- `Reporter` owns command progress presentation and stage accounting independently of the processing code. It selects plain, terminal, or quiet presentation without changing the producer contract ([progress.go](progress.go), [reporter.go](reporter.go)).

**Boundary and invariants.**

- `New` requires a writer. Call `Begin` with the planned stage count, start at most one stage at a time, and terminate it through `Complete`, `Fail`, or `Cancel`. `Finish` requires every planned stage to have terminated and no active stage ([reporter.go](reporter.go), [reporter_test.go](reporter_test.go)).
- A summary stage has `WorkNone` and cannot accept totals or progress. A live stage may report work. Failed/cancelled stages require a non-nil error and still count as terminated; successful `Finish` is accounting, not proof that the workflow succeeded ([reporter.go](reporter.go)).
- Rendering and writer errors are returned to the caller. `DiagnosticWriter` coordinates log output with progress, and `Close` flushes pending diagnostics and releases presentation resources; repeated closes are harmless ([reporter.go](reporter.go), [diagnostic_writer.go](diagnostic_writer.go)).

**Related operations.**

- `New`, `Begin`, `StartStage`, `Finish`, `DiagnosticWriter`, and `Close` form the lifecycle. `Stage` adds completion/failure/cancellation to the producer-facing `Sink` ([progress.go](progress.go)).

**Proper-use patterns.**

- CLI setup sends logging through `DiagnosticWriter` and joins command and close errors. Command boundaries own stage transitions and publish only the active `Sink` into pipeline state; processing stages do not own the reporter lifecycle ([../../cmd/codeviz/main.go](../../cmd/codeviz/main.go), [../../cmd/codeviz/progress_boundaries.go](../../cmd/codeviz/progress_boundaries.go), [../stages/progress.go](../stages/progress.go)).

**Anti-patterns.**

- Do not start overlapping stages, call `Finish` while work is active, discard reporting errors, or use `Finish` in place of checking the pipeline error ([reporter_test.go](reporter_test.go), [../../cmd/codeviz/progress_boundaries.go](../../cmd/codeviz/progress_boundaries.go)).
- Do not write diagnostics directly over a live terminal display or skip `Close` on failure ([diagnostic_writer.go](diagnostic_writer.go), [../../cmd/codeviz/main.go](../../cmd/codeviz/main.go)).

**Source locations.**

- [progress.go](progress.go) and [reporter.go](reporter.go) — interfaces and lifecycle enforcement.
- [../../cmd/codeviz/main.go](../../cmd/codeviz/main.go), [../../cmd/codeviz/progress_boundaries.go](../../cmd/codeviz/progress_boundaries.go), and [../stages/progress.go](../stages/progress.go) — setup, orchestration, and processing consumers.

## Sink

**Purpose.**

- `Sink` is the processing-facing contract for reporting an active stage's work unit, optional total, absolute progress, and status without controlling its lifecycle ([progress.go](progress.go)).

**Boundary and invariants.**

- `SetProgress` reports an absolute, non-negative, non-decreasing count, not an increment. Once a positive total is known, subsequent progress cannot exceed it. `SetTotal` accepts the same positive total again but rejects changing it ([reporter.go](reporter.go), [reporter_test.go](reporter_test.go)).
- `WorkObservations` counts file/metric observations, not distinct files; `WorkCommits` counts commit work. Producers must honor `WorkKind` so unrelated callbacks do not advance the active work scale ([../provider/run.go#L28](../provider/run.go#L28), [../stages/progress.go](../stages/progress.go)).
- Stage reporting is mutex-protected, but adapters must still serialize their own increment counters. `SetStatus` only emits in verbose mode, and all reporting errors must reach the workflow ([reporter.go](reporter.go), [../stages/progress.go](../stages/progress.go)).

**Related operations.**

- `WorkKind`, `SetTotal`, `SetProgress`, `SetStatus`; the orchestrator retains `Stage` to call `Complete`, `Fail`, or `Cancel` when processing returns ([progress.go](progress.go), [../../cmd/codeviz/progress_boundaries.go](../../cmd/codeviz/progress_boundaries.go)).

**Proper-use patterns.**

- Shared adapters translate scan/metric/history callbacks and latch reporting errors for their caller. Alluvial acquisition receives a sink per reference; the shared progress segment adds an offset when several passes share one total ([../stages/progress.go](../stages/progress.go), [../alluvial/pipeline.go#L237](../alluvial/pipeline.go#L237)).

**Anti-patterns.**

- Do not pass `1` on every processed item: progress is absolute. Do not count both observations and commits against the same work scale, or treat automatic completion-to-total as evidence that every item was processed ([reporter.go](reporter.go), [../stages/progress.go](../stages/progress.go)).

**Source locations.**

- [progress.go](progress.go), [reporter.go](reporter.go), and [reporter_test.go](reporter_test.go) — producer API and validation.
- [../stages/progress.go](../stages/progress.go), [../stages/providers.go](../stages/providers.go), and [../alluvial/pipeline.go](../alluvial/pipeline.go) — callback adaptation and independent acquisition workflows.
