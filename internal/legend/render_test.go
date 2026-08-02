package legend_test

import (
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/canvas"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/mock"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
)

func TestRenderInto_DecomposesToPrimitives(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cv := canvas.NewCanvas(800, 600)

	pal := palette.GetPalette(palette.Temperature)
	fillInk := inks.NumericInk("file-size", []float64{10, 50, 100}, pal)

	cfg := &legend.Config{
		Position:    model.LegendPositionBottomRight,
		Orientation: model.LegendOrientationVertical,
		Entries: []legend.Entry{
			{Role: legend.RoleFill, MetricName: "file-size", Ink: fillInk},
		},
	}

	legend.RenderInto(cv, cfg)

	mb := mock.NewBackend()
	err := cv.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(mb.Calls).NotTo(BeEmpty())
	g.Expect(mb.Calls[0].Method).To(Equal("DrawRectangle"))

	hasLabel := false
	hasMetric := false

	for _, call := range mb.Calls {
		if call.Method == "DrawText" && call.Text == "Fill" {
			hasLabel = true
		}

		if call.Method == "DrawText" && call.Text == "file-size" {
			hasMetric = true
		}
	}

	g.Expect(hasLabel).To(BeTrue(), "expected label text 'Fill'")
	g.Expect(hasMetric).To(BeTrue(), "expected metric text 'file-size'")
}

func TestRenderInto_HorizontalOrientation_RendersAllSwatches(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cv := canvas.NewCanvas(1200, 200)

	pal := palette.GetPalette(palette.Temperature)
	fillInk := inks.NumericInk("file-size", []float64{10, 50, 100}, pal)

	cfg := &legend.Config{
		Position:    model.LegendPositionBottomCenter,
		Orientation: model.LegendOrientationHorizontal,
		Entries: []legend.Entry{
			{Role: legend.RoleFill, MetricName: "file-size", Ink: fillInk},
		},
	}

	legend.RenderInto(cv, cfg)

	mb := mock.NewBackend()
	err := cv.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())

	// Horizontal layout should still emit label and metric text
	hasLabel := false
	hasMetric := false

	for _, call := range mb.Calls {
		if call.Method == "DrawText" && call.Text == "Fill" {
			hasLabel = true
		}

		if call.Method == "DrawText" && call.Text == "file-size" {
			hasMetric = true
		}
	}

	g.Expect(hasLabel).To(BeTrue(), "expected label text 'Fill'")
	g.Expect(hasMetric).To(BeTrue(), "expected metric text 'file-size'")
}

func TestRenderInto_CategoricalEntry_RendersSwatchPerCategory(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cv := canvas.NewCanvas(800, 600)

	catInk := inks.CategoricalInk("file-type", []string{"go", "json", "yaml"}, palette.GetPalette(palette.Categorization))

	cfg := &legend.Config{
		Position:    model.LegendPositionBottomRight,
		Orientation: model.LegendOrientationVertical,
		Entries: []legend.Entry{
			{Role: legend.RoleFill, MetricName: "file-type", Ink: catInk},
		},
	}

	legend.RenderInto(cv, cfg)

	mb := mock.NewBackend()
	err := cv.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())

	// Each category should appear as a label
	categoryLabels := map[string]bool{}

	for _, call := range mb.Calls {
		if call.Method == "DrawText" {
			categoryLabels[call.Text] = true
		}
	}

	g.Expect(categoryLabels).To(HaveKey("go"))
	g.Expect(categoryLabels).To(HaveKey("json"))
	g.Expect(categoryLabels).To(HaveKey("yaml"))
}

func TestRenderInto_BorderEntry_UsesOutlineSwatch(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cv := canvas.NewCanvas(800, 600)

	pal := palette.GetPalette(palette.Temperature)
	borderInk := inks.NumericInk("commit-density", []float64{1, 5, 10}, pal)

	cfg := &legend.Config{
		Position:    model.LegendPositionBottomRight,
		Orientation: model.LegendOrientationVertical,
		Entries: []legend.Entry{
			{Role: legend.RoleBorder, MetricName: "commit-density", Ink: borderInk},
		},
	}

	legend.RenderInto(cv, cfg)

	mb := mock.NewBackend()
	err := cv.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())

	hasMetric := false

	for _, call := range mb.Calls {
		if call.Method == "DrawText" && call.Text == "commit-density" {
			hasMetric = true
		}
	}

	g.Expect(hasMetric).To(BeTrue(), "expected metric text 'commit-density'")

	// Should have rendered at least the background + swatch rectangles
	rectCount := 0

	for _, call := range mb.Calls {
		if call.Method == "DrawRectangle" {
			rectCount++
		}
	}

	g.Expect(rectCount).To(BeNumerically(">", 1))
}

func TestRenderInto_LabelSample_RendersSampleBeforeEntries(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cv := canvas.NewCanvas(800, 600)
	pal := palette.GetPalette(palette.Temperature)
	fillInk := inks.NumericInk("file-size", []float64{10, 50, 100}, pal)

	cfg := &legend.Config{
		Position:    model.LegendPositionBottomRight,
		Orientation: model.LegendOrientationVertical,
		LabelSample: []string{"file-name", "file-size", "file-type"},
		Entries: []legend.Entry{
			{Role: legend.RoleFill, MetricName: "file-size", Ink: fillInk},
		},
	}

	legend.RenderInto(cv, cfg)

	mb := mock.NewBackend()
	err := cv.RenderTo(mb)
	g.Expect(err).NotTo(HaveOccurred())

	var (
		sampleY     float64
		sampleFound bool
		titleY      float64
		titleFound  bool
	)

	for _, call := range mb.Calls {
		if call.Method == "DrawText" && call.Text == "file-name" {
			sampleY = call.Pos.Y
			sampleFound = true
		}

		if call.Method == "DrawText" && call.Text == "Fill" {
			titleY = call.Pos.Y
			titleFound = true
		}
	}

	g.Expect(sampleFound).To(BeTrue())
	g.Expect(titleFound).To(BeTrue())
	g.Expect(sampleY).To(BeNumerically("<", titleY))
}
