package alluvial_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/alluvial"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/progress"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

type inactiveSink struct{}

func (inactiveSink) WorkKind() progress.WorkKind { return progress.WorkNone }
func (inactiveSink) SetTotal(int64) error        { return nil }
func (inactiveSink) SetProgress(int64) error     { return nil }
func (inactiveSink) SetStatus(string) error      { return nil }

func TestAcquisitionPlanRejectsReferenceBeforePreparation(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	plan := alluvial.NewAcquisitionPlan()
	err := plan.AcquireReference(
		context.Background(),
		inactiveSink{},
		&alluvial.State{},
		"HEAD",
		0,
	)

	g.Expect(err).To(MatchError(ContainSubstring("reference index 0 is out of range")))
}

func TestAcquisitionPlanRequestsResolvedLabelMetricsForSnapshots(t *testing.T) {
	t.Parallel()
	g := NewWithT(t)

	metricName := "file-size"
	state := &alluvial.State{}
	common := &stages.CommonState{}

	g.Expect(alluvial.ResolveMetrics(common, state, &config.Alluvial{
		Metric:     &metricName,
		References: []string{"before", "after"},
		Labels:     []metric.Name{"file-lines", "file-type"},
	})).To(Succeed())

	g.Expect(state.LabelMetrics).To(Equal([]metric.Name{"file-lines.sum", "file-type.mode"}))
	g.Expect(common.Requested.Expressions).To(HaveLen(3))
	g.Expect(common.Requested.Expressions[1].ResultName).To(Equal(metric.Name("file-lines.sum")))
	g.Expect(common.Requested.Expressions[2].ResultName).To(Equal(metric.Name("file-type.mode")))
}
