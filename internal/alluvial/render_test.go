package alluvial_test

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/alluvial"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/mock"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/textlayout"
	"github.com/theunrepentantgeek/code-visualizer/internal/config"
	"github.com/theunrepentantgeek/code-visualizer/internal/geometry"
	"github.com/theunrepentantgeek/code-visualizer/internal/inks"
	"github.com/theunrepentantgeek/code-visualizer/internal/legend"
	"github.com/theunrepentantgeek/code-visualizer/internal/palette"
	"github.com/theunrepentantgeek/code-visualizer/internal/stages"
	"github.com/theunrepentantgeek/code-visualizer/internal/viz"
)

func TestRenderToCanvas_AddsFilledPathForEachValidFlow(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cv := alluvial.RenderToCanvas(alluvial.Layout{
		Flows: []alluvial.Flow{
			{
				Path: "continuing", FillValue: -1,
				FromX: 20, ToX: 180, FromTop: 10, FromBottom: 40, ToTop: 20, ToBottom: 70,
			},
			{
				Path: "introduced", FillValue: 1,
				FromX: 20, ToX: 180, FromTop: 60, FromBottom: 60, ToTop: 50, ToBottom: 80,
			},
			{Path: "empty", FromX: 20, ToX: 180, FromTop: 90, FromBottom: 90, ToTop: 90, ToBottom: 90},
		},
	}, 200, 100, inks.NumericInk("fill", []float64{-1, 1}, palette.GetPalette(palette.Neutral)), "")
	backend := mock.NewBackend()

	g.Expect(cv.RenderTo(backend)).To(Succeed())

	filledPaths := make([]mock.Call, 0)

	for _, call := range backend.Calls {
		if call.Method == "DrawFilledPath" {
			filledPaths = append(filledPaths, call)
		}
	}

	g.Expect(filledPaths).To(HaveLen(2))
	firstPath := filledPaths[0]
	g.Expect(firstPath.Loops).To(HaveLen(1))
	g.Expect(len(firstPath.Loops[0])).To(BeNumerically(">", 4))
	g.Expect(firstPath.Loops[0][0]).To(Equal(geometry.Point{X: 20, Y: 10}))
	g.Expect(firstPath.Loops[0][len(firstPath.Loops[0])-1]).To(Equal(geometry.Point{X: 20, Y: 40}))
	g.Expect(filledPaths[1].Loops[0][0]).To(Equal(filledPaths[1].Loops[0][len(filledPaths[1].Loops[0])-1]))
	g.Expect(filledPaths[0].Fill).NotTo(Equal(filledPaths[1].Fill))
}

func TestRenderToCanvas_AddsExplicitFillValueToBandLabel(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	cv := alluvial.RenderToCanvas(alluvial.Layout{
		Columns: []alluvial.ColumnLayout{{
			X: 100,
			Bands: []alluvial.Band{{
				Path: "api", Top: 10, Bottom: 90, Width: 12, FillValue: -3,
			}},
		}},
	}, 200, 100, inks.NumericInk("fill", []float64{-3}, palette.GetPalette(palette.Neutral)), "fill.delta")
	backend := mock.NewBackend()

	g.Expect(cv.RenderTo(backend)).To(Succeed())

	var labels []string

	for _, call := range backend.Calls {
		if call.Method == "DrawText" {
			labels = append(labels, call.Text)
		}
	}

	g.Expect(labels).To(ContainElements("api", "12", "-3"))
}

func TestRenderToCanvas_UsesContrastingInkForBandLabels(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	fillInk := inks.NumericInk("fill", []float64{0, 100}, palette.GetPalette(palette.Neutral))
	cv := alluvial.RenderToCanvas(alluvial.Layout{
		Columns: []alluvial.ColumnLayout{{
			X: 100,
			Bands: []alluvial.Band{{
				Path: "api", Top: 10, Bottom: 90, Width: 12, FillValue: 0,
			}},
		}},
	}, 200, 100, fillInk, "")
	backend := mock.NewBackend()

	g.Expect(cv.RenderTo(backend)).To(Succeed())

	for _, call := range backend.Calls {
		if call.Method == "DrawText" && call.Text == "api" {
			g.Expect(call.Fill).To(Equal(palette.White))

			return
		}
	}

	t.Fatal("expected api band label")
}

