package legend_test

import (
	"slices"
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

	catInk := inks.CategoricalInk(
		"file-type",
		[]string{"go", "json", "yaml"},
		palette.GetPalette(palette.Categorization),
	)

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

func TestRenderInto_DefaultSquareLabelSample_RendersSampleBeforeEntries(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cv := canvas.NewCanvas(800, 600)
	pal := palette.GetPalette(palette.Temperature)
	fillInk := inks.NumericInk("file-size", []float64{10, 50, 100}, pal)

	cfg := &legend.Config{
		Position:    model.LegendPositionBottomRight,
		Orientation: model.LegendOrientationVertical,
		LabelSample: legend.LabelSample{
			Lines: []string{"file-name", "file-size", "file-type"},
		},
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
	g.Expect(mb.Calls[1].Method).To(Equal("DrawRectangle"))
}

func TestRenderInto_CircleLabelSample_RendersDiscBeforeEntryHeading(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cv := canvas.NewCanvas(800, 600)
	pal := palette.GetPalette(palette.Temperature)
	fillInk := inks.NumericInk("file-size", []float64{10, 50, 100}, pal)
	cfg := &legend.Config{
		Position:    model.LegendPositionBottomRight,
		Orientation: model.LegendOrientationVertical,
		LabelSample: legend.LabelSample{
			Shape: legend.LabelSampleCircle,
			Lines: []string{"sample-one", "sample-two"},
		},
		Entries: []legend.Entry{
			{Role: legend.RoleFill, MetricName: "file-size", Ink: fillInk},
		},
	}

	legend.RenderInto(cv, cfg)

	mb := mock.NewBackend()
	g.Expect(cv.RenderTo(mb)).To(Succeed())

	discIndex := -1
	sampleLineIndexes := make([]int, 0, 2)
	entryHeadingIndex := -1

	for i, call := range mb.Calls {
		switch {
		case call.Method == "DrawDisc":
			discIndex = i
		case call.Method == "DrawText" && (call.Text == "sample-one" || call.Text == "sample-two"):
			sampleLineIndexes = append(sampleLineIndexes, i)
		case call.Method == "DrawText" && call.Text == "Fill":
			entryHeadingIndex = i
		default:
			continue
		}
	}

	g.Expect(discIndex).To(BeNumerically(">=", 0))
	g.Expect(sampleLineIndexes).To(HaveLen(2))

	for _, sampleLineIndex := range sampleLineIndexes {
		g.Expect(sampleLineIndex).To(BeNumerically(">", discIndex))
		g.Expect(sampleLineIndex).To(BeNumerically("<", entryHeadingIndex))
	}
}

func TestRenderInto_ConstrainedCircleSampleScalesWithinDrawingBounds(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	const (
		width       = 280.0
		drawingMinY = 40.0
		drawingMaxY = 160.0
	)

	sampleLines := []string{
		"directory-name",
		"source-file-name",
		"relative-path",
		"language",
		"permissions",
		"last-change",
		"modified-date",
		"owner",
	}

	cv := canvas.NewCanvas(int(width), 200)
	cv.SetDrawingBounds(int(drawingMinY), int(drawingMaxY))

	pal := palette.GetPalette(palette.Temperature)
	fillInk := inks.NumericInk("file-size", []float64{10, 50, 100}, pal)
	cfg := &legend.Config{
		Position:    model.LegendPositionBottomCenter,
		Orientation: model.LegendOrientationVertical,
		LabelSample: legend.LabelSample{
			Shape: legend.LabelSampleCircle,
			Lines: sampleLines,
		},
		Entries: []legend.Entry{
			{Role: legend.RoleFill, MetricName: "file-size", Ink: fillInk},
			{Role: legend.RoleBorder, MetricName: "line-count", Ink: fillInk},
		},
	}

	legend.RenderInto(cv, cfg)

	mb := mock.NewBackend()
	g.Expect(cv.RenderTo(mb)).To(Succeed())

	background, disc, sampleTexts := collectConstrainedCircleSampleCalls(mb.Calls, sampleLines)

	if background == nil || disc == nil {
		t.Fatal("expected legend background and sample disc")
	}

	if len(sampleTexts) != len(sampleLines) {
		t.Fatal("expected all label sample lines")
	}

	g.Expect(sampleTexts[0].FontSize).To(BeNumerically("<", model.LegendFontSize))

	g.Expect(background.Pos.X).To(BeNumerically(">=", 0))
	g.Expect(background.Pos.Y).To(BeNumerically(">=", drawingMinY))
	g.Expect(background.Pos.X + background.Size.Width).To(BeNumerically("<=", width))
	g.Expect(background.Pos.Y + background.Size.Height).To(BeNumerically("<=", drawingMaxY))

	scale := sampleTexts[0].FontSize / model.LegendFontSize
	sampleSide := scale * (float64(len(sampleLines))*model.LegendLineHeight + 2*model.LabelGap)
	g.Expect(disc.Pos.X - sampleSide/2).To(BeNumerically(">=", 0))
	g.Expect(disc.Pos.Y - sampleSide/2).To(BeNumerically(">=", drawingMinY))
	g.Expect(disc.Pos.X + sampleSide/2).To(BeNumerically("<=", width))
	g.Expect(disc.Pos.Y + sampleSide/2).To(BeNumerically("<=", drawingMaxY))

	for _, sampleText := range sampleTexts {
		g.Expect(sampleText.Pos.X).To(BeNumerically(">=", 0))
		g.Expect(sampleText.Pos.X).To(BeNumerically("<=", width))
		g.Expect(sampleText.Pos.Y).To(BeNumerically(">=", drawingMinY))
		g.Expect(sampleText.Pos.Y).To(BeNumerically("<=", drawingMaxY))
	}
}

func TestRenderInto_ConstrainedCircleSamplePreservesExplicitVerticalOrientation(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cv := canvas.NewCanvas(280, 200)
	cv.SetDrawingBounds(40, 160)

	fillInk := inks.NumericInk("file-size", []float64{10, 50, 100}, palette.GetPalette(palette.Temperature))
	cfg := &legend.Config{
		Position:    model.LegendPositionBottomCenter,
		Orientation: model.LegendOrientationVertical,
		LabelSample: legend.LabelSample{
			Shape: legend.LabelSampleCircle,
			Lines: []string{
				"directory-name", "source-file-name", "relative-path", "language",
				"permissions", "last-change", "modified-date", "owner",
			},
		},
		Entries: []legend.Entry{
			{Role: legend.RoleFill, MetricName: "file-size", Ink: fillInk},
			{Role: legend.RoleBorder, MetricName: "line-count", Ink: fillInk},
		},
	}

	legend.RenderInto(cv, cfg)

	mb := mock.NewBackend()
	g.Expect(cv.RenderTo(mb)).To(Succeed())

	var fill, border *mock.Call

	for i := range mb.Calls {
		call := &mb.Calls[i]
		switch call.Text {
		case "Fill":
			fill = call
		case "Border":
			border = call
		default:
			continue
		}
	}

	if fill == nil || border == nil {
		t.Fatal("expected fill and border headings")
	}

	g.Expect(fill.Pos.X).To(BeNumerically("==", border.Pos.X))
	g.Expect(fill.Pos.Y).To(BeNumerically("<", border.Pos.Y))
}

func collectConstrainedCircleSampleCalls(
	calls []mock.Call,
	sampleLines []string,
) (background *mock.Call, disc *mock.Call, sampleTexts []*mock.Call) {
	for i := range calls {
		call := &calls[i]
		switch {
		case call.Method == "DrawRectangle" && background == nil:
			background = call
		case call.Method == "DrawDisc":
			disc = call
		case call.Method == "DrawText" && slices.Contains(sampleLines, call.Text):
			sampleTexts = append(sampleTexts, call)
		default:
			continue
		}
	}

	return background, disc, sampleTexts
}
