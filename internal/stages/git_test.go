package stages_test

import (
	"errors"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestCheckGitRequirementHelper_NoGitMetric_OK(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	g.Expect(stages.CheckGitRequirementHelper("/nonexistent", []metric.Name{"file-size"})).To(Succeed())
}

func TestCheckGitRequirement_Stage_SkipsWhenNoGitMetricRequested(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &fakeState{common: stages.CommonState{
		TargetPath: "/no/such/dir",
		Requested:  []metric.Name{"file-size"},
	}}

	g.Expect(stages.CheckGitRequirement[*fakeState](s)).To(Succeed())
}

func TestCheckGitRequirement_Stage_FailsWhenGitMetricRequestedAndNoRepo(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()

	s := &fakeState{common: stages.CommonState{
		TargetPath: dir,
		Requested:  []metric.Name{"file-age"},
	}}
	err := stages.CheckGitRequirement[*fakeState](s)

	var gre *stages.GitRequiredError
	g.Expect(errors.As(err, &gre)).To(BeTrue())
}
