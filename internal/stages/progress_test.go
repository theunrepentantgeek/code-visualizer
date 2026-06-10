package stages_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

// ---------------------------------------------------------------------------
// BuildMetricProgress — flag-based gating
// ---------------------------------------------------------------------------

func TestBuildMetricProgress_VerboseMode_ReturnsTracker(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	flags := &stages.Flags{Verbose: true}
	prog, stop := stages.BuildMetricProgress(flags, 10)
	defer stop()

	g.Expect(prog).NotTo(BeNil())
}

func TestBuildMetricProgress_DebugMode_ReturnsTracker(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	flags := &stages.Flags{Debug: true}
	prog, stop := stages.BuildMetricProgress(flags, 10)
	defer stop()

	g.Expect(prog).NotTo(BeNil())
}

func TestBuildMetricProgress_QuietMode_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	flags := &stages.Flags{Quiet: true}
	prog, stop := stages.BuildMetricProgress(flags, 10)
	defer stop()

	g.Expect(prog).To(BeNil())
}

func TestBuildMetricProgress_Stop_IsCallable(t *testing.T) {
	t.Parallel()

	flags := &stages.Flags{Verbose: true}
	_, stop := stages.BuildMetricProgress(flags, 10)

	// Stop must not panic or block.
	stop()
}

func TestBuildMetricProgress_NilStop_IsCallableWhenSuppressed(t *testing.T) {
	t.Parallel()

	flags := &stages.Flags{Quiet: true}
	_, stop := stages.BuildMetricProgress(flags, 10)

	// no-op stop must not panic.
	stop()
}

// ---------------------------------------------------------------------------
// metricProgressTracker callbacks
// ---------------------------------------------------------------------------

func TestBuildMetricProgress_OnMetricStarted_RecordsMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	flags := &stages.Flags{Verbose: true}
	prog, stop := stages.BuildMetricProgress(flags, 5)
	defer stop()

	prog.OnMetricStarted(metric.Name("file-lines"))
	prog.OnMetricStarted(metric.Name("commit-count"))

	// After starting both metrics, finishing one should leave the other active.
	// We verify this by finishing one and then checking OnFileProcessed
	// doesn't panic (it should be a no-op for finished metrics).
	prog.OnMetricFinished(metric.Name("file-lines"))
	prog.OnFileProcessed(metric.Name("file-lines"))

	// Remaining metric still works.
	prog.OnFileProcessed(metric.Name("commit-count"))
	prog.OnMetricFinished(metric.Name("commit-count"))

	g.Expect(true).To(BeTrue()) // reached without panic
}

func TestBuildMetricProgress_OnFileProcessed_UnknownMetric_IsNoop(t *testing.T) {
	t.Parallel()

	flags := &stages.Flags{Verbose: true}
	prog, stop := stages.BuildMetricProgress(flags, 5)
	defer stop()

	// Processing a file for a metric that was never started should not panic.
	prog.OnFileProcessed(metric.Name("unknown-metric"))
}

// ---------------------------------------------------------------------------
// BuildScanProgress — flag-based gating
// ---------------------------------------------------------------------------

func TestBuildScanProgress_VerboseMode_ReturnsProgress(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	flags := &stages.Flags{Verbose: true}
	prog, stop := stages.BuildScanProgress(flags)
	defer stop()

	g.Expect(prog).NotTo(BeNil())
}

func TestBuildScanProgress_DebugMode_ReturnsProgress(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	flags := &stages.Flags{Debug: true}
	prog, stop := stages.BuildScanProgress(flags)
	defer stop()

	g.Expect(prog).NotTo(BeNil())
}

// ---------------------------------------------------------------------------
// BuildHistoryProgress — flag-based gating
// ---------------------------------------------------------------------------

func TestBuildHistoryProgress_VerboseMode_ReturnsCallback(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	flags := &stages.Flags{Verbose: true}
	onCommit, stop := stages.BuildHistoryProgress(flags)
	defer stop()

	g.Expect(onCommit).NotTo(BeNil())
}

func TestBuildHistoryProgress_DebugMode_ReturnsCallback(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	flags := &stages.Flags{Debug: true}
	onCommit, stop := stages.BuildHistoryProgress(flags)
	defer stop()

	g.Expect(onCommit).NotTo(BeNil())
}

func TestBuildHistoryProgress_QuietMode_ReturnsNil(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	flags := &stages.Flags{Quiet: true}
	onCommit, stop := stages.BuildHistoryProgress(flags)
	defer stop()

	g.Expect(onCommit).To(BeNil())
}

func TestBuildHistoryProgress_Callback_IsCallable(t *testing.T) {
	t.Parallel()

	flags := &stages.Flags{Verbose: true}
	onCommit, stop := stages.BuildHistoryProgress(flags)
	defer stop()

	// The callback should not panic when invoked.
	onCommit()
	onCommit()
}
