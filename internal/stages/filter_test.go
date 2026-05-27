package stages_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/filter"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestBuildFilterRulesHelper_MergesConfigAndCLI(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	rule, err := filter.ParseFilterFlag("*.go")
	g.Expect(err).NotTo(HaveOccurred())

	cfg := &config.Config{FileFilter: []filter.Rule{rule}}

	excludeRule, err := filter.NewRule("*_test.go", filter.Exclude)
	g.Expect(err).NotTo(HaveOccurred())

	got := stages.BuildFilterRulesHelper(cfg, []filter.Rule{excludeRule})

	g.Expect(got).To(HaveLen(2))
}

func TestBuildFilterRules_Stage_PopulatesCommon(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &fakeState{common: stages.CommonState{
		RootConfig: &config.Config{},
		CLIFilters: []filter.Rule{{Pattern: "*.go", Mode: filter.Include}},
	}}

	g.Expect(stages.BuildFilterRules[*fakeState](s)).To(Succeed())
	g.Expect(s.Common().FilterRules).To(HaveLen(1))
}
