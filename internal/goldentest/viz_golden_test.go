package goldentest

import (
	"os"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/sebdah/goldie/v2"

	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/pipeline"
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

	return s.Err()
}

func TestGolden_Treemap(t *testing.T) {
	cases := []struct {
		name string
		ext  string
	}{
		{"treemap-png", "png"},
		{"treemap-svg", "svg"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out := filepath.Join(t.TempDir(), "out."+tc.ext)
			data := runViz(t, out, renderTreemap)

			g := goldie.New(t)
			g.Assert(t, tc.name, data)
		})
	}
}
