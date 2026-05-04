package main

import (
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/bevan/code-visualizer/internal/metric"
	"github.com/bevan/code-visualizer/internal/provider"
	"github.com/bevan/code-visualizer/internal/scan"
)

// buildScanProgress creates a scan.Progress adapter and (if applicable) starts a
// ticker goroutine that logs cumulative progress every second.
// The caller must invoke the returned stop function when scanning completes.
func buildScanProgress(flags *Flags) (scan.Progress, func()) {
	if !flags.Verbose && !flags.Debug {
		return nil, func() {}
	}

	counter := &scanCounter{debug: flags.Debug}
	stop := startScanTicker(counter)

	return counter, stop
}

// buildMetricProgress creates a provider.MetricProgress adapter for verbose mode.
// totalFiles is the number of files in the scanned tree, used as the denominator
// for per-file progress reporting.
// The caller must invoke the returned stop function when metric calculation completes.
func buildMetricProgress(flags *Flags, totalFiles int) (provider.MetricProgress, func()) {
	if !flags.Verbose && !flags.Debug {
		return nil, func() {}
	}

	tracker := &metricProgressTracker{totalFiles: totalFiles}
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
			"totaldirs", s.dirs.Load())
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
// It tracks which metrics are active, how many have completed, and per-file
// progress within each running metric.
type metricProgressTracker struct {
	mu         sync.Mutex
	active     []metric.Name
	completed  atomic.Int64
	totalFiles int
	fileCounts sync.Map // metric.Name -> *atomic.Int64
}

func (t *metricProgressTracker) OnMetricStarted(name metric.Name) {
	t.fileCounts.Store(name, &atomic.Int64{})

	t.mu.Lock()
	t.active = append(t.active, name)
	t.mu.Unlock()

	slog.Debug("Metric started", "metric", string(name))
}

func (t *metricProgressTracker) OnMetricFinished(name metric.Name) {
	t.mu.Lock()
	t.active = removeMetric(t.active, name)
	t.mu.Unlock()

	t.completed.Add(1)

	slog.Debug("Metric finished", "metric", string(name))
}

func (t *metricProgressTracker) OnFileProcessed(name metric.Name) {
	if v, ok := t.fileCounts.Load(name); ok {
		v.(*atomic.Int64).Add(1)
	}
}

// filesProcessed returns the number of files processed for the given metric.
func (t *metricProgressTracker) filesProcessed(name metric.Name) int64 {
	if v, ok := t.fileCounts.Load(name); ok {
		return v.(*atomic.Int64).Load()
	}

	return 0
}

// activeNames returns a snapshot of the currently active metric names.
func (t *metricProgressTracker) activeNames() []metric.Name {
	t.mu.Lock()
	defer t.mu.Unlock()

	result := make([]metric.Name, len(t.active))
	copy(result, t.active)

	return result
}

// startMetricTicker starts a goroutine that logs metric calculation progress every second.
// Call the returned stop function when metric calculation is done.
func startMetricTicker(tracker *metricProgressTracker) (stop func()) {
	return startProgressTicker(func() {
		logMetricProgress(tracker)
	})
}

func logMetricProgress(tracker *metricProgressTracker) {
	active := tracker.activeNames()

	for _, name := range active {
		files := tracker.filesProcessed(name)
		if files > 0 {
			slog.Debug("Calculating...",
				"metric", string(name),
				"files", fmt.Sprintf("%d/%d", files, tracker.totalFiles))
		} else {
			slog.Debug("Calculating...", "metric", string(name))
		}
	}
}

func removeMetric(names []metric.Name, target metric.Name) []metric.Name {
	for i, n := range names {
		if n == target {
			return append(names[:i], names[i+1:]...)
		}
	}

	return names
}

// buildHistoryProgress creates a per-commit callback and (if applicable) starts a
// ticker goroutine that logs commit history loading progress every second.
// The caller must invoke the returned stop function when loading completes.
func buildHistoryProgress(flags *Flags) (onCommit func(), stop func()) {
	if !flags.Verbose && !flags.Debug {
		return nil, func() {}
	}

	counter := &atomic.Int64{}
	stop = startHistoryTicker(counter)

	return func() { counter.Add(1) }, stop
}

// startHistoryTicker starts a goroutine that logs commit history progress every second.
func startHistoryTicker(counter *atomic.Int64) (stop func()) {
	return startProgressTicker(func() {
		slog.Debug("Loading history...",
			"commits", counter.Load())
	})
}
