package stages_test

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
)

func TestExportConfig_NoFlag_NoOp(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &stages.CommonState{
		Flags:      &stages.Flags{ExportConfig: ""},
		RootConfig: config.New(),
	}

	g.Expect(stages.ExportConfig(s)).To(Succeed())
}

func TestExportConfig_OnlyWritesSelectedVizSection(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "out.yaml")
	rootCfg := config.New()

	s := &stages.CommonState{
		Flags:      &stages.Flags{ExportConfig: path},
		RootConfig: rootCfg,
		VizName:    "tree-map",
	}

	g.Expect(stages.ExportConfig(s)).To(Succeed())

	loaded := &config.Config{}
	g.Expect(loaded.Load(path)).To(Succeed())
	g.Expect(loaded.Treemap).NotTo(BeNil())
	g.Expect(loaded.Radial).To(BeNil())
	g.Expect(loaded.Bubbletree).To(BeNil())
	g.Expect(loaded.Spiral).To(BeNil())
	g.Expect(loaded.Scatter).To(BeNil())
}
