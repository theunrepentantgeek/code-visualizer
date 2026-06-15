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

	s := &stages.CommonState{
		TargetPath: "/no/such/dir",
		Requested:  stages.RequestedMetrics{BaseMetrics: []metric.Name{"file-size"}},
	}

	g.Expect(stages.CheckGitRequirement(s)).To(Succeed())
}

func TestCheckGitRequirement_Stage_FailsWhenGitMetricRequestedAndNoRepo(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()

	s := &stages.CommonState{
		TargetPath: dir,
		Requested:  stages.RequestedMetrics{BaseMetrics: []metric.Name{"file-age"}},
	}
	err := stages.CheckGitRequirement(s)

	var gre *stages.GitRequiredError
	g.Expect(errors.As(err, &gre)).To(BeTrue())
}

func TestCheckGitRepoHelper_NonGitDir_ReturnsGitRequiredError(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	err := stages.CheckGitRepoHelper(dir)

	var gre *stages.GitRequiredError
	g.Expect(errors.As(err, &gre)).To(BeTrue())
}
