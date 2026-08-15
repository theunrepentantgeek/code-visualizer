package stages

import (
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/scan"
)

// BuildScanProgress creates a scan.Progress adapter and (if applicable) starts a
// ticker goroutine that logs cumulative progress every second.
// The caller must invoke the returned stop function when scanning completes.
func BuildScanProgress(flags *Flags) (scan.Progress, func()) {
	if !flags.Verbose && !flags.Debug {
		return nil, func() {}
	}

	counter := &scanCounter{debug: flags.Debug}
	stop := startScanTicker(counter)

	return counter, stop
}

// BuildMetricProgress creates a provider.MetricProgress adapter that logs periodic
// progress during metric calculation.
// The caller must invoke the returned stop function when metric calculation completes.
func BuildMetricProgress(flags *Flags, total int64) (provider.MetricProgress, func()) {
	if flags.Quiet {
		return nil, func() {}
	}

	tracker := &metricProgressTracker{total: total}
	stop := startMetricTicker(tracker)

	return tracker, stop
}

// scanCounter implements scan.Progress and tracks cumulative scan totals.
// In debug mode it also logs a line per directory.
type scanCounter struct {
	files atomic.Int64
	dirs  atomic.Int64
	debug bool
}

func (s *scanCounter) OnDirectoryScanned(path string, fileCount int) {
	s.files.Add(int64(fileCount))
	s.dirs.Add(1)

	if s.debug {
		slog.Debug(
			"Scanned directory",
			"path", path,
			"newfiles", fileCount,
			"totalfiles", s.files.Load(),
			"totaldirs", s.dirs.Load(),
		)
	}
}

// startProgressTicker starts a goroutine that calls logFn every second.
// Call the returned stop function when the operation completes.
func startProgressTicker(logFn func()) (stop func()) {
	done := make(chan struct{})

	go func() {
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

	return func() { close(done) }
}

// startScanTicker starts a goroutine that logs cumulative scan progress every second.
// Call the returned stop function when scanning is done.
func startScanTicker(counter *scanCounter) (stop func()) {
	return startProgressTicker(func() {
		slog.Debug("Scanning...", "files", counter.files.Load(), "dirs", counter.dirs.Load())
	})
}

// metricProgressTracker implements provider.MetricProgress for verbose mode.
// It tracks the number of metric observations loaded by file-based loaders.
type metricProgressTracker struct {
	loaded atomic.Int64
	total  int64
}

func (*metricProgressTracker) OnMetricStarted(name metric.Name) {
	slog.Debug("Metric started", "metric", string(name))
}

func (*metricProgressTracker) OnMetricFinished(name metric.Name) {
	slog.Debug("Metric finished", "metric", string(name))
}

func (t *metricProgressTracker) OnFileProcessed(metric.Name) { t.loaded.Add(1) }

// startMetricTicker starts a goroutine that logs metric calculation progress every second.
// Call the returned stop function when metric calculation is done.
func startMetricTicker(tracker *metricProgressTracker) (stop func()) {
	return startProgressTicker(func() {
		logMetricProgress(tracker)
	})
}

func logMetricProgress(tracker *metricProgressTracker) {
	loaded := tracker.loaded.Load()
	percentage := int64(0)

	if tracker.total > 0 {
		percentage = min(loaded*100/tracker.total, 100)
	}

	slog.Info(
		"Loading metrics.",
		"loaded", loaded,
		"total", tracker.total,
		"percentage", percentage,
	)
}

// BuildHistoryProgress creates a per-commit callback and (if applicable) starts a
// ticker goroutine that logs commit history loading progress every second.
// The caller must invoke the returned stop function when loading completes.
func BuildHistoryProgress(flags *Flags) (onCommit func(), stop func()) {
	if flags.Quiet {
		return nil, func() {}
	}

	counter := &atomic.Int64{}
	stop = startHistoryTicker(counter)

	return func() { counter.Add(1) }, stop
}

func startHistoryTicker(counter *atomic.Int64) (stop func()) {
	return startProgressTicker(func() {
		logHistoryProgress(counter)
	})
}

func logHistoryProgress(counter *atomic.Int64) {
	slog.Info("Loading history.", "commits", counter.Load())
}
