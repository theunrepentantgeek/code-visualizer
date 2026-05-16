package main

import (
	"log/slog"
	"sync/atomic"
	"time"
)

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
