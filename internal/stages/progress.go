package stages

import (
	"fmt"
	"sync"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/progress"
)

// AcquisitionWork identifies the determinate work represented by a live
// acquisition stage.
func AcquisitionWork(c *CommonState) progress.WorkKind {
	if c.Requested.HasCommitExpressions() || hasAuthorshipMetric(c.Requested.BaseMetrics) {
		return progress.WorkCommits
	}

	return progress.WorkObservations
}

type progressAdapter struct {
	mu      sync.Mutex
	sink    progress.Sink
	current int64
	err     error
}

func (a *progressAdapter) Err() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	return a.err
}

func (a *progressAdapter) setTotal(total int64) {
	if total <= 0 {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.err == nil {
		a.err = a.sink.SetTotal(total)
	}
}

func (a *progressAdapter) advance() {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.err != nil {
		return
	}

	a.current++
	a.err = a.sink.SetProgress(a.current)
}

func (a *progressAdapter) status(message string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.err == nil {
		a.err = a.sink.SetStatus(message)
	}
}

type scanProgressAdapter struct {
	progressAdapter
	files int64
	dirs  int64
}

func newScanProgress(sink progress.Sink) *scanProgressAdapter {
	return &scanProgressAdapter{progressAdapter: progressAdapter{sink: sink}}
}

func (a *scanProgressAdapter) OnDirectoryScanned(_ string, fileCount int) {
	a.files += int64(fileCount)
	a.dirs++
	a.status(fmt.Sprintf("Discovered %d files in %d directories", a.files, a.dirs))
}

type metricProgressAdapter struct {
	progressAdapter
	selected bool
}

func newMetricProgress(sink progress.Sink, total int64) *metricProgressAdapter {
	a := &metricProgressAdapter{
		progressAdapter: progressAdapter{sink: sink},
		selected:        sink.WorkKind() == progress.WorkObservations,
	}
	if a.selected {
		a.setTotal(total)
	}

	return a
}

func (a *metricProgressAdapter) OnMetricStarted(name metric.Name) {
	a.status("Loading metric " + string(name))
}

func (a *metricProgressAdapter) OnMetricFinished(name metric.Name) {
	a.status("Loaded metric " + string(name))
}

func (a *metricProgressAdapter) OnFileProcessed(metric.Name) {
	if a.selected {
		a.advance()
	}
}

type historyProgressAdapter struct {
	progressAdapter
	selected bool
}

func newHistoryProgress(sink progress.Sink, total int64) *historyProgressAdapter {
	a := &historyProgressAdapter{
		progressAdapter: progressAdapter{sink: sink},
		selected:        sink.WorkKind() == progress.WorkCommits,
	}
	if a.selected {
		a.setTotal(total)
	}

	return a
}

func (a *historyProgressAdapter) OnCommit() {
	if a.selected {
		a.advance()
	}
}
