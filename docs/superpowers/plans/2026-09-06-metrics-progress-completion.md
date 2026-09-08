# Metrics Progress Completion Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make metric-loading progress finish at the stated total and emit `Loaded metrics` after successful work without passing operation state into a cleanup callback.

**Architecture:** `RunProviders` owns the operation result, stops the existing progress ticker, and emits terminal success only after loading succeeds. `BuildMetricProgress` keeps returning `(provider.MetricProgress, func())`, suppresses zero-work progress, and uses synchronous ticker shutdown so no periodic line can race after completion.

**Tech Stack:** Go 1.26, `log/slog`, `sync/atomic`, Gomega

---

## File Structure

- Modify `internal/stages/progress.go` to suppress zero-total progress, make generic ticker shutdown synchronous, and format terminal metric progress.
- Modify `internal/stages/progress_internal_test.go` to specify zero-total suppression.
- Modify `internal/stages/progress_test.go` to use positive totals in tests that expect a reporter.
- Modify `internal/stages/providers.go` to own loading success, ticker shutdown, and completion logging.
- Create `internal/stages/providers_test.go` to cover successful and failed provider paths through the registry.

### Task 1: Suppress Progress When There Is No File-Metric Work

**Files:**
- Modify: `internal/stages/progress_internal_test.go:38-54`
- Modify: `internal/stages/progress_test.go:16-117`
- Modify: `internal/stages/progress.go:28-40`

- [ ] **Step 1: Write the failing zero-total test**

Add this test to `internal/stages/progress_internal_test.go`:

```go
//nolint:paralleltest // mutates global slog default logger
func TestBuildMetricProgressSuppressesZeroTotal(t *testing.T) {
	g := NewGomegaWithT(t)

	var buf bytes.Buffer

	oldDefault := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{})))
	defer slog.SetDefault(oldDefault)

	progress, stop := BuildMetricProgress(&Flags{}, 0)
	stop()

	g.Expect(progress).To(BeNil())
	g.Expect(buf.String()).To(BeEmpty())
}
```

- [ ] **Step 2: Run the focused test to verify it fails**

Run:

```bash
go test ./internal/stages -run TestBuildMetricProgressSuppressesZeroTotal -count=1
```

Expected: FAIL because `BuildMetricProgress` currently returns a tracker and logs `Loading metrics.` for a zero total.

- [ ] **Step 3: Suppress zero-total progress**

Change `BuildMetricProgress` in `internal/stages/progress.go` to:

```go
func BuildMetricProgress(flags *Flags, total int64) (provider.MetricProgress, func()) {
	if flags.Quiet || total <= 0 {
		return nil, func() {}
	}

	tracker := &metricProgressTracker{total: total}
	stop := startMetricTicker(tracker)

	return tracker, stop
}
```

- [ ] **Step 4: Update reporter tests to use real work totals**

In `internal/stages/progress_test.go`, change every non-quiet test that expects a non-nil metric reporter from total `0` to a positive total. Use `1` for the gating, stop, and unknown-metric tests, and use `2` for `TestBuildMetricProgress_OnMetricStarted_RecordsMetric`:

```go
prog, stop := stages.BuildMetricProgress(flags, 1)
```

```go
prog, stop := stages.BuildMetricProgress(flags, 2)
```

Leave the quiet-mode tests unchanged because quiet mode must return `nil` regardless of the total.

- [ ] **Step 5: Run the focused progress tests**

Run:

```bash
go test ./internal/stages -run 'Test(BuildMetricProgress|LogMetricProgress)' -count=1
```

Expected: PASS.

- [ ] **Step 6: Commit zero-work suppression**

```bash
git add internal/stages/progress.go internal/stages/progress_internal_test.go internal/stages/progress_test.go
git commit -m "Suppress empty metric progress"
```

### Task 2: Report Provider Completion From the Owning Stage

