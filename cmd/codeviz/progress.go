package main

import (
	"log/slog"
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
func buildMetricProgress(flags *Flags) provider.MetricProgress {
	if !flags.Verbose && !flags.Debug {
		return nil
	}

	return &metricProgressLogger{}
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
		slog.Debug("scanned directory", "path", path, "files", s.files.Load(), "dirs", s.dirs.Load())
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
				slog.Debug("scanning...", "files", counter.files.Load(), "dirs", counter.dirs.Load())

			case <-done:
				return
			}
		}
	}()

	return func() { close(done) }
}

// metricProgressLogger implements provider.MetricProgress for verbose mode.
// OnMetricStarted and OnMetricFinished may be called concurrently.
type metricProgressLogger struct{}

func (*metricProgressLogger) OnMetricStarted(name metric.Name) {
	slog.Debug("metric started", "metric", string(name))
}

func (*metricProgressLogger) OnMetricFinished(name metric.Name) {
	slog.Debug("metric finished", "metric", string(name))
}
