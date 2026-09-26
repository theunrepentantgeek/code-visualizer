package stages

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/progress"
)

type recordingSink struct {
	work     progress.WorkKind
	totals   []int64
	current  []int64
	statuses []string
}

func newTestSink(work progress.WorkKind) *recordingSink {
	return &recordingSink{work: work}
}

func (s *recordingSink) WorkKind() progress.WorkKind { return s.work }
func (s *recordingSink) SetTotal(total int64) error {
	s.totals = append(s.totals, total)
	return nil
}
func (s *recordingSink) SetProgress(current int64) error {
	s.current = append(s.current, current)
	return nil
}
func (s *recordingSink) SetStatus(status string) error {
	s.statuses = append(s.statuses, status)
	return nil
}

func TestScanProgressReportsDiscoveredFilesWithoutTotal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	sink := &recordingSink{work: progress.WorkObservations}
	adapter := newScanProgress(sink)

	adapter.OnDirectoryScanned("first", 2)
	adapter.OnDirectoryScanned("second", 3)

	g.Expect(adapter.Err()).NotTo(HaveOccurred())
	g.Expect(sink.totals).To(BeEmpty())
	g.Expect(sink.statuses).To(ContainElement("Discovered 5 files in 2 directories"))
}

func TestMetricProgressReportsExactObservationTotalAndAbsoluteProgress(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	sink := &recordingSink{work: progress.WorkObservations}
	adapter := newMetricProgress(sink, 3)

	adapter.OnMetricStarted(metric.Name("lines"))
	adapter.OnFileProcessed(metric.Name("lines"))
	adapter.OnFileProcessed(metric.Name("lines"))

	g.Expect(adapter.Err()).NotTo(HaveOccurred())
	g.Expect(sink.totals).To(Equal([]int64{3}))
	g.Expect(sink.current).To(Equal([]int64{1, 2}))
}

func TestHistoryProgressReportsCommitTotalAndAbsoluteProgress(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	sink := &recordingSink{work: progress.WorkCommits}
	adapter := newHistoryProgress(sink, 3)

	adapter.OnCommit()
	adapter.OnCommit()

	g.Expect(adapter.Err()).NotTo(HaveOccurred())
	g.Expect(sink.totals).To(Equal([]int64{3}))
	g.Expect(sink.current).To(Equal([]int64{1, 2}))
}

func TestNonSelectedProgressDoesNotReplaceTotal(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	sink := &recordingSink{work: progress.WorkCommits}
	adapter := newMetricProgress(sink, 3)

	adapter.OnMetricStarted(metric.Name("lines"))
	adapter.OnFileProcessed(metric.Name("lines"))

	g.Expect(adapter.Err()).NotTo(HaveOccurred())
	g.Expect(sink.totals).To(BeEmpty())
	g.Expect(sink.current).To(BeEmpty())
	g.Expect(sink.statuses).NotTo(BeEmpty())
}
