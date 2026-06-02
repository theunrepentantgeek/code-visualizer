package treemap_test

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/metric"
	"github.com/theunrepentantgeek/code-visualizer/internal/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/provider/filesystem"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
	"github.com/theunrepentantgeek/code-visualizer/internal/treemap"
)

func TestResolveMetrics_SizeOnly(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	s := &treemap.State{
		Config: &config.Treemap{Size: &sizeStr},
	}

	g.Expect(treemap.ResolveMetrics(s)).To(Succeed())
	g.Expect(s.Size).To(Equal(metric.Name("file-size")))
	g.Expect(s.FillMetric).To(Equal(metric.Name("file-size")))
	g.Expect(s.Common().Requested).To(ConsistOf(metric.Name("file-size")))
}

func TestResolveMetrics_FillOverridesSizeAsFillMetric(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	sizeStr := "file-size"
	s := &treemap.State{
		Config: &config.Treemap{
			Size: &sizeStr,
			Fill: &config.MetricSpec{Metric: "file-type"},
		},
	}

	g.Expect(treemap.ResolveMetrics(s)).To(Succeed())
	g.Expect(s.FillMetric).To(Equal(metric.Name("file-type")))
	g.Expect(s.Common().Requested).To(ContainElements(metric.Name("file-size"), metric.Name("file-type")))
}

func TestState_CommonReturnsEmbeddedPointer(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &treemap.State{}
	c := s.Common()
	c.Width = 42
	g.Expect(s.CommonState.Width).To(Equal(42))
}

func TestState_IncludeBinary(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	on := &treemap.State{IncludeBinaryFiles: true}
	off := &treemap.State{IncludeBinaryFiles: false}

	g.Expect(on.IncludeBinary()).To(BeTrue())
	g.Expect(off.IncludeBinary()).To(BeFalse())

	var _ stages.BinaryFilterToggler = on
}

func TestBuildInksStage_WrapsFillInkUnlessFlat(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name        string
		flat        bool
		wantWrapped bool
	}{
		{name: "gradient", flat: false, wantWrapped: true},
		{name: "flat", flat: true, wantWrapped: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			g := NewGomegaWithT(t)

			root := &model.Directory{
				Name:  "root",
				Files: []*model.File{makeTestFile("a.go", "go", 100)},
			}

			s := &treemap.State{
				CommonState: stages.CommonState{Root: root, Output: "out.png", Width: 100, Height: 100},
				FillMetric:  filesystem.FileSize,
				FillPalette: palette.Temperature,
				Flat:        tc.flat,
			}

			g.Expect(treemap.BuildInksStage(s)).To(Succeed())

			_, isWrapped := s.Inks.Fill.(*canvas.RadialGradientInk)
			g.Expect(isWrapped).To(Equal(tc.wantWrapped))
		})
	}
}

func TestBuildLegendStage_AddsLabelSampleLines(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	s := &treemap.State{
		Config: &config.Treemap{
			Fill:   &config.MetricSpec{Metric: "file-type"},
			Border: &config.MetricSpec{Metric: "file-lines"},
		},
		FillMetric:   metric.Name("file-type"),
		BorderMetric: metric.Name("file-lines"),
		Size:         metric.Name("file-size"),
		Inks: treemap.Inks{
			Fill:   canvas.FixedInk(color.RGBA{R: 255, G: 255, B: 255, A: 255}),
			Border: canvas.FixedInk(color.RGBA{R: 0, G: 0, B: 0, A: 255}),
		},
	}

	g.Expect(treemap.BuildLegendStage(s)).To(Succeed())
	g.Expect(s.LegendConfig).NotTo(BeNil())

	if s.LegendConfig == nil {
		return
	}

	g.Expect(s.LegendConfig.LabelSample).To(Equal([]string{
		"file-name",
		"file-size",
		"file-type",
		"file-lines",
	}))
}

func TestLayoutStage_FooterEnabled_ReducesAvailableHeight(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeTestFile("a.go", "go", 100)},
	}

	cfg := config.New()
	// Footer has text set by default (from config.New()), so it will be shown.

	const width, height = 800, 600

	s := &treemap.State{
		CommonState: stages.CommonState{
			Root:       root,
			Width:      width,
			Height:     height,
			RootConfig: cfg,
		},
		Size:        metric.Name("file-size"),
		FillMetric:  metric.Name("file-size"),
		FillPalette: palette.Temperature,
	}

	g.Expect(treemap.LayoutStage(s)).To(Succeed())

	// The layout rectangle must not extend into the footer zone.
	footerH := canvas.FooterReservedHeight
	maxY := s.Root.Y + s.Root.H
	g.Expect(maxY).To(BeNumerically("<=", float64(height)-footerH),
		"layout rect extends into footer zone")
}

func TestLayoutStage_FooterDisabled_UsesFullHeight(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	root := &model.Directory{
		Name:  "root",
		Files: []*model.File{makeTestFile("a.go", "go", 100)},
	}

	cfgWithFooter := config.New()
	cfgWithFooter.OverrideHideFooter(true)

	const width, height = 800, 600

	sNoFooter := &treemap.State{
		CommonState: stages.CommonState{
			Root:       root,
			Width:      width,
			Height:     height,
			RootConfig: cfgWithFooter,
		},
		Size:        metric.Name("file-size"),
		FillMetric:  metric.Name("file-size"),
		FillPalette: palette.Temperature,
	}

	sWithFooter := &treemap.State{
		CommonState: stages.CommonState{
			Root:       root,
			Width:      width,
			Height:     height,
			RootConfig: config.New(),
		},
		Size:        metric.Name("file-size"),
		FillMetric:  metric.Name("file-size"),
		FillPalette: palette.Temperature,
	}

	g.Expect(treemap.LayoutStage(sNoFooter)).To(Succeed())
	g.Expect(treemap.LayoutStage(sWithFooter)).To(Succeed())

	// With footer hidden, layout uses more vertical space than when footer is shown.
	maxYNoFooter := sNoFooter.Root.Y + sNoFooter.Root.H
	maxYWithFooter := sWithFooter.Root.Y + sWithFooter.Root.H
	g.Expect(maxYNoFooter).To(BeNumerically(">", maxYWithFooter),
		"footer-hidden layout should use more height than footer-shown layout")
}
