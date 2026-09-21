package alluvial_test

import (
	"image/color"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/theunrepentantgeek/code-visualizer/internal/alluvial"
	"github.com/theunrepentantgeek/code-visualizer/internal/canvas/mock"
	canvasmodel "github.com/theunrepentantgeek/code-visualizer/internal/canvas/model"
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

func TestBuildLegendStage_UsesFillMetricAndNumericSplits(t *testing.T) {
	t.Parallel()
	g := NewGomegaWithT(t)
	state := &alluvial.State{
		WidthMetric:   "file-lines.sum",
		Fill:          viz.ColourEncoding{Metric: "file-lines.sum", Palette: palette.Neutral},
		FillLabel:     "file-lines.delta",
		FillSpecified: true,
		FillDelta:     true,
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
	middle := state.FillInk.Dip(inks.MeasureValue(0))
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
