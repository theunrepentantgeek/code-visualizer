package stages_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
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

// ExportData tests

func TestExportData_EmptyPath_NoOp(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &stages.CommonState{
		Flags: &stages.Flags{ExportData: ""},
		Root:  &model.Directory{Name: "root"},
	}

	g.Expect(stages.ExportData(s)).To(Succeed())
}

func TestExportData_WritesJSON(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	dir := t.TempDir()
	path := filepath.Join(dir, "data.json")

	f := &model.File{Name: "main.go", Path: "main.go"}
	f.SetQuantity(metric.Name("file-size"), 1024)

	root := &model.Directory{Name: "root"}
	root.Files = []*model.File{f}

	s := &stages.CommonState{
		Flags:    &stages.Flags{ExportData: path},
		Root:     root,
		Requested: stages.RequestedMetrics{BaseMetrics: []metric.Name{"file-size"}},
	}

	g.Expect(stages.ExportData(s)).To(Succeed())

	raw, err := os.ReadFile(path)
	g.Expect(err).NotTo(HaveOccurred())

	var parsed map[string]any
	g.Expect(json.Unmarshal(raw, &parsed)).To(Succeed())
	g.Expect(parsed).To(HaveKey("root"))
}
