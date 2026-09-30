package alluvial_test

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/alluvial"
	"github.com/theunrepentantgeek/code-visualizer/internal/progress"
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
