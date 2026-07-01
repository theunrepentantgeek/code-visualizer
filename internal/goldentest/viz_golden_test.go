package goldentest

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/rotisserie/eris"
	"github.com/sebdah/goldie/v2"

	"github.com/theunrepentantgeek/code-visualizer/internal/bubbletree"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
	"github.com/theunrepentantgeek/code-visualizer/internal/radialtree"
	scatterviz "github.com/theunrepentantgeek/code-visualizer/internal/scatter"
	"github.com/theunrepentantgeek/code-visualizer/internal/spiral"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
	"github.com/theunrepentantgeek/code-visualizer/internal/treemap"
)

// vizFixtureWidth/Height keep golden images small and fast while still
// exercising layout, legend, title and footer.
const (
	vizFixtureWidth  = 320
	vizFixtureHeight = 240
)

// newCommonState builds a CommonState with the synthetic model injected and a
// config whose dimensions are the small fixture size. outputPath drives the
// WriteCanvas format (png/svg) via its extension.
func newCommonState(outputPath string, cfg *config.Config) *stages.CommonState {
	w, h := vizFixtureWidth, vizFixtureHeight
	cfg.ImageSize = &config.ImageSize{Width: &w, Height: &h}

	// config.New seeds the footer with "...at $time on $date", which ApplyFooter
	// expands via time.Now() — non-deterministic. Pin it to a static string so
	// golden images stay byte-stable across runs while still rendering a footer.
	*cfg.Footer.Text = "code-visualizer golden"

	return &stages.CommonState{
		Output:     outputPath,
		Flags:      &stages.Flags{Config: cfg},
		RootConfig: cfg,
		VizName:    "golden",
		Root:       buildVizModel(),
	}
}

// runViz writes the visualization to outputPath using the supplied render
// closure, then returns the bytes.
func runViz(t *testing.T, outputPath string, render func(*stages.CommonState) error) []byte {
	t.Helper()
	g := NewGomegaWithT(t)

	cfg := config.New()
	common := newCommonState(outputPath, cfg)

	g.Expect(render(common)).To(Succeed())

	data, err := os.ReadFile(outputPath)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(data).NotTo(BeEmpty())

	return data
}

// renderTreemap resolves metrics and runs treemap.RenderPipeline against the
// pre-built model. size=file-lines, fill=file-type mirrors the structure preset.
func renderTreemap(common *stages.CommonState) error {
	cfg := common.RootConfig
	size := "file-lines"
	cfg.Treemap = &config.Treemap{
		Size: &size,
		Fill: &config.MetricSpec{Metric: "file-type"},
	}

	viz := &treemap.State{}
	s := pipeline.NewState(common, cfg.Treemap, viz)

	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncX(s, stages.RegisterSelectionMetrics)
	pipeline.ApplyFuncXYZ(s, treemap.ResolveMetrics)
	treemap.RenderPipeline(s)

	return eris.Wrap(s.Err(), "treemap render failed")
}

//nolint:paralleltest // mutates the global metric registry
func TestGolden_Treemap(t *testing.T) {
	runVizGolden(t, "treemap", renderTreemap)
}

// renderRadial: discSize=file-lines, fill=file-type.
func renderRadial(common *stages.CommonState) error {
	cfg := common.RootConfig
	discSize := "file-lines"

	if cfg.Radial == nil {
		cfg.Radial = &config.Radial{}
	}

	cfg.Radial.DiscSize = &discSize
	cfg.Radial.Fill = &config.MetricSpec{Metric: "file-type"}

	viz := &radialtree.State{}
	s := pipeline.NewState(common, cfg.Radial, viz)

	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncX(s, stages.RegisterSelectionMetrics)
	pipeline.ApplyFuncXYZ(s, radialtree.ResolveMetrics)
	radialtree.RenderPipeline(s)

	return eris.Wrap(s.Err(), "radial render failed")
}

// renderBubbletree: size=file-lines, fill=file-type.
func renderBubbletree(common *stages.CommonState) error {
	cfg := common.RootConfig
	size := "file-lines"

	if cfg.Bubbletree == nil {
		cfg.Bubbletree = &config.Bubbletree{}
	}

	cfg.Bubbletree.Size = &size
	cfg.Bubbletree.Fill = &config.MetricSpec{Metric: "file-type"}

	viz := &bubbletree.State{}
	s := pipeline.NewState(common, cfg.Bubbletree, viz)

	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncX(s, stages.RegisterSelectionMetrics)
	pipeline.ApplyFuncXYZ(s, bubbletree.ResolveMetrics)
	bubbletree.RenderPipeline(s)

	return eris.Wrap(s.Err(), "bubbletree render failed")
}

// renderScatter: x-axis=file-size, y-axis=file-lines, size=file-lines, fill=file-type.
func renderScatter(common *stages.CommonState) error {
	cfg := common.RootConfig
	x := "file-size"
	y := "file-lines"
	size := "file-lines"

	if cfg.Scatter == nil {
		cfg.Scatter = &config.Scatter{}
	}

	cfg.Scatter.XAxis = &x
	cfg.Scatter.YAxis = &y
	cfg.Scatter.Size = &size
	cfg.Scatter.Fill = &config.MetricSpec{Metric: "file-type"}

	viz := &scatterviz.State{}
	s := pipeline.NewState(common, cfg.Scatter, viz)

	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncX(s, stages.RegisterSelectionMetrics)
	pipeline.ApplyFuncXYZ(s, scatterviz.ResolveMetrics)
	scatterviz.RenderPipeline(s)

	return eris.Wrap(s.Err(), "scatter render failed")
}

//nolint:paralleltest // mutates the global metric registry
func TestGolden_Radial(t *testing.T) { runVizGolden(t, "radial", renderRadial) }

//nolint:paralleltest // mutates the global metric registry
func TestGolden_Bubbletree(t *testing.T) { runVizGolden(t, "bubbletree", renderBubbletree) }

//nolint:paralleltest // mutates the global metric registry
func TestGolden_Scatter(t *testing.T) { runVizGolden(t, "scatter", renderScatter) }

// renderSpiral: size=file-lines, fill=file-type, with synthetic git history
// injected so the time-bucket stages have data.
func renderSpiral(common *stages.CommonState) error {
	cfg := common.RootConfig
	size := "file-lines"

	if cfg.Spiral == nil {
		cfg.Spiral = &config.Spiral{}
	}

	cfg.Spiral.Size = &size
	cfg.Spiral.Fill = &config.MetricSpec{Metric: "file-type"}

	common.FileHistory, common.FileTimeRange = buildSpiralHistory(common.Root)

	viz := &spiral.State{}
	s := pipeline.NewState(common, cfg.Spiral, viz)

	pipeline.ApplyFuncX(s, stages.BuildFilterRules)
	pipeline.ApplyFuncX(s, stages.RegisterSelectionMetrics)
	pipeline.ApplyFuncXYZ(s, spiral.ResolveMetrics)
	spiral.RenderPipeline(s)

	return eris.Wrap(s.Err(), "spiral render failed")
}

//nolint:paralleltest // mutates the global metric registry
func TestGolden_Spiral(t *testing.T) { runVizGolden(t, "spiral", renderSpiral) }

// runVizGolden renders the named viz to PNG and SVG and golden-compares both.
func runVizGolden(t *testing.T, name string, render func(*stages.CommonState) error) {
	t.Helper()

	for _, ext := range []string{"png", "svg"} {
		t.Run(name+"-"+ext, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "out."+ext)
			data := runViz(t, out, render)

			g := goldie.New(t)
			g.Assert(t, name+"-"+ext, data)
		})
	}
}