**Files:**
- Create: `internal/stages/providers_test.go`
- Modify: `internal/stages/providers.go:16-48`
- Modify: `internal/stages/progress.go:65-86,123-136`

- [ ] **Step 1: Add provider-path regression fixtures**

Create `internal/stages/providers_test.go` with a registered file-progress loader that can either succeed or fail after reporting every file:

```go
package stages_test

import (
	"bytes"
	"errors"
	"log/slog"
	"strings"
	"sync"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

const progressMetric metric.Name = "test-progress"

type progressLoader struct {
	onFile func()
	mu     sync.Mutex
	err    error
}

func (l *progressLoader) SetOnFileProcessed(fn func()) {
	l.onFile = fn
}

func (l *progressLoader) FileProgressMutex() *sync.Mutex {
	return &l.mu
}

func (l *progressLoader) Load(root *model.Directory, _ []metric.Name) error {
	for range root.Files {
		l.onFile()
	}

	return l.err
}

func registerProgressLoader(t *testing.T, loadErr error) {
	t.Helper()

	provider.ResetBaseRegistryForTesting()
	t.Cleanup(func() {
		provider.ResetBaseRegistryForTesting()
		filesystem.Register()
		git.Register()
	})

	loader := &progressLoader{err: loadErr}
	provider.RegisterLoader(provider.BaseMetricLoader{
		Metrics:  []metric.Name{progressMetric},
		Load:     loader.Load,
		Reporter: loader,
	})
}

func progressState() *stages.CommonState {
	return &stages.CommonState{
		Flags: &stages.Flags{},
		Root: &model.Directory{
			Files: []*model.File{{}, {}},
		},
		Requested: stages.RequestedMetrics{
			BaseMetrics: []metric.Name{progressMetric},
		},
	}
}
```

- [ ] **Step 2: Write the failing successful-provider test**

Append this test to `internal/stages/providers_test.go`:

```go
//nolint:paralleltest // mutates the global provider registry and slog logger
func TestRunProvidersReportsCompletedMetricProgress(t *testing.T) {
	g := NewGomegaWithT(t)
	registerProgressLoader(t, nil)

	var buf bytes.Buffer

	oldDefault := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{})))
	defer slog.SetDefault(oldDefault)

	g.Expect(stages.RunProviders(progressState())).To(Succeed())

	output := buf.String()
	g.Expect(output).To(ContainSubstring(`msg="Loading metrics." loaded=0/2 percentage=0.0`))
	g.Expect(output).To(ContainSubstring(`msg="Loaded metrics" loaded=2/2 percentage=100.0`))
	g.Expect(strings.LastIndex(output, `msg="Loaded metrics"`)).
		To(BeNumerically(">", strings.LastIndex(output, `msg="Loading metrics."`)))
}
```

- [ ] **Step 3: Write the failed-provider safety test**

Append this test to `internal/stages/providers_test.go`:

```go
//nolint:paralleltest // mutates the global provider registry and slog logger
func TestRunProvidersOmitsCompletionWhenLoadingFailsAtTotal(t *testing.T) {
	g := NewGomegaWithT(t)
	registerProgressLoader(t, errors.New("load failed after reporting progress"))

	var buf bytes.Buffer

	oldDefault := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&buf, &slog.HandlerOptions{})))
	defer slog.SetDefault(oldDefault)

	err := stages.RunProviders(progressState())

	g.Expect(err).To(MatchError(ContainSubstring("load failed after reporting progress")))
	g.Expect(buf.String()).NotTo(ContainSubstring(`msg="Loaded metrics"`))
}
```

- [ ] **Step 4: Run the provider tests to verify the successful case fails**

Run:

```bash
go test ./internal/stages -run TestRunProviders -count=1
```

Expected: `TestRunProvidersReportsCompletedMetricProgress` FAILS because no `Loaded metrics` line exists. The failure-path test should already pass.

- [ ] **Step 5: Make ticker shutdown synchronous**

Replace `startProgressTicker` in `internal/stages/progress.go` with:

