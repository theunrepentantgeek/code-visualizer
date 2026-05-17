package stages_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestExportConfig_NoFlag_NoOp(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &fakeState{common: stages.CommonState{
		Flags:      &stages.Flags{ExportConfig: ""},
		RootConfig: config.New(),
	}}

	g.Expect(stages.ExportConfig[*fakeState](s)).To(Succeed())
}
