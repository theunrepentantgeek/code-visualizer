package main

import (
	"log/slog"
	"strings"
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
// The caller must invoke the returned stop function when metric calculation completes.
func buildMetricProgress(flags *Flags) (provider.MetricProgress, func()) {
	if !flags.Verbose && !flags.Debug {
		return nil, func() {}
	}

	tracker := &metricProgressTracker{}
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

// startScanTicker starts a goroutine that logs cumulative scan progress every second.
// Call the returned stop function when scanning is done.
func startScanTicker(counter *scanCounter) (stop func()) {
	done := make(chan struct{})

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				slog.Debug("Scanning...", "files", counter.files.Load(), "dirs", counter.dirs.Load())

			case <-done:
				return
			}
		}
	}()

	return func() { close(done) }
}

// metricProgressTracker implements provider.MetricProgress for verbose mode.
// It tracks which metrics are active and how many have completed,
// providing meaningful progress during metric calculation.
type metricProgressTracker struct {
	mu        sync.Mutex
	active    []metric.Name
	completed atomic.Int64
}

func (t *metricProgressTracker) OnMetricStarted(name metric.Name) {
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
	done := make(chan struct{})

	go func() {
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				active := tracker.activeNames()
				if len(active) > 0 {
					slog.Debug(
						"Calculating...",
						"metric", joinMetricNames(active),
						"completed", tracker.completed.Load())
				}

			case <-done:
				return
			}
		}
	}()

	return func() { close(done) }
}

func removeMetric(names []metric.Name, target metric.Name) []metric.Name {
	for i, n := range names {
		if n == target {
			return append(names[:i], names[i+1:]...)
		}
	}

	return names
}

func joinMetricNames(names []metric.Name) string {
	strs := make([]string, len(names))
	for i, n := range names {
		strs[i] = string(n)
	}

	return strings.Join(strs, ", ")
}
