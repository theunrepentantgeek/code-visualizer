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

func TestRenderToCanvas_KeepsEdgeColumnLabelsInsideBands(t *testing.T) {
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
	}, 200, 150, inks.NumericInk("fill", []float64{0}, palette.GetPalette(palette.Neutral)), "")
	backend := mock.NewBackend()

	g.Expect(cv.RenderTo(backend)).To(Succeed())

	var leftPositions, rightPositions []float64

	for _, call := range backend.Calls {
		if call.Method != "DrawText" {
			continue
		}

		width, _ := textlayout.MeasureString(call.Text, call.FontSize)
		switch call.Text {
		case "long-left-label", "x", "12", "8":
			g.Expect(call.Pos.X - width/2).To(BeNumerically(">", leftX))
			leftPositions = append(leftPositions, call.Pos.X)
		case "middle", "10":
			g.Expect(call.Pos.X).To(BeNumerically("==", 100))
		case "long-right-label", "y", "14", "9":
			g.Expect(call.Pos.X + width/2).To(BeNumerically("<", rightX))
			rightPositions = append(rightPositions, call.Pos.X)
		default:
			continue
		}
	}

	g.Expect(leftPositions).To(HaveLen(4))
	g.Expect(rightPositions).To(HaveLen(4))

	if len(leftPositions) == 0 || len(rightPositions) == 0 {
		t.Fatal("expected labels in both edge columns")
	}

	g.Expect(leftPositions).To(HaveEach(Equal(leftPositions[0])))
	g.Expect(rightPositions).To(HaveEach(Equal(rightPositions[0])))
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
