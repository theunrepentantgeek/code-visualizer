package stages_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/progress"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/git"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
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

func TestAcquisitionWorkUsesCommitsForCommitExpressions(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	state := &stages.CommonState{
		Requested: stages.RequestedMetrics{
			Expressions: []provider.ResolvedMetric{{SourceLevel: metric.LevelCommit}},
		},
	}

	g.Expect(stages.AcquisitionWork(state)).To(Equal(progress.WorkCommits))
}

func TestAcquisitionWorkUsesCommitsForAuthorshipMetrics(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)
	state := &stages.CommonState{
		Requested: stages.RequestedMetrics{
			BaseMetrics: []metric.Name{git.CodeOwnerMetric},
		},
	}

	g.Expect(stages.AcquisitionWork(state)).To(Equal(progress.WorkCommits))
}

func TestAcquisitionWorkUsesObservationsOtherwise(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	g.Expect(stages.AcquisitionWork(&stages.CommonState{})).To(Equal(progress.WorkObservations))
}