```go
func startProgressTicker(logFn func()) (stop func()) {
	done := make(chan struct{})
	stopped := make(chan struct{})

	go func() {
		defer close(stopped)

		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				logFn()

			case <-done:
				return
			}
		}
	}()

	return func() {
		close(done)
		<-stopped
	}
}
```

This preserves the existing `func()` cleanup contract while guaranteeing that the ticker cannot log after `stop` returns.

- [ ] **Step 6: Add terminal metric formatting**

Add this function after `logMetricProgress` in `internal/stages/progress.go`:

```go
func logMetricCompletion(total int64) {
	slog.Info(
		"Loaded metrics",
		"loaded", fmt.Sprintf("%d/%d", total, total),
		"percentage", "100.0",
	)
}
```

- [ ] **Step 7: Move operation ownership into `RunProviders`**

Replace the body of `RunProviders` and extract the existing loading logic into `loadRequestedMetrics` in `internal/stages/providers.go`:

```go
func RunProviders(c *CommonState) error {
	slog.Info("Calculating metrics")

	total := provider.FileProgressTotal(c.Requested.BaseMetrics, model.CountFiles(c.Root))
	metricProg, stopMetricTicker := BuildMetricProgress(c.Flags, total)

	err := loadRequestedMetrics(c, metricProg)
	stopMetricTicker()

	if err != nil {
		return err
	}

	if metricProg != nil {
		logMetricCompletion(total)
	}

	return nil
}

func loadRequestedMetrics(c *CommonState, metricProg provider.MetricProgress) error {
	requested := c.Requested.BaseMetrics
	if hasAuthorshipMetric(requested) {
		if err := git.LoadAuthorshipMetricsInHistoryRange(
			c.Root,
			authorshipParams(c.RootConfig),
			c.Flags.HistoryRange,
		); err != nil {
			return eris.Wrap(err, "failed to load authorship metrics")
		}

		requested = withoutAuthorshipMetrics(requested)
	}

	requested, err := loadFileGitMetrics(c, requested, metricProg)
	if err != nil {
		return err
	}

	return eris.Wrap(
		provider.RunLoaders(c.Root, requested, metricProg),
		"failed to load metrics",
	)
}
```

This has one ticker shutdown point. The stop function remains pure cleanup, while `RunProviders` decides whether the operation succeeded.

- [ ] **Step 8: Format and run the stage tests**

Run:

```bash
task fmt
go test ./internal/stages -count=1
```

Expected: both commands PASS. The success log is last, and failed loading emits no completion.

- [ ] **Step 9: Commit provider completion**

```bash
git add internal/stages/progress.go internal/stages/providers.go internal/stages/providers_test.go
git commit -m "Report completed metric loading"
```

### Task 3: Verify the Complete Fix

**Files:**
- Verify: `internal/stages/progress.go`
- Verify: `internal/stages/providers.go`
- Verify: `internal/stages/progress_internal_test.go`
- Verify: `internal/stages/progress_test.go`
- Verify: `internal/stages/providers_test.go`

- [ ] **Step 1: Run all tests**

Run:

```bash
task test
```

Expected: PASS with no failing packages.

- [ ] **Step 2: Run repository CI**

Run `task ci` through an Explore or equivalent subagent because its intentionally verbose lint output must not flood the main session.

Expected summary:

```text
exit status: 0
failing linters: none
failing tests: none
```

- [ ] **Step 3: Confirm the final diff stays in scope**

Run:

```bash
git status --short
git diff --check
git diff --stat HEAD~2
git diff HEAD~2 -- .custom-gcl.yml
```

Expected:

- only the five implementation/test files from this plan are changed by the implementation commits;
- `git diff --check` produces no output;
- `.custom-gcl.yml` produces no diff.

- [ ] **Step 4: Confirm the API rejected in PR #729 was not introduced**

Run:

```bash
rg 'func\(bool\)' internal/stages
```

Expected: no matches.
