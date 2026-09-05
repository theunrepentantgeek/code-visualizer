# Metrics Progress Completion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Always report successful metric-loading completion at the stated total without claiming completion after an incomplete or failed load.

**Architecture:** Extend the generic ticker shutdown with a completion signal so callers can wait until its goroutine has stopped. The metric-specific stop wrapper receives the stage outcome, then emits one terminal success line only when loading succeeded and the atomic counter equals the expected total.

**Tech Stack:** Go 1.26, `log/slog`, `sync/atomic`, Gomega

---

### Task 1: Specify Terminal Metric Progress

**Files:**
- Modify: `internal/stages/progress_internal_test.go`

- [ ] **Step 1: Write failing completion tests**

Add a test that builds metric progress with total `4`, sends four
`OnFileProcessed` callbacks, invokes `stop`, and asserts exactly one
`Loaded metrics` line with `loaded=4/4 percentage=100.0`.

Add a second test that reports all four callbacks but stops with a failed stage
outcome and asserts no `Loaded metrics` line appears. Cover successful zero-work
loading separately.

- [ ] **Step 2: Run tests and verify failure**

Run:

```bash
go test ./internal/stages -run 'TestBuildMetricProgress.*Completion' -count=1
```

Expected: the completed-progress test fails because no `Loaded metrics` line is
currently emitted.

### Task 2: Synchronize Shutdown and Log Completion

**Files:**
- Modify: `internal/stages/progress.go`

- [ ] **Step 1: Make ticker shutdown synchronous**

Add a `stopped` channel to `startProgressTicker`. Close it when the goroutine
exits, then make the returned stop function close `done` and wait on `stopped`.

- [ ] **Step 2: Wrap metric ticker shutdown**

In `startMetricTicker`, retain the ticker stop function and return a wrapper
that accepts the stage outcome and stops the ticker first. If loading succeeded
and `tracker.loaded.Load() == tracker.total`, log:

```text
Loaded metrics loaded=N/N percentage=100.0
```

Do not log completion when the values differ.

- [ ] **Step 3: Run focused tests**

Run:

```bash
go test ./internal/stages -run 'Test(Build|Log)MetricProgress' -count=1
```

Expected: PASS.

- [ ] **Step 4: Run provider progress tests**

Run:

```bash
go test ./internal/provider -run 'Test(FileProgressTotal|RunLoaders.*Progress)' -count=1
```

Expected: PASS, confirming loader callbacks still match their stated totals.

### Task 3: Verify the Complete Change

**Files:**
- Verify: `internal/stages/progress.go`
- Verify: `internal/stages/progress_internal_test.go`

- [ ] **Step 1: Exercise the sample command**

Run the donut-tree sample command and inspect captured logs. Confirm the metric
stage starts at zero and ends with `Loaded metrics` at the stated total before
rendering.

- [ ] **Step 2: Run repository CI**

Run `task ci` through the repository-required task subagent and require a zero
exit status with no failing tests or linters.

- [ ] **Step 3: Scan changed files and review security**

Scan all changed files for secrets, run CodeQL, and address any findings.

- [ ] **Step 4: Commit and report**

Commit the focused implementation, tests, and approved documentation through
the progress-reporting tool.
