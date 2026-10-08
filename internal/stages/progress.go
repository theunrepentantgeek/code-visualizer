package stages

import (
	"fmt"
	"slices"
	"sync"

	"github.com/rotisserie/eris"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/progress"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
)

// NeedsGitMetrics reports whether acquisition includes Git-backed metric work.
func NeedsGitMetrics(c *CommonState) bool {
	return c.VizName == spiralVisualization ||
		c.Requested.HasCommitExpressions() ||
		slices.ContainsFunc(c.Requested.BaseMetrics, git.IsGitMetric)
}

// AcquisitionWork identifies the determinate work used by Alluvial reference
// acquisition.
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

type progressSegment struct {
	sink   progress.Sink
	total  int64
	offset int64
}

func (s progressSegment) WorkKind() progress.WorkKind { return s.sink.WorkKind() }

func (s progressSegment) SetTotal(total int64) error {
	if total != s.total {
		return fmt.Errorf("progress segment total %d does not match expected total %d", total, s.total)
	}

	return nil
}

func (s progressSegment) SetProgress(current int64) error {
	return eris.Wrap(s.sink.SetProgress(s.offset+current), "report progress segment")
}

func (s progressSegment) SetStatus(message string) error {
	return eris.Wrap(s.sink.SetStatus(message), "report progress segment status")
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
	return &scanProgressAdapter{sink: sink}
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
		sink:     sink,
		selected: sink.WorkKind() == progress.WorkObservations,
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
		sink:     sink,
		selected: sink.WorkKind() == progress.WorkCommits,
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