func TestRenderToCanvas_ExtendsEdgeBandsBehindCenteredLabels(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)

	const (
		leftX  = 20.0
		rightX = 180.0
	)

	cv := alluvial.RenderToCanvas(alluvial.Layout{
		Columns: []alluvial.ColumnLayout{
			{
				X: leftX,
				Bands: []alluvial.Band{
					{Path: "long-left-label", Top: 10, Bottom: 70, Width: 12},
					{Path: "x", Top: 80, Bottom: 140, Width: 8},
				},
			},
			{
				X: 100,
				Bands: []alluvial.Band{
					{Path: "middle", Top: 10, Bottom: 70, Width: 10},
				},
			},
			{
				X: rightX,
				Bands: []alluvial.Band{
					{Path: "long-right-label", Top: 10, Bottom: 70, Width: 14},
					{Path: "y", Top: 80, Bottom: 140, Width: 9},
				},
			},
		},
		Flows: []alluvial.Flow{
			{
				Path: "long-left-label", FromX: leftX, ToX: 100,
				FromTop: 10, FromBottom: 70, ToTop: 10, ToBottom: 70,
			},
			{
				Path: "long-right-label", FromX: 100, ToX: rightX,
				FromTop: 80, FromBottom: 140, ToTop: 80, ToBottom: 140,
			},
		},
	}, 200, 150, inks.NumericInk("fill", []float64{0}, palette.GetPalette(palette.Neutral)), "")
	backend := mock.NewBackend()

	g.Expect(cv.RenderTo(backend)).To(Succeed())

	var leftLabelWidth, rightLabelWidth float64

	for _, call := range backend.Calls {
		if call.Method != "DrawText" {
			continue
		}

		width, _ := textlayout.MeasureString(call.Text, call.FontSize)
		switch call.Text {
		case "long-left-label", "x", "12", "8":
			g.Expect(call.Pos.X).To(BeNumerically("==", leftX))

			leftLabelWidth = max(leftLabelWidth, width)
		case "middle", "10":
			g.Expect(call.Pos.X).To(BeNumerically("==", 100))
		case "long-right-label", "y", "14", "9":
			g.Expect(call.Pos.X).To(BeNumerically("==", rightX))

			rightLabelWidth = max(rightLabelWidth, width)
		default:
			continue
		}
	}

	var paths []mock.Call

	for _, call := range backend.Calls {
		if call.Method == "DrawFilledPath" {
			paths = append(paths, call)
		}
	}

	g.Expect(paths).To(HaveLen(6))

	if len(paths) != 6 {
		t.Fatalf("expected 6 filled paths, got %d", len(paths))
	}

	minimumX, maximumX := pathXBounds(paths)
	g.Expect(minimumX).To(BeNumerically("<", leftX-leftLabelWidth/2))
	g.Expect(maximumX).To(BeNumerically(">", rightX+rightLabelWidth/2))

	g.Expect(paths[4].Loops[0][0].X).To(BeNumerically("==", leftX))
	g.Expect(paths[5].Loops[0][0].X).To(BeNumerically("==", 100))
}

func pathXBounds(paths []mock.Call) (minimum, maximum float64) {
	minimum = paths[0].Loops[0][0].X
	maximum = minimum

	for _, path := range paths {
		for _, point := range path.Loops[0] {
			minimum = min(minimum, point.X)
			maximum = max(maximum, point.X)
		}
	}

	return minimum, maximum
}

func TestBuildLegendStage_UsesFillMetricAndNumericSplits(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	state := &alluvial.State{
		WidthMetric: "file-lines.sum",
		Fill: alluvial.BandFill{
			Encoding: viz.ColourEncoding{Metric: "file-lines.sum", Palette: palette.Neutral},
			Label:    "file-lines.delta",
			Explicit: true,
			Delta:    true,
		},
		Data: alluvial.Data{Columns: []alluvial.Column{
			{Values: []alluvial.Value{{Path: "api", Width: 10}, {Path: "docs", Width: 5}}},
		}, FillValues: map[string]float64{"api": 5, "docs": -2}},
	}

	g.Expect(alluvial.BuildLegendStage(
		&stages.CommonState{RootConfig: config.New()},
		state,
	)).To(Succeed())

	g.Expect(state.Legend.Entries).To(HaveLen(2))
	g.Expect(state.Legend.Entries[0].MetricName).To(Equal("file-lines.delta"))
	g.Expect(state.Legend.Entries[1].MetricName).To(Equal("file-lines.sum"))
	g.Expect(state.Legend.LabelSample).To(Equal(legend.LabelSample{
		Shape: legend.LabelSampleSquare,
		Lines: []string{"Directory", "file-lines.sum", "file-lines.delta"},
	}))
	middle := state.Fill.Ink.Dip(inks.MeasureValue(0))
	g.Expect(middle).To(Equal(palette.GetPalette(palette.Neutral).Colours[4]))
}

func TestLayoutStage_ReservesSpaceForRightLegend(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	legendConfig := &legend.Config{
		Position:    canvasmodel.LegendPositionCenterRight,
		Orientation: canvasmodel.LegendOrientationVertical,
		Entries: []legend.Entry{{
			Role:       legend.RoleFill,
			MetricName: "api",
			Ink:        inks.FixedInk(color.RGBA{R: 0x44, G: 0x88, B: 0xcc, A: 0xff}),
		}},
	}
	common := &stages.CommonState{Width: 1200, Height: 800}
	stages.InitDrawingBounds(common) //nolint:errcheck // always succeeds

	state := &alluvial.State{
		Data: alluvial.Data{Columns: []alluvial.Column{
			{Reference: "before", Values: []alluvial.Value{{Path: "api", Width: 10}}},
			{Reference: "after", Values: []alluvial.Value{{Path: "api", Width: 20}}},
		}},
		Legend: legendConfig,
	}

	g.Expect(alluvial.LayoutStage(common, state)).To(Succeed())

	reservation := legend.ReserveLayout(legendConfig, common.Width, common.Height)
	g.Expect(state.Layout.Columns[1].X).To(BeNumerically("<=", reservation.Offset.X+float64(reservation.Width)))
}
